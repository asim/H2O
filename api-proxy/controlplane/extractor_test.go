package controlplane

import (
	"bytes"
	"net/http"
	"testing"
)

type testExtractor struct {
	hob, path, source, host string
	values, headers         map[string]string
}

func (e *testExtractor) Hob() string       { return e.hob }
func (e *testExtractor) Path() string      { return e.path }
func (e *testExtractor) SetHob(hob string) { e.hob = hob }
func (e *testExtractor) Source() string    { return e.source }
func (e *testExtractor) Host() string      { return e.host }
func (e *testExtractor) Value(name string) string {
	if e.values == nil {
		return ""
	}
	return e.values[name]
}
func (e *testExtractor) Header(name string) string {
	if e.headers == nil {
		return ""
	}
	return e.headers[name]
}

// TestHobExtraction sets hob/city param in query string and body then attempts
// extraction.
func TestHobExtraction(t *testing.T) {
	// Set Hob in query
	r, _ := http.NewRequest("GET", "/v1/point/batch?hob=LON", nil)
	e := newExtractor(r)

	if hob := e.Hob(); hob != "LON" {
		t.Errorf("Expected hob LON, got %s", hob)
	}

	// Set City in query
	r, _ = http.NewRequest("GET", "/v1/point/batch?city=LON", nil)
	e = newExtractor(r)

	if city := e.Hob(); city != "LON" {
		t.Errorf("Expected city LON, got %s", city)
	}

	// Set Hob in body
	b := bytes.NewBuffer([]byte("hob=LON"))
	r, _ = http.NewRequest("POST", "/v1/point/batch", b)
	e = newExtractor(r)

	if hob := e.Hob(); hob != "LON" {
		t.Errorf("Expected hob LON, got %s", hob)
	}

	// Set City in body
	b = bytes.NewBuffer([]byte("city=LON"))
	r, _ = http.NewRequest("POST", "/v1/point/batch", b)
	e = newExtractor(r)

	if city := e.Hob(); city != "LON" {
		t.Errorf("Expected city LON, got %s", city)
	}
}

func TestSourceExtraction(t *testing.T) {
	// Set Source to `customer` in hostname
	r, _ := http.NewRequest("GET", "http://api-customer.elasticride.com", nil)
	e := newExtractor(r)

	if src := e.Source(); src != "customer" {
		t.Errorf("Expected source `customer`, got %s", src)
	}

	// Set Source to `driver` in hostname
	r, _ = http.NewRequest("GET", "http://api-driver.elasticride.com", nil)
	e = newExtractor(r)

	if src := e.Source(); src != "driver" {
		t.Errorf("Expected source `driver`, got %s", src)
	}

	// Set Source in header
	r, _ = http.NewRequest("GET", "http://api.elasticride.com", nil)
	r.Header.Set("X-H-Source", "customer")
	e = newExtractor(r)

	if src := e.Source(); src != "customer" {
		t.Errorf("Expected source `customer`, got %s", src)
	}
}
