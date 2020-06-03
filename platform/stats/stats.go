package stats

// Record server stats and publish for consumption by the monitoring-service.

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/nu7hatch/gouuid"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/util"

	pstats "github.com/hailo-platform/H2O/platform/proto/stats"
)

const (
	recordInterval = time.Duration(60) * time.Second
	running        = "RUNNING"
	started        = "STARTED"
	stopped        = "STOPPED"
)

var (
	ServiceName    string
	ServiceVersion uint64
	ServiceType    string
	AzName         string
	InstanceID     string
	hostname       string
	defaultStats   = newStats()
)

type stats struct {
	mtx        sync.RWMutex
	registry   metrics.Registry
	startTime  time.Time
	timers     map[Endpoint]endpointTimers
	recordChan chan *endpointResult
	terminate  chan chan bool
}

type endpointResult struct {
	ep  Endpoint
	err error
	d   time.Duration
}

type endpointTimers struct {
	success metrics.Timer
	errors  metrics.Timer
}

type Endpoint interface {
	GetName() string
	GetMean() int32
	GetUpper95() int32
}

func init() {
	// Set defaults for top level vars
	ServiceName = "com.hailocab.unknown-service"
	ServiceVersion = 20140101000000
	ServiceType = "h2.unknown"

	var err error
	if hostname, err = os.Hostname(); err != nil {
		hostname = "localhost.unknown"
	}

	if uuid, err := uuid.NewV4(); err != nil {
		InstanceID = newUuid()
	} else {
		InstanceID = uuid.String()
	}

	AzName, _ = util.GetAwsAZName()
}

func newUuid() string {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return "unknown"
	}

	uuid[8] = 0x80
	uuid[4] = 0x40

	return hex.EncodeToString(uuid)
}

func newStats() *stats {
	return &stats{
		registry:   metrics.NewRegistry(),
		startTime:  time.Now(),
		timers:     make(map[Endpoint]endpointTimers),
		recordChan: make(chan *endpointResult, 100),
		terminate:  make(chan chan bool),
	}
}

func milli(t float64) float32 {
	return float32(t / 1.e6)
}

func rate(t float64) float32 {
	return float32(t * 100)
}

func captureGCStats(r metrics.Registry, d time.Duration) {
	for {
		var val int64
		var gc debug.GCStats
		debug.ReadGCStats(&gc)

		if len(gc.Pause) > 0 {
			val = gc.Pause[0].Nanoseconds()
		}

		if g := getGauge(r, "local.GCStats.LastGCDuration"); g != nil {
			g.(metrics.Gauge).Update(val)
		}

		time.Sleep(d)
	}
}

func captureRusageStats(r metrics.Registry, d time.Duration) {
	for {
		ru := getRusage()
		time.Sleep(d)
		ru2 := getRusage()

		if ru == nil || ru2 == nil {
			continue
		}

		rVals := map[string]int64{
			"UserTime":   syscall.TimevalToNsec(ru2.Utime) - syscall.TimevalToNsec(ru.Utime),
			"SystemTime": syscall.TimevalToNsec(ru2.Stime) - syscall.TimevalToNsec(ru.Stime),
			"MaxRss":     ru.Maxrss,
			"InBlock":    ru2.Inblock - ru.Inblock,
			"OuBlock":    ru2.Oublock - ru.Oublock,
		}

		for metric, val := range rVals {
			if g := getGauge(r, "rusage."+metric); g != nil {
				g.(metrics.Gauge).Update(val)
			}
		}
	}
}

func endpointSLA(ep Endpoint) *pstats.EndpointSLA {
	return &pstats.EndpointSLA{
		Mean:    proto.Float32(float32(ep.GetMean())),
		Upper95: proto.Float32(float32(ep.GetUpper95())),
	}
}

func endpointStat(t metrics.Timer) *pstats.EndpointStat {
	return &pstats.EndpointStat{
		Rate1:   proto.Float32(rate(t.Rate1())),
		Rate5:   proto.Float32(rate(t.Rate5())),
		Rate15:  proto.Float32(rate(t.Rate15())),
		Mean:    proto.Float32(milli(t.Mean())),
		StdDev:  proto.Float32(milli(t.StdDev())),
		Upper95: proto.Float32(milli(t.Percentile(0.95))),
	}
}

func getCpuUsage(r metrics.Registry, name string) float32 {
	t := float64(recordInterval.Nanoseconds())
	usage := float64(getGaugeVal(r, name))
	return float32((usage / t) * 100.0)
}

func getGauge(r metrics.Registry, name string) metrics.Gauge {
	if v := r.Get(name); v != nil {
		return v.(metrics.Gauge)
	}
	return nil
}

func getGaugeVal(r metrics.Registry, name string) int64 {
	if g := getGauge(r, name); g != nil {
		return g.Value()
	}

	return int64(0)
}

func getRusage() *syscall.Rusage {
	var r syscall.Rusage
	err := syscall.Getrusage(0, &r)
	if err != nil {
		log.Errorf("[Server] error getting rusage %v", err)
		return nil
	}

	return &r
}

func registerRusageStats(r metrics.Registry) {
	for _, metric := range []string{"UserTime", "SystemTime", "MaxRss", "InBlock", "OuBlock"} {
		g := metrics.NewGauge()
		r.Register("rusage."+metric, g)
	}
}

func registerGCStats(r metrics.Registry) {
	g := metrics.NewGauge()
	r.Register("local.GCStats.LastGCDuration", g)
}

func (s *stats) monitor() {
	ticker := time.NewTicker(recordInterval)

	// Tell the monitoring service we have started.
	s.publish(started)

	for {
		select {
		case <-ticker.C:
			s.publish(running)
		case es := <-s.recordChan:
			s.updateTimer(es)
		case done := <-s.terminate:
			s.publish(stopped)
			done <- true
			break
		}
	}
}

// publish
func (s *stats) publish(status string) {
	if err := client.Pub("com.hailocab.monitor.stats", s.get(status)); err != nil {
		log.Errorf("[Server] Failed to publish service monitoring stats: %v", err)
	}
}

// get returns a snapshot of platform stats
func (s *stats) get(status string) *pstats.PlatformStats {
	rusageStats := s.rusage()
	runtimeStats := s.runtime()
	endpointStats := s.endpoints()
	return &pstats.PlatformStats{
		ServiceName:    proto.String(ServiceName),
		ServiceVersion: proto.Uint64(ServiceVersion),
		ServiceType:    proto.String(ServiceType),
		AzName:         proto.String(AzName),
		Hostname:       proto.String(hostname),
		InstanceId:     proto.String(InstanceID),
		Status:         proto.String(status),
		Timestamp:      proto.Int64(time.Now().Unix()),
		Uptime:         proto.Int64(int64(time.Since(s.startTime).Seconds())),
		Rusage:         rusageStats,
		Runtime:        runtimeStats,
		Endpoints:      endpointStats,
	}
}

func (s *stats) start() {
	// Register GC stats.
	registerGCStats(s.registry)
	go captureGCStats(s.registry, recordInterval)

	// Register rusage stats.
	registerRusageStats(s.registry)
	go captureRusageStats(s.registry, recordInterval)

	// Register debug stats.
	metrics.RegisterDebugGCStats(s.registry)
	go metrics.CaptureDebugGCStats(s.registry, recordInterval)

	// Register runtime stats.
	metrics.RegisterRuntimeMemStats(s.registry)
	go metrics.CaptureRuntimeMemStats(s.registry, recordInterval)

	// Start monitoring
	go s.monitor()
}

func (s *stats) stop() {
	done := make(chan bool)
	s.terminate <- done
	<-done
}

func (s *stats) record(ep Endpoint, err error, d time.Duration) {
	es := &endpointResult{
		ep:  ep,
		err: err,
		d:   d,
	}
	s.recordChan <- es
}

func (s *stats) updateTimer(es *endpointResult) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	timers, ok := s.timers[es.ep]
	if !ok {
		log.Errorf("[Server] Missing timer metric for %v", es.ep.GetName())
		return
	}

	if es.err == nil {
		timers.success.Update(es.d)
	} else {
		timers.errors.Update(es.d)
	}
}

func (s *stats) registerEndpoint(ep Endpoint) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if _, ok := s.timers[ep]; ok {
		return
	}

	sTimer := metrics.NewTimer()
	eTimer := metrics.NewTimer()

	s.registry.Register(ep.GetName()+".success", sTimer)
	s.registry.Register(ep.GetName()+".error", eTimer)

	s.timers[ep] = endpointTimers{
		success: sTimer,
		errors:  eTimer,
	}
}

func (s *stats) registerEndpoints(eps []Endpoint) {
	for _, ep := range eps {
		s.registerEndpoint(ep)
	}
}

func (s *stats) runtime() *pstats.RuntimeStats {
	return &pstats.RuntimeStats{
		HeapInUse:      proto.Uint64(uint64(getGaugeVal(s.registry, "runtime.MemStats.HeapInuse"))),
		HeapTotal:      proto.Uint64(uint64(getGaugeVal(s.registry, "runtime.MemStats.TotalAlloc"))),
		HeapReleased:   proto.Uint64(uint64(getGaugeVal(s.registry, "runtime.MemStats.HeapReleased"))),
		LastGCDuration: proto.Uint64(uint64(getGaugeVal(s.registry, "local.GCStats.LastGCDuration"))),
		NumGC:          proto.Uint32(uint32(getGaugeVal(s.registry, "debug.GCStats.NumGC"))),
		NumGoRoutines:  proto.Uint32(uint32(getGaugeVal(s.registry, "runtime.NumGoroutine"))),
	}
}

func (s *stats) rusage() *pstats.RusageStats {
	return &pstats.RusageStats{
		UserTime:   proto.Float32(getCpuUsage(s.registry, "rusage.UserTime")),
		SystemTime: proto.Float32(getCpuUsage(s.registry, "rusage.SystemTime")),
		MaxRss:     proto.Int64(getGaugeVal(s.registry, "rusage.MaxRss")),
		InBlock:    proto.Int64(getGaugeVal(s.registry, "rusage.InBlock")),
		OuBlock:    proto.Int64(getGaugeVal(s.registry, "rusage.OuBlock")),
	}
}

func (s *stats) endpoints() []*pstats.EndpointStats {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	var eps []*pstats.EndpointStats

	for ep, timers := range s.timers {
		eps = append(eps, s.endpoint(ep, timers))
	}

	return eps
}

func (s *stats) endpoint(ep Endpoint, t endpointTimers) *pstats.EndpointStats {
	return &pstats.EndpointStats{
		EndpointName: proto.String(ep.GetName()),
		Sla:          endpointSLA(ep),
		Success:      endpointStat(t.success),
		Error:        endpointStat(t.errors),
	}
}

func Get() *pstats.PlatformStats {
	return defaultStats.get(running)
}

func GetEndpoint(ep Endpoint) *pstats.EndpointStats {
	defaultStats.mtx.RLock()
	defer defaultStats.mtx.RUnlock()

	timers, ok := defaultStats.timers[ep]
	if !ok {
		return nil
	}

	return defaultStats.endpoint(ep, timers)
}

func Record(ep Endpoint, err error, d time.Duration) {
	defaultStats.record(ep, err, d)
}

func RegisterEndpoint(ep Endpoint) {
	defaultStats.registerEndpoint(ep)
}

func RegisterEndpoints(eps []Endpoint) {
	defaultStats.registerEndpoints(eps)
}

func Start() {
	defaultStats.start()
}

func Stop() {
	defaultStats.stop()
}
