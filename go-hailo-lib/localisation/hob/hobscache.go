package hob

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/jpillora/backoff"
	"github.com/stretchr/testify/mock"

	"github.com/HailoOSS/platform/multiclient"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/go-service-layer/config"
)

var (
	memoryCacheImpl map[string]*HobData = make(map[string]*HobData) // map[LON] to *HobData of London
)

type HobsCache interface {
	ReadHob(hob string) *Hob
	ReadServiceTypes(hob string) ServiceTypes
}

type HobData struct {
	Hob              *Hob         `json:"hob"`
	HobHash          string       `json:"hobHash"`
	ServiceTypes     ServiceTypes `json:"serviceTypes"`
	ServiceTypesHash string       `json:"serviceTypesHash"`
}

type MockHobsCache struct {
	mock.Mock
}

func (c *MockHobsCache) ReadHob(hob string) *Hob {
	ret := c.Mock.Called(hob)

	return ret.Get(0).(*Hob)
}

func (c *MockHobsCache) ReadServiceTypes(hob string) ServiceTypes {
	ret := c.Mock.Called(hob)

	return ret.Get(0).(ServiceTypes)
}

type cache struct {
	initRefresher sync.Once
	mtx           sync.RWMutex
	rnd           *rand.Rand
	scopeFrom     multiclient.Scoper
	refresh       chan string
	backoff       *backoff.Backoff
}

func MemoryHobsCache() HobsCache {
	return &cache{
		rnd:       rand.New(rand.NewSource(time.Now().UTC().UnixNano())),
		scopeFrom: server.Scoper(),
		refresh:   make(chan string, 10),
		backoff: &backoff.Backoff{
			Min:    failedMinLoadRetry,
			Max:    failedMaxLoadRetry,
			Factor: failedLoadRetryFactor,
			Jitter: true,
		},
	}
}

func (c *cache) ReadHob(hob string) *Hob {
	log.Debugf("cache.ReadHob for hob:%v", hob)
	hobData := c.readHobDataFromCache(hob)
	if hobData != nil {
		return hobData.Hob
	} // else
	return nil
}

func (c *cache) ReadServiceTypes(hob string) ServiceTypes {
	log.Debugf("cache.ReadServiceTypes for hob:%v", hob)
	hobData := c.readHobDataFromCache(hob)
	if hobData != nil {
		return hobData.ServiceTypes
	} // else
	return nil
}

func (c *cache) readHobDataFromCache(hob string) *HobData {
	// Read the hobData from memcache
	c.mtx.RLock()
	readHob, ok := memoryCacheImpl[hob]
	c.mtx.RUnlock()
	if ok && readHob != nil {
		return readHob
	} // else

	c.mtx.Lock()
	readHob, ok = memoryCacheImpl[hob] // double check the values after acquiring the lock
	if !ok {
		memoryCacheImpl[hob] = nil
	}
	c.mtx.Unlock()

	if ok && readHob != nil {
		return readHob
	}
	// we added a new hob that is not cached, run the multiconfig load of all the known the hobs
	c.initRefresher.Do(c.scheduleReloadPeriodically)
	go func() {
		if len(c.refresh) == 0 {
			c.refresh <- hob
		}
	}()
	return nil
}

func (c *cache) scheduleReloadPeriodically() {
	go c.reloadPeriodically()
}

func (c *cache) reloadPeriodically() {
	ch := config.SubscribeChanges()
	log.Infof("Scheduling reload")
	tick := time.NewTicker(time.Duration(int64(forceReloadPeriod) + int64(c.jitter(forceReloadJitter))))
	for {
		select {
		case <-tick.C:
			log.Debugf("Config reload triggered by timer")
		case <-ch:
			log.Debugf("Config reload triggered by config notification")
			// add some jitter
			time.Sleep(c.jitter(triggerReloadJitter))
		case hob, ok := <-c.refresh:
			log.Debugf("Config reload triggered by query for new hob:%s ok:%v", hob, ok)
		}
		knownHobs, hobHashes, serviceTypesHashes := c.createMultiConfigParams()
		if len(knownHobs) == 0 {
			log.Warnf("got no hobs in knownHobs list")
			continue
		}
		// block until we have loaded
		for {
			if err := c.readMulticonfig(knownHobs, hobHashes, serviceTypesHashes); err != nil {
				if c.backoff.Attempt() > reloadMaxRetryAttempts {
					break
				}
				time.Sleep(c.backoff.Duration())
				continue
			}
			break
		}
		c.backoff.Reset()
	}
}

func (c *cache) createMultiConfigParams() ([]string, []string, []string) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	knownHobsCount := len(memoryCacheImpl)
	hobs := make([]string, knownHobsCount)
	hobHashes := make([]string, knownHobsCount)
	serviceTypesHashes := make([]string, knownHobsCount)
	i := 0
	for hob, hobData := range memoryCacheImpl {
		hobHash := ""
		serviceTypesHash := ""
		if hobData != nil {
			hobHash = hobData.HobHash
			serviceTypesHash = hobData.ServiceTypesHash
		}
		hobs[i] = hob
		hobHashes[i] = hobHash
		serviceTypesHashes[i] = serviceTypesHash
		i++
	}
	return hobs, hobHashes, serviceTypesHashes
}

func (c *cache) jitter(j time.Duration) time.Duration {
	return time.Duration(int64(c.rnd.Float64() * float64(j)))
}
func (c *cache) readMulticonfig(hobs, hobHashes, serviceTypesHashes []string) error {
	log.Debugf("running hobservice.multiconfig with hobs:%v, hobHashes:%v, serviceTypeHashes:%v", hobs, hobHashes, serviceTypesHashes)
	newHobs, newHobsConfigs, newHobsHashes, newServiceTypesConfigs, newServiceTypesHashes, err := h2HobService.Multiconfig(hobs, hobHashes, serviceTypesHashes)
	if err != nil {
		return err
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	for i, hob := range newHobs {
		hData := memoryCacheImpl[hob]
		if hData == nil {
			hData = &HobData{}
		}

		if len(newHobsConfigs[i]) > 0 {
			h := &Hob{}
			err := json.Unmarshal([]byte(newHobsConfigs[i]), h)
			if err != nil {
				log.Errorf("error unmarshalling hob:%v err:%v", hob, err)
				return err
			}
			hData.Hob = h
			hData.HobHash = newHobsHashes[i]
		}

		if len(newServiceTypesConfigs[i]) > 0 {
			serviceTypes := make(ServiceTypes)
			err := json.Unmarshal([]byte(newServiceTypesConfigs[i]), &serviceTypes)
			if err != nil {
				log.Errorf("error unmarshalling serviceTypes:%v serviceTypes:%v\n trying to unmarshal:%v", hob, err, newServiceTypesConfigs[i])
				return err
			}
			hData.ServiceTypes = serviceTypes
			hData.ServiceTypesHash = newServiceTypesHashes[i]
		}
		memoryCacheImpl[hob] = hData
	}
	return nil
}
