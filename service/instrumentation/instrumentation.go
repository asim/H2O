package instrumentation

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	log "github.com/cihub/seelog"
	"github.com/peterbourgon/g2s"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/HailoOSS/service/config"
)

const runtimeSampleInterval = 60 * time.Second
const rusageSampleInterval = 60 * time.Second

type Instrumentation struct {
	namespace          string
	confHash           string
	statsd             g2s.Statter
	registry           metrics.Registry
	savedTimers        map[string]metrics.Timer
	savedCounters      map[string]metrics.Counter
	savedGauges        map[string]metrics.Gauge
	savedFloat64Gauges map[string]metrics.GaugeFloat64
	instRuntime        bool
	instRusage         bool
	launched           time.Time
	mtx                sync.RWMutex
	runtimeOnce        sync.Once
	rusageOnce         sync.Once
}

// New will mint a new Instrumentation - getting statsd connection details from the config service and then looking out
// for any changes.
func New() *Instrumentation {
	ch := config.SubscribeChanges()
	addr := loadStatsdAddr()
	inst := &Instrumentation{
		namespace:          "default",
		confHash:           addr,
		statsd:             loadStatsd(addr),
		registry:           metrics.NewRegistry(),
		savedTimers:        make(map[string]metrics.Timer),
		savedCounters:      make(map[string]metrics.Counter),
		savedGauges:        make(map[string]metrics.Gauge),
		savedFloat64Gauges: make(map[string]metrics.GaugeFloat64),
		launched:           time.Now(),
		instRuntime:        false,
		instRusage:         false,
	}

	inst.StartRuntime()
	inst.StartRusage()

	// Launch listener for config changes
	go func() {
		for _ = range ch {
			inst.mtx.Lock()
			if addr := loadStatsdAddr(); addr != inst.confHash {
				// @todo close old statsd here -- but no way to do this yet
				inst.statsd = loadStatsd(addr)
			}
			inst.mtx.Unlock()
		}
	}()

	return inst
}

func loadStatsdAddr() string {
	host := config.AtPath("hailo", "service", "instrumentation", "statsd", "host").AsString("localhost")
	port := config.AtPath("hailo", "service", "instrumentation", "statsd", "port").AsInt(8125)
	return fmt.Sprintf("%s:%v", host, port)
}

func loadStatsd(addr string) g2s.Statter {
	disabled := config.AtPath("hailo", "service", "instrumentation", "statsd", "disabled").AsBool()
	if disabled {
		return g2s.Noop()
	}

	s, err := g2s.Dial("udp", addr)
	if err != nil {
		log.Warnf("Error initialising statsd connection to %v", addr)
		return nil
	}

	return s
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

var defaultClient = New()

// SetNamespace defines a namespace for all metrics send to statsd - where this will be prepended to each bucket name
func (i *Instrumentation) SetNamespace(ns string) {
	i.mtx.Lock()
	defer i.mtx.Unlock()
	i.namespace = ns
}

// StartRuntime starts a sampling loop that samples runtime statistics and feeds them into this instrumentation value.
// It spawns a background goroutine.
func (i *Instrumentation) StartRuntime() {
	i.runtimeOnce.Do(func() {
		i.mtx.Lock()
		i.instRuntime = true
		i.mtx.Unlock()

		// Capture runtime stats

		i.SaveTiming("runtime.lastGcDuration")
		i.SaveGauge("runtime.lastGcDuration")
		i.SaveGauge("runtime.uptimeSecs")
		i.SaveGauge("runtime.memInUse")
		i.SaveGauge("runtime.memTotal")
		i.SaveGauge("runtime.heapSys")
		i.SaveGauge("runtime.heapIdle")
		i.SaveGauge("runtime.heapInUse")
		i.SaveGauge("runtime.heapReleased")
		i.SaveGauge("runtime.heapObjects")
		i.SaveGauge("runtime.numGC")
		i.SaveGauge("runtime.numGoRoutines")

		go func() {
			for {
				i.sampleRuntime()
				time.Sleep(runtimeSampleInterval)
			}
		}()
	})
}

// StartRuntime starts a sampling loop that samples runtime statistics and feeds them into this instrumentation value.
// It spawns a background goroutine.
func (i *Instrumentation) StartRusage() {
	i.rusageOnce.Do(func() {
		i.mtx.Lock()
		i.instRusage = true
		i.mtx.Unlock()

		// Capture rusage stats

		i.SaveGauge("rusage.userType")
		i.SaveGauge("rusage.systemTime")
		i.SaveGauge("rusage.maxRss")
		i.SaveGauge("rusage.inBlock")
		i.SaveGauge("rusage.outBlock")

		i.SaveGaugeFloat64("rusage.userTimeUsage")
		i.SaveGaugeFloat64("rusage.systemTimeUsage")
		i.SaveGaugeFloat64("rusage.cpuUsage")

		go func() {
			for {
				ru := getRusage()
				time.Sleep(rusageSampleInterval)
				ru2 := getRusage()

				i.sampleRusage(ru, ru2)
			}
		}()
	})
}

// Counter records an increment or decrement to a counter value for the specified bucket.
// If sampleRate is < 1.0 then we will sample values to send on to statsd appropriately.
func (i *Instrumentation) Counter(sampleRate float32, bucket string, n ...int) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()
	if i.statsd != nil {
		go i.statsd.Counter(sampleRate, fmt.Sprintf("%s.%s", i.namespace, bucket), n...)
	}
	if c, exists := i.savedCounters[bucket]; exists {
		for _, v := range n {
			c.Inc(int64(v))
		}
	}
}

// Timing records a duration to a timer value for the specified bucket.
// If sampleRate is < 1.0 then we will sample values to send on to statsd appropriately.
func (i *Instrumentation) Timing(sampleRate float32, bucket string, d ...time.Duration) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()
	if i.statsd != nil {
		go i.statsd.Timing(sampleRate, fmt.Sprintf("%s.%s", i.namespace, bucket), d...)
	}
	if c, exists := i.savedTimers[bucket]; exists {
		for _, v := range d {
			c.Update(v)
		}
	}
}

// Gauge records a value to the gauge for the specified bucket.
// If sampleRate is < 1.0 then we will sample values to send on to statsd appropriately, but use this functionality with
// care, especially with gauges that may not be updated very often.
func (i *Instrumentation) Gauge(sampleRate float32, bucket string, n ...int) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()
	if i.statsd != nil {
		strs := make([]string, len(n))
		for i, v := range n {
			strs[i] = strconv.Itoa(v)
		}
		go i.statsd.Gauge(sampleRate, fmt.Sprintf("%s.%s", i.namespace, bucket), strs...)
	}
	if g, exists := i.savedGauges[bucket]; exists {
		for _, v := range n {
			g.Update(int64(v))
		}
	}
}

// GaugeFloat32 records a float value to the gauge for the specified bucket.
// If sampleRate is < 1.0 then we will sample values to send on to statsd appropriately, but use this functionality with
// care, especially with gauges that may not be updated very often.
func (i *Instrumentation) GaugeFloat64(sampleRate float32, bucket string, n ...float64) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()
	if i.statsd != nil {
		strs := make([]string, len(n))
		for i, v := range n {
			strs[i] = strconv.FormatFloat(v, 'G', -1, 64)
		}
		go i.statsd.Gauge(sampleRate, fmt.Sprintf("%s.%s", i.namespace, bucket), strs...)
	}
	if g, exists := i.savedFloat64Gauges[bucket]; exists {
		for _, v := range n {
			g.Update(v)
		}
	}
}

// SaveCounter indicates that we want to store an internal representation of the counts in this bucket so that we can
// query it (via GetCounter).
func (i *Instrumentation) SaveCounter(bucket string) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	if _, exists := i.savedCounters[bucket]; exists {
		return fmt.Errorf("Counter bucket %s already registered", bucket)
	}

	c := metrics.NewCounter()
	i.registry.Register(bucket, c)
	i.savedCounters[bucket] = c

	return nil
}

// SaveTiming indicates that we want to store an internal representation of all timings in this bucket so that we can
// query it (via GetTiming).
func (i *Instrumentation) SaveTiming(bucket string) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	if _, exists := i.savedTimers[bucket]; exists {
		return fmt.Errorf("Timer bucket %s already registered", bucket)
	}

	t := metrics.NewTimer()
	i.registry.Register(bucket, t)
	i.savedTimers[bucket] = t

	return nil
}

// SaveGauge indicates that we want to store an internal representation of the value of this gauge so that we can query
// it (via GetGauge).
func (i *Instrumentation) SaveGauge(bucket string) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	if _, exists := i.savedGauges[bucket]; exists {
		return fmt.Errorf("Gauge bucket %s already registered", bucket)
	}

	g := metrics.NewGauge()
	i.registry.Register(bucket, g)
	i.savedGauges[bucket] = g

	return nil
}

// SaveGauge indicates that we want to store an internal representation of the value of this gauge so that we can query
// it (via GetGauge).
func (i *Instrumentation) SaveGaugeFloat64(bucket string) error {
	i.mtx.Lock()
	defer i.mtx.Unlock()

	if _, exists := i.savedGauges[bucket]; exists {
		return fmt.Errorf("Gauge bucket %s already registered", bucket)
	}

	g := metrics.NewGaugeFloat64()
	i.registry.Register(bucket, g)
	i.savedFloat64Gauges[bucket] = g

	return nil
}

// GetCounter yields a Counter that we can query directly to find out how many things have been counted.
func (i *Instrumentation) GetCounter(bucket string) metrics.Counter {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	if c, exists := i.savedCounters[bucket]; exists {
		return c
	}

	return nil
}

// GetTiming yields a Timer that we can query directly to find out how many things have been timed (the rate) plus
// various stats about the aggregate timing like mean, stddev etc.
func (i *Instrumentation) GetTiming(bucket string) metrics.Timer {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	if t, exists := i.savedTimers[bucket]; exists {
		return t
	}

	return nil
}

// GetGauge returns a Gauge that we can query directly to find out the value of a gauge.
func (i *Instrumentation) GetGauge(bucket string) metrics.Gauge {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	if g, exists := i.savedGauges[bucket]; exists {
		return g
	}

	return nil
}

// GetGauge returns a Gauge that we can query directly to find out the value of a gauge.
func (i *Instrumentation) GetGaugeFloat64(bucket string) metrics.GaugeFloat64 {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	if g, exists := i.savedFloat64Gauges[bucket]; exists {
		return g
	}

	return nil
}

func (i *Instrumentation) sampleRuntime() {
	memStats := new(runtime.MemStats)
	runtime.ReadMemStats(memStats)

	i.Timing(1.0, "runtime.lastGcDuration", time.Duration(memStats.PauseNs[(memStats.NumGC%256)])*time.Nanosecond)
	i.Gauge(1.0, "runtime.lastGcDuration", int(memStats.PauseNs[(memStats.NumGC+255)%256]))
	i.Gauge(1.0, "runtime.uptimeSecs", int(time.Since(i.launched).Seconds()))
	i.Gauge(1.0, "runtime.memInUse", int(memStats.Alloc))
	i.Gauge(1.0, "runtime.memTotal", int(memStats.TotalAlloc))
	i.Gauge(1.0, "runtime.heapSys", int(memStats.HeapSys))
	i.Gauge(1.0, "runtime.heapIdle", int(memStats.HeapIdle))
	i.Gauge(1.0, "runtime.heapInUse", int(memStats.HeapAlloc))
	i.Gauge(1.0, "runtime.heapReleased", int(memStats.HeapReleased))
	i.Gauge(1.0, "runtime.heapObjects", int(memStats.HeapObjects))
	i.Gauge(1.0, "runtime.numGC", int(memStats.NumGC))
	i.Gauge(1.0, "runtime.numGoRoutines", runtime.NumGoroutine())
}

func (i *Instrumentation) sampleRusage(ru, ru2 *syscall.Rusage) {
	if ru == nil || ru2 == nil {
		return
	}

	userTime := syscall.TimevalToNsec(ru2.Utime) - syscall.TimevalToNsec(ru.Utime)
	systemTime := syscall.TimevalToNsec(ru2.Stime) - syscall.TimevalToNsec(ru.Stime)

	i.Gauge(1.0, "rusage.userTime", int(userTime))
	i.Gauge(1.0, "rusage.systemTime", int(systemTime))
	i.Gauge(1.0, "rusage.maxRss", int(ru.Maxrss))
	i.Gauge(1.0, "rusage.inBlock", int(ru2.Inblock-ru.Inblock))
	i.Gauge(1.0, "rusage.outBlock", int(ru2.Oublock-ru.Oublock))

	// CPU usage
	userTimePercent := getCPUUsage(userTime, rusageSampleInterval)
	systemTimePercent := getCPUUsage(systemTime, rusageSampleInterval)

	i.GaugeFloat64(1.0, "rusage.userTimeUsage", float64(userTimePercent))
	i.GaugeFloat64(1.0, "rusage.systemTimeUsage", float64(systemTimePercent))
	i.GaugeFloat64(1.0, "rusage.cpuUsage", float64(userTimePercent+systemTimePercent))
}

func getCPUUsage(value int64, interval time.Duration) float32 {
	t := float64(interval.Nanoseconds())
	usage := float64(value)
	return float32((usage / t) * 100.0)
}

// SetNamespace wraps defaultClient.SetNamespace
// The defaultClient will automatically gather and record runtime stats.
func SetNamespace(ns string) {
	defaultClient.SetNamespace(ns)
}

// Counter wraps defaultClient.Counter
func Counter(sampleRate float32, bucket string, n ...int) {
	defaultClient.Counter(sampleRate, bucket, n...)
}

// Timing wraps defaultClient.Timing
func Timing(sampleRate float32, bucket string, d ...time.Duration) {
	defaultClient.Timing(sampleRate, bucket, d...)
}

// Gauge wraps defaultClient.Gauge
func Gauge(sampleRate float32, bucket string, n ...int) {
	defaultClient.Gauge(sampleRate, bucket, n...)
}

// Gauge wraps defaultClient.Gauge
func GaugeFloat64(sampleRate float32, bucket string, n ...float64) {
	defaultClient.GaugeFloat64(sampleRate, bucket, n...)
}

// SaveCounter wraps defaultClient.SaveCounter
func SaveCounter(bucket string) error {
	return defaultClient.SaveCounter(bucket)
}

// SaveTiming wraps defaultClient.SaveTiming
func SaveTiming(bucket string) error {
	return defaultClient.SaveTiming(bucket)
}

// SaveGauge wraps defaultClient.SaveGauge
func SaveGauge(bucket string) error {
	return defaultClient.SaveGauge(bucket)
}

// SaveGauge wraps defaultClient.SaveGauge
func SaveGaugeFloat64(bucket string) error {
	return defaultClient.SaveGaugeFloat64(bucket)
}

// GetCounter wraps defaultClient.GetCounter
func GetCounter(bucket string) metrics.Counter {
	return defaultClient.GetCounter(bucket)
}

// GetTiming wraps defaultClient.GetTiming
func GetTiming(bucket string) metrics.Timer {
	return defaultClient.GetTiming(bucket)
}

// GetGauge wraps defaultClient.GetGauge
func GetGauge(bucket string) metrics.Gauge {
	return defaultClient.GetGauge(bucket)
}

// GetGauge wraps defaultClient.GetGauge
func GetGaugeFloat64(bucket string) metrics.GaugeFloat64 {
	return defaultClient.GetGaugeFloat64(bucket)
}
