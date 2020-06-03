package controlplane

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	"github.com/hailo-platform/H2O/service/config"
)

func TestGetHobMode(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()

	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "LON",
		},
		control: cp,
	}

	hm := r.GetHobMode()
	assert.Equal(t, "h1", hm)

	r = &RuleRouter{
		extractor: &testExtractor{
			hob: "BOS",
		},
		control: cp,
	}
	hm = r.GetHobMode()
	assert.Equal(t, "h2", hm)
}

func BenchmarkGetHobMode(b *testing.B) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "BOS",
		},
		control: cp,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.GetHobMode()
	}
}

func TestParallelGetHobMode(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	// fire off N goroutines to compete against the same controlplane - as is the code
	p := 1000

	cases := []struct {
		hob, mode string
	}{
		{
			hob:  "LON",
			mode: "h1",
		},
		{
			hob:  "MAD",
			mode: "h1",
		},
		{
			hob:  "BOS",
			mode: "h2",
		},
		{
			hob:  "MNC",
			mode: "h2",
		},
	}

	results := make(chan error, p)

	for i := 0; i < p; i++ {
		tc := cases[i%4]

		r := &RuleRouter{
			extractor: &testExtractor{
				hob: tc.hob,
			},
			control: cp,
		}
		go func(r *RuleRouter, expected string) {
			var err error
			for j := 0; j < 10000; j++ {
				if !assert.Equal(t, expected, r.GetHobMode()) {
					break
				}
			}
			results <- err
		}(r, tc.mode)
	}

	for i := 0; i < p; i++ {
		err := <-results
		assert.NoError(t, err, "Error testing HOB mode")
	}
}

func TestGetHobModeWithForced(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "LON",
		},
		control: cp,
	}

	hm := r.GetHobMode()
	assert.Equal(t, "h1", hm)

	// now force this to H2
	r = &RuleRouter{
		extractor: &testExtractor{
			hob:     "LON",
			headers: map[string]string{"X-Hailo-Route": "H2"},
		},
		control: cp,
	}
	hm = r.GetHobMode()
	assert.Equal(t, "h2", hm, "Expecting h2 HOB mode to be forced")

	// repeat the other way round

	r = &RuleRouter{
		extractor: &testExtractor{
			hob: "BOS",
		},
		control: cp,
	}
	hm = r.GetHobMode()
	assert.Equal(t, "h2", hm)

	// now force this to H1
	r = &RuleRouter{
		extractor: &testExtractor{
			hob:     "BOS",
			headers: map[string]string{"X-Hailo-Route": "H1"},
		},
		control: cp,
	}
	hm = r.GetHobMode()
	assert.Equal(t, "h1", hm)
}

func TestRoute(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob:    "LON",
			source: "driver",
			path:   "/v1/driver/index",
		},
		control: cp,
	}

	rule := r.Route()
	if rule == nil {
		t.Error("Expecting a rule matched against LON /v1/driver/index")
	}
	if rule.Action != ActionProxyToH1 {
		t.Errorf("Expecting LON /v1/driver/index to be routed to H1, got %v", rule.Action)
	}

	r = &RuleRouter{
		extractor: &testExtractor{
			hob:    "ATL",
			source: "customer",
			path:   "/v1/customer/neardrivers",
		},
		control: cp,
	}

	rule = r.Route()
	if rule == nil {
		t.Error("Expecting a rule matched against ATL /v1/customer/neardrivers")
	}
	if rule.Action != ActionSendToH2 {
		t.Errorf("Expecting ATL /v1/customer/neardrivers to be routed to H2, got %v", rule.Action)
	}
}

func BenchmarkRoute(b *testing.B) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob:    "ATL",
			source: "customer",
			path:   "/v1/customer/neardrivers",
		},
		control: cp,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Route()
	}
}

func TestParallelRoute(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	// fire off N goroutines to compete against the same controlplane - as is the code
	p := 1000

	cases := []struct {
		hob, source, path string
		expected          Action
	}{
		{
			hob:      "LON",
			source:   "customer",
			path:     "/v1/customer/neardrivers",
			expected: ActionProxyToH1,
		},
		{
			hob:      "ATL",
			source:   "customer",
			path:     "/v1/customer/neardrivers",
			expected: ActionSendToH2,
		},
	}

	results := make(chan error, p)

	for i := 0; i < p; i++ {
		tc := cases[i%2]

		r := &RuleRouter{
			extractor: &testExtractor{
				hob:    tc.hob,
				source: tc.source,
				path:   tc.path,
			},
			control: cp,
		}
		go func(r *RuleRouter, expected Action) {
			var err error
			for j := 0; j < 1000; j++ {
				rule := r.Route()
				if rule.Action != expected {
					err = fmt.Errorf("Expecting action %v; got %v", expected, rule.Action)
					break
				}
			}
			results <- err
		}(r, tc.expected)
	}

	for i := 0; i < p; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Error testing Route(): %v", err)
		}
	}
}

// TestRegionFound tests when we have a region defined for a HOB
func TestRegionFound(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "LON",
		},
		control: cp,
	}
	reg, version := r.Region()
	if reg.Id != "eu-west-1" {
		t.Errorf("Expecting eu-west-1 region; got %v", reg.Id)
	}
	if version != 1404710310 {
		t.Errorf("Expecting version 1404710310; got %v", version)
	}

	r = &RuleRouter{
		extractor: &testExtractor{
			hob: "BOS",
		},
		control: cp,
	}
	reg, version = r.Region()
	if reg.Id != "us-east-1" {
		t.Errorf("Expecting us-east-1 region; got %v", reg.Id)
	}
	if version != 1404710310 {
		t.Errorf("Expecting version 1404710310; got %v", version)
	}
}

// TestRegionDefault tests when there is no specific region defined for a HOB
func TestRegionDefault(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "ZZZ",
		},
		control: cp,
	}
	reg, version := r.Region()
	assert.NotNil(t, reg)

	if reg.Id != "eu-west-1" {
		t.Errorf("Expecting default of eu-west-1 region; got %v", reg.Id)
	}
	if version != 1404710310 {
		t.Errorf("Expecting version 1404710310; got %v", version)
	}
}

// TestRegionDefaultAlwaysSame makes sure we always pick the _same_ default region
// even if config reloads
func TestRegionDefaultAlwaysSame(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	// fire off N goroutines to compete against the same controlplane - as is the code
	p := 1000

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "ZZZ",
		},
		control: cp,
	}

	results := make(chan error, p)

	for i := 0; i < p; i++ {
		go func(r *RuleRouter) {
			var err error
			for j := 0; j < 1000; j++ {
				reg, _ := r.Region()
				if !assert.Equal(t, "eu-west-1", reg.Id, "Unexpected default region") {
					break
				}
			}
			results <- err
		}(r)

		// reload the config too
		loadRules()
	}

	for i := 0; i < p; i++ {
		err := <-results
		assert.NoError(t, err)
	}
}

// TestRegionNoneDefined tests when we have no regions defined at all
func TestRegionNoneDefined(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()

	cp := New()
	cfg := cp.loadedConfig()
	cfg.regions = make(Regions)
	atomic.StorePointer(&cp._loadedConfig, unsafe.Pointer(&cfg))

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "BOS",
		},
		control: cp,
	}
	reg, version := r.Region()

	assert.Nil(t, reg, "Expecting nil region when none defined; got %v", reg)
	assert.Equal(t, int64(1404710310), version)
}

// TestRegionOfflineWithFailoverDefined tests what happens when our primary HOB region
// is defined, but is offline, and there is a failover available
func TestRegionOfflineWithFailoverDefined(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "LON",
		},
		control: cp,
	}
	reg, _ := r.Region()
	if reg.Id != "eu-west-1" {
		t.Errorf("Expecting eu-west-1 region; got %v", reg.Id)
	}

	cp.loadedConfig().regions["eu-west-1"].Status = "OFFLINE"

	reg, _ = r.Region()
	if reg.Id != "us-east-1" {
		t.Errorf("Expecting us-east-1 region (when EU offline); got %v", reg.Id)
	}
}

// TestRegionOfflineWithNoFailover tests what happens when our primary HOB region is
// defined, but is offline, and there is no fallback region
func TestRegionOfflineWithNoFailover(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	cp.loadedConfig().regions["eu-west-1"].Status = "OFFLINE"

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "LON",
		},
		control: cp,
	}

	reg, _ := r.Region()
	if reg.Id != "us-east-1" {
		t.Errorf("Expecting us-east-1 region (when EU offline); got %v", reg.Id)
	}

	// Don't deifne any fail over
	cp.loadedConfig().regions["eu-west-1"].Failover = []string{}

	reg, _ = r.Region()
	if reg.Id != "eu-west-1" {
		t.Errorf("Expecting eu-west-1 region (if no failover defined - even if offline); got %v", reg.Id)
	}
}

func BenchmarkRegion(b *testing.B) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{
			hob: "BOS",
		},
		control: cp,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Region()
	}
}

func TestCorrectRegionWithNoHob(t *testing.T) {
	origConfigBuf := bytes.NewBuffer(config.Raw())
	defer config.Load(origConfigBuf)
	loadRules()
	cp := New()

	r := &RuleRouter{
		extractor: &testExtractor{},
		control:   cp,
	}

	rw := httptest.NewRecorder()
	err, corr, _, version := r.CorrectHostname(rw)

	if err == nil {
		t.Error("When we don't know the HOB, we expect an error")
	}

	if corr {
		t.Error("When we don't know the HOB, we expect the hostname check to have not been possible")
	}
	if version != 0 {
		t.Errorf("Expecting version 0 - since no HOB; got %v", version)
	}
}

func loadRules() {
	rules := `{"controlPlane":{"configVersion":1.40471031e+09,"hobModes":{"BCN":"h1","BOS":"h2","CHI":"h1","DUB":"h1","GWY":"h1","LMK":"h1","LON":"h1","MAD":"h1","MTR":"h1","NYC":"h1","ORK":"h1","OSA":"h1","TOR":"h1","TYO":"h1","WAS":"h1","default":"h2"},"hobRegions":{"ATL":"us-east-1","BCN":"eu-west-1","BOS":"us-east-1","CHI":"us-east-1","DUB":"eu-west-1","LBA":"eu-west-1","LIV":"eu-west-1","LON":"eu-west-1","MAD":"eu-west-1","MNC":"eu-west-1","NYC":"us-east-1","SIN":"us-east-1","TOR":"us-east-1","WAS":"us-east-1"},"regions":{"eu-west-1":{"apps":{"customer":{"api":"api-customer-eu-west-1-live.elasticride.com","hms":"h2o-hms-eu-west-1-live.elasticride.com"},"default":{"api":"api2-eu-west-1-live.elasticride.com","hms":"h2o-hms-eu-west-1-live.elasticride.com"},"driver":{"api":"api-driver-eu-west-1-live.elasticride.com","hms":"h2o-hms-eu-west-1-live.elasticride.com"}},"failover":["us-east-1"],"id":"eu-west-1","status":"ONLINE"},"us-east-1":{"apps":{"customer":{"api":"api-customer-us-east-1-live.elasticride.com","hms":"h2o-hms-us-east-1-live.elasticride.com"},"default":{"api":"api2-us-east-1-live.elasticride.com","hms":"h2o-hms-us-east-1-live.elasticride.com"},"driver":{"api":"api-driver-us-east-1-live.elasticride.com","hms":"h2o-hms-us-east-1-live.elasticride.com"}},"failover":["eu-west-1"],"id":"us-east-1","status":"ONLINE"}},"rules":{"0d9b0195":{"action":2,"match":{"path":"/v1/point","proportion":0.6,"regulatoryArea":"LON,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO"}},"1b057907":{"action":2,"match":{"path":"/v1/heatmap","proportion":1}},"3553e538":{"action":1,"match":{"path":"/v1/quote","proportion":1,"regulatoryArea":"LON,DUB,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO,ORK,GWY,LMK,IRE","source":"customer"}},"3621b972":{"action":2,"match":{"path":"/v1/customer/neardrivers","proportion":1,"source":"customer"}},"37fe2578":{"action":2,"match":{"path":"/v1/order","proportion":1,"source":"customer"}},"47bd8440":{"action":1,"match":{"path":"/v1/order","proportion":1,"regulatoryArea":"LON,DUB,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO,ORK,GWY,LMK,IRE","source":"customer"}},"62016165":{"action":1,"match":{"proportion":1,"source":"customer"}},"63688074":{"action":1,"match":{"path":"/v1/customer/neardrivers","proportion":1,"regulatoryArea":"LON,DUB,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO,ORK,GWY,LMK,IRE","source":"customer"}},"6d773686":{"action":1,"match":{"path":"/v1/features/index","proportion":1,"regulatoryArea":"LON,DUB,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO,ORK,GWY,LMK,IRE","source":"customer"}},"7b671808":{"action":2,"match":{"path":"/v1/gamification","proportion":1}},"84deb6fd":{"action":1,"match":{"path":"/v1/track","proportion":1,"regulatoryArea":"LON,DUB,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO,ORK,GWY,LMK,IRE","source":"customer"}},"8deb5b6f":{"action":3,"match":{"path":"/v2/throttle","proportion":1}},"a6b2f642":{"action":2,"match":{"path":"/v1/experiment","proportion":1}},"b4f3d570":{"action":1,"match":{"path":"/v1/job/hailo/details","proportion":1,"regulatoryArea":"LON,DUB,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO,ORK,GWY,LMK,IRE,BOS","source":"driver"}},"c006bb83":{"action":2,"match":{"path":"/v1/point","proportion":0.2,"regulatoryArea":"DUB,ORK,GWY,LMK,IRE"}},"c1200ba4":{"action":2,"match":{"path":"/v1/quote","proportion":1,"source":"customer"}},"c515b23b":{"action":1,"match":{"proportion":1,"regulatoryArea":"LON,DUB,CHI,NYC,TOR,MTR,MAD,BCN,WAS,OSA,TYO,ORK,GWY,LMK,IRE"}},"d4f54388":{"action":2,"match":{"path":"/v1/features/index","proportion":1,"source":"customer"}},"ecc5696d":{"action":2,"match":{"path":"/v1/experiment","proportion":1,"source":"customer"}},"fd4559e9":{"action":2,"match":{"path":"/v1/track","proportion":1,"source":"customer"}}}}}`
	buf := bytes.NewBufferString(rules)
	config.Load(buf)
}
