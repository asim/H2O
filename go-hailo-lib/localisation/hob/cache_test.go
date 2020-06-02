package hob

import (
	"encoding/json"
	"testing"
	"time"
)

type mockConfigService struct{}

func (m *mockConfigService) ReadHob(hob string) (*Hob, error) {
	return &Hob{
		Code: "MCK",
		Name: "Mock",
	}, nil
}

func (m *mockConfigService) ReadServiceTypes(hob string) (ServiceTypes, error) {
	s := make(ServiceTypes)
	s["a"] = &ServiceType{
		Id:   "a",
		Name: "service type a",
	}
	s["b"] = &ServiceType{
		Id:   "b",
		Name: "service type b",
	}
	return s, nil
}

func (m *mockConfigService) Multiconfig(hobs, hobHashes, serviceTypesHashes []string) (newHobs, newHobsConfigs, newHobsHashes, newServiceTypesConfigs, newServiceTypesHashes []string, err error) {
	time.Sleep(100 * time.Millisecond) // Wait a bit before filling the cache
	h, err := m.ReadHob("MCK")
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	hobBytes, err := json.Marshal(h)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	st, err := m.ReadServiceTypes("MCK")
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	stBytes, err := json.Marshal(st)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return []string{"MCK"}, []string{string(hobBytes)}, []string{""}, []string{string(stBytes)}, []string{""}, nil
}

func setup() {
	h2HobService = &mockConfigService{}
	memoryCacheImpl = make(map[string]*HobData)
}

func TestCache(t *testing.T) {
	setup()
	h, err := h2HobService.ReadHob("MCK")
	s, err := h2HobService.ReadServiceTypes("MCK")
	hobData1 := &HobData{
		Hob:              h,
		HobHash:          "aaa",
		ServiceTypes:     s,
		ServiceTypesHash: "ss",
	}
	b, err := json.Marshal(*hobData1)
	if err != nil {
		t.Errorf("error:%v", err)
	}
	hobData2 := &HobData{}
	err = json.Unmarshal(b, hobData2)
	if err != nil {
		t.Errorf("error:%v", err)
	}

	if hobData1.Hob.Code != hobData2.Hob.Code {
		t.Errorf("marshaling error, expected:%v got:%v", hobData1.Hob, hobData2.Hob)
		return
	}

	if len(hobData1.ServiceTypes) != len(hobData2.ServiceTypes) {
		t.Errorf("marshaling error, expected:%v got:%v", hobData1.ServiceTypes, hobData2.ServiceTypes)
		return
	}

	hob := Cache.ReadHob("MCK")
	if hob != nil {
		t.Errorf("Expected cache miss, but got cache hit")
	}
	st := Cache.ReadServiceTypes("MCK")
	if st != nil {
		t.Errorf("Expected cache miss in serviceTypes, but got cache hit")
	}

	time.Sleep(time.Second) // reload the multiconfig
	hob = Cache.ReadHob("MCK")
	if hob == nil {
		t.Errorf("Expected cache hit, but got cache miss:%v", hob)
	}

	servt := Cache.ReadServiceTypes("MCK")
	if servt == nil {
		t.Errorf("Expected cache hit in service types, but got cache miss:%v", hob)
	}

}

type HobMockNewField struct {
	Code         string    `json:"code" readOnly:"true"`
	Name         string    `json:"name"`
	Status       HobStatus `json:"status" enum:"BETA,ENABLED,DISABLED,HIDDEN"`
	Country      Country   `json:"country"`
	Currency     string    `json:"currency"`
	Language     string    `json:"language"`
	GeoInfo      GeoInfo   `json:"geoInfo"`
	Phone        Phone     `json:"phone"`
	Timezone     string    `json:"timezone"`
	Locale       string    `json:"locale"`
	Misc         Misc      `json:"misc"`
	FastestFirst bool      `json:"fastestFirst"` // Specified whether the "fastest first" feature in the passenger app is enabled
	A_NEW_FIELD  bool      `json:"anewfield"`
}
