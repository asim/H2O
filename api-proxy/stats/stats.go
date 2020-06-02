package stats

import (
	"errors"
	pstats "github.com/HailoOSS/platform/stats"
	"sync"
	"time"
)

type statsManager struct {
	mtx       sync.RWMutex
	endpoints map[string]*endpoint
	metrics   map[string]float32
}

var (
	defaultManager = newStatsManager()
)

func newStatsManager() *statsManager {
	return &statsManager{
		endpoints: make(map[string]*endpoint),
		metrics:   make(map[string]float32),
	}
}

func (s *statsManager) record(ep string, success bool, d time.Duration) {
	epoint, ok := s.endpoints[ep]
	if !ok {
		return
	}

	var err error
	if !success {
		err = errors.New("error")
	}

	pstats.Record(epoint, err, d)
}

func (s *statsManager) register(ep string) {
	if _, ok := s.endpoints[ep]; ok {
		return
	}

	epoint := &endpoint{
		name:    ep,
		mean:    defaultMean,
		upper95: defaultUpper95,
	}

	s.endpoints[ep] = epoint
	pstats.RegisterEndpoint(epoint)
}

func (s *statsManager) rate1(ep string) (float32, float32) {
	epp, ok := s.endpoints[ep]
	if !ok {
		return 0.0, 0.0
	}

	st := pstats.GetEndpoint(epp)
	if st == nil {
		return 0.0, 0.0
	}

	var su, e, sRate, eRate float32

	if succ := st.GetSuccess(); succ != nil {
		su = succ.GetRate1()
	}

	if err := st.GetError(); err != nil {
		e = err.GetRate1()
	}

	if e != 0.0 {
		eRate = e / (su + e)
	}

	if su != 0.0 {
		sRate = su / (e + su)
	}

	return sRate, eRate
}

func (s *statsManager) rate1Total(ep string) float32 {
	epp, ok := s.endpoints[ep]
	if !ok {
		return 0.0
	}

	st := pstats.GetEndpoint(epp)
	if st == nil {
		return 0.0
	}

	var sucRate, errRate float32

	if succ := st.GetSuccess(); succ != nil {
		sucRate = succ.GetRate1()
	}

	if err := st.GetError(); err != nil {
		errRate = err.GetRate1()
	}

	return sucRate + errRate
}

func (s *statsManager) getMetric(key string) float32 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.metrics[key]
}

func (s *statsManager) saveMetric(key string, value float32) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.metrics[key] = value
}

func GetMetric(key string) float32 {
	return defaultManager.getMetric(key)
}

func SaveMetric(key string, value float32) {
	defaultManager.saveMetric(key, value)
}

func Init(name string, version uint64, id string) {
	pstats.ServiceName = name
	pstats.ServiceVersion = version
	pstats.ServiceType = "h2.proxy"
	pstats.InstanceID = id
}

func Rate1(ep string) (float32, float32) {
	return defaultManager.rate1(ep)
}

func Rate1Total(ep string) float32 {
	return defaultManager.rate1Total(ep)
}

func Record(ep string, success bool, d time.Duration) {
	defaultManager.record(ep, success, d)
}

func Register(ep string) {
	defaultManager.register(ep)
}

func Start() {
	go pstats.Start()
}

func Stop() {
	pstats.Stop()
}
