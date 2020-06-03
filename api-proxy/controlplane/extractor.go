package controlplane

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/api-proxy/session"
)

const (
	// Key added by a client (to the query or body parameters) to specify a city
	cityCodeKey = "city"
	hobCodeKey  = "hob"
)

// extractor wraps an HTTP request and can extract vars from it, leaving the original request intact
// for further processing (in terms of body) and also with the ability to tack stuff into the
// request for further processing (eg: city code to query string)
// not thread safe
type extractor struct {
	req *http.Request
	// Contains extracted values from HTTP body or query (in that order – ie. a parameter found in BOTH the body and the
	// query will take the value from the body)
	extractedValues map[string]string
}

// Creates a new extractor (which will load values from the request lazily)
func newExtractor(req *http.Request) *extractor {
	return &extractor{req: req}
}

func (e *extractor) SetHob(code string) {
	q := e.req.URL.Query()
	q.Add(cityCodeKey, code)
	e.req.URL.RawQuery = q.Encode()
	if e.extractedValues == nil {
		e.doExtraction()
	} else {
		e.extractedValues[cityCodeKey] = code
	}
}

func (e *extractor) CityOrHob() string {
	if hob := e.Value(cityCodeKey); len(hob) > 0 {
		return hob
	}

	return e.Value(hobCodeKey)
}

// Hob will extract a city code from the request, looking at either a hostname match against a known list, or a
// query/body parameter match
func (e *extractor) Hob() string {
	if e.req == nil {
		log.Trace("[extractor] req is nil, cannot determine Hob()")
		return ""
	}

	hob := e.CityOrHob()
	if len(hob) != 0 {
		return hob
	}

	if code, ok := hostMap[e.req.Host]; ok {
		// Add to the query string
		log.Tracef("[extractor] HOB match to %s from HTTP Host header: %s", code, e.req.Host)
		e.SetHob(code)

		return code
	}

	log.Tracef("[extractor] Unable to detect hob from request")
	return ""
}

// Value will extract a value from either POST or GET vars, as appropriate
func (e *extractor) Value(name string) string {
	if e.extractedValues == nil {
		e.doExtraction()
	}
	return e.extractedValues[name]
}

// Path returns path of HTTP request
func (e *extractor) Path() string {
	if e.extractedValues == nil {
		e.doExtraction()
	}
	return e.req.URL.Path
}

// Source returns the source of this API request, in terms of "customer" or "driver"
func (e *extractor) Source() string {
	if e.extractedValues == nil {
		e.doExtraction()
	}
	if strings.Contains(e.req.Host, "driver") {
		return "driver"
	} else if strings.Contains(e.req.Host, "customer") {
		return "customer"
	}

	return e.req.Header.Get("X-H-Source")
}

// Host returns the hostname used to make this request
func (e *extractor) Host() string {
	return e.req.Host
}

// Header gets the value of the given HTTP header from the request
func (e *extractor) Header(hdr string) string {
	return e.req.Header.Get(hdr)
}

// doExtraction is invoked lazily when required to parse the HTTP request
func (e *extractor) doExtraction() {
	e.extractedValues = make(map[string]string)
	// We need to backup the request body since req.ParseForm() mutates it and other methods need it intact later

	// Query (GET) parameters first
	for k, vs := range e.req.URL.Query() {
		e.extractedValues[k] = vs[0]
	}

	// Copy the request body, and restore it on exit
	savedRequestBody := e.req.Body
	defer func() {
		e.req.Body = savedRequestBody
	}()

	var err error
	if e.req.Body != nil {
		savedRequestBody, e.req.Body, err = drainBody(e.req.Body)
		if err != nil {
			return
		}
	}

	// Not all clients send the correct mime type. Add it if it's missing, set to application/x-www-form-urlencoded
	if e.req.Method == "POST" || e.req.Method == "PUT" {
		ct, _, err := mime.ParseMediaType(e.req.Header.Get("Content-Type"))
		if ct == "" || err != nil {
			e.req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	err = e.req.ParseForm()
	if err != nil {
		log.Errorf("Error parsing request body: %v", err)
		return
	}

	for k, vs := range e.req.PostForm {
		e.extractedValues[k] = vs[0]
	}

	// session_id extracted in the same way we extract when we decide what to use for auth
	// and overrides everything else
	e.extractedValues["session_id"] = session.SessionId(e.req)
}

// Taken from the stdlib, httputil.DumpRequest
// One of the copies, say from b to r2, could be avoided by using a more
// elaborate trick where the other copy is made during Request/Response.Write.
// This would complicate things too much, given that these functions are for
// debugging only.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, nil, err
	}
	if err = b.Close(); err != nil {
		return nil, nil, err
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewBuffer(buf.Bytes())), nil
}
