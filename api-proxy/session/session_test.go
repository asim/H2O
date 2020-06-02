package session

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sessId = "WW6ey7SFylhtrrn+DCAz/ov2Z0VJ0/4FBCtBE7p0ARaKE8cAnSfks1"

func TestSessionIdExtractionFromQueryString(t *testing.T) {
	data := url.Values{}
	data.Set("session_id", sessId)

	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?session_id=boop"

	req, _ := http.NewRequest("GET", urlStr, nil)
	s := SessionId(req)

	assert.Equal(t, sessId, s, "they should be equal")
}

func TestSessionIdExtractionFromApiTokenQueryString(t *testing.T) {
	data := url.Values{}
	data.Set("api_token", sessId)

	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?api_token=boop"

	req, _ := http.NewRequest("GET", urlStr, nil)
	s := SessionId(req)

	assert.Equal(t, sessId, s, "they should be equal")
}

func TestSessionIdExtractionFromFormData(t *testing.T) {
	var err error

	data := url.Values{}
	data.Set("session_id", sessId)

	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?session_id=boop"

	// Build request, content type must be set for ParseForm to work
	req, _ := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	err = req.ParseForm()
	assert.Nil(t, err, "err should be nil when parsing form")

	s := SessionId(req)

	assert.Equal(t, sessId, s, "they should be equal")
}

func TestSessionIdExtractionFromApiTokenFormData(t *testing.T) {
	var err error

	data := url.Values{}
	data.Set("api_token", sessId)

	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?session_id=boop"

	// Build request, content type must be set for ParseForm to work
	req, _ := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	err = req.ParseForm()
	assert.Nil(t, err, "err should be nil when parsing form")

	s := SessionId(req)

	assert.Equal(t, sessId, s, "they should be equal")
}

func TestSessionIdExtractionFromApiTokenHeader(t *testing.T) {
	data := url.Values{}
	data.Set("shoop", "da whoop")
	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?shoop=da+whoop"

	testcases := []string{
		"X-API-TOKEN",
		"x-api-token",
		"X-Api-Token", // technically correct according to http spec
		"x-ApI-tOkEn", // OmG
	}

	var req *http.Request
	for _, tc := range testcases {
		req, _ = http.NewRequest("GET", urlStr, nil)
		req.Header.Set(tc, sessId)
		s := SessionId(req)
		assert.Equal(t, sessId, s, "they should be equal")
	}
}

func TestSessionIdExtractionFromAuthorizationHeader(t *testing.T) {
	data := url.Values{}
	data.Set("shoop", "da whoop")
	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?shoop=da+whoop"

	req, _ := http.NewRequest("GET", urlStr, nil)

	// Authorization: token WW6ey7SFylhtrrn+DCAz/ov2Z0VJ0/4FBCtBE7p0ARaKE8cAnSfks1...
	req.Header.Set("Authorization", fmt.Sprintf("token %s", sessId))

	s := SessionId(req)
	assert.Equal(t, sessId, s, "they should be equal")
}

func TestSessionIdExtractionFailureFromUnknownAuthorizationScheme(t *testing.T) {
	data := url.Values{}
	data.Set("shoop", "da whoop")
	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?shoop=da+whoop"

	req, _ := http.NewRequest("GET", urlStr, nil)

	// Authorization: laserauth WW6ey7SFylhtrrn+DCAz/ov2Z0VJ0/4FBCtBE7p0ARaKE8cAnSfks1...
	req.Header.Set("Authorization", fmt.Sprintf("laserauth %s", sessId))

	s := SessionId(req)
	assert.Equal(t, "", s, "they should be equal")
}

func TestSessionIdExtractionFromMultipleAuthorizationHeaders(t *testing.T) {
	data := url.Values{}
	data.Set("shoop", "da whoop")
	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?shoop=da+whoop"

	req, _ := http.NewRequest("GET", urlStr, nil)

	// We should pick the first valid known authorization scheme
	req.Header.Set("Authorization", fmt.Sprintf("token %s", sessId))
	req.Header.Add("Authorization", fmt.Sprintf("token %s", "someothertokendwhichshouldbeignoredasitssecond"))

	s := SessionId(req)
	assert.Equal(t, sessId, s, "they should be equal")
}

func TestSessionIdExtractionFromMultipleAndUnknownAuthorizationHeaders(t *testing.T) {
	data := url.Values{}
	data.Set("shoop", "da whoop")
	u, _ := url.ParseRequestURI("http://localhost")
	u.Path = "/"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u) // "http://localhost/?shoop=da+whoop"

	req, _ := http.NewRequest("GET", urlStr, nil)

	// Authorization: laserauth WW6ey7SFylhtrrn+DCAz/ov2Z0VJ0/4FBCtBE7p0ARaKE8cAnSfks1...
	// should be ignored, as it's an unknown authorization scheme
	req.Header.Set("Authorization", fmt.Sprintf("laserauth %s", "omglazerspewpewpew"))

	// token is a known scheme, so we should get this
	req.Header.Add("Authorization", fmt.Sprintf("token %s", sessId))

	s := SessionId(req)
	assert.Equal(t, sessId, s, "they should be equal")
}
