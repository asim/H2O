package session

import (
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
)

var authorizationScheme = "token"

type headerExtractor func(key, value string) string

var headerExtractors = map[string]headerExtractor{
	"X-Api-Token":   extractXApiTokenHeader,
	"Authorization": extractAuthorizationHeader,
}

// SessionId extracts the session ID from an HTTP request
func SessionId(r *http.Request) string {
	if r == nil {
		return ""
	}

	var sessId string

	// Try to obtain from the query string
	if sessId = r.URL.Query().Get("session_id"); sessId != "" {
		return sessId
	}
	if sessId = r.URL.Query().Get("api_token"); sessId != "" {
		return sessId
	}

	// Try request body if we don't have a session from the query string
	if sessId = r.Form.Get("session_id"); sessId != "" {
		return sessId
	}
	if sessId = r.Form.Get("api_token"); sessId != "" {
		return sessId
	}

	// Finally try to extract from headers
	// Grab the header, and iterate over each instance (http allows multiple headers with the same key)
	for hdr, extractor := range headerExtractors {
		for _, v := range r.Header[hdr] {
			if sessId = extractor(hdr, v); sessId != "" {
				log.Tracef("[Session] Session ID extracted from %s header: '%s'", hdr, sessId)
				return sessId
			}
		}
	}

	return ""
}

// extractXApiTokenHeader extracts a token from the X-Api-Token header
// this header just contains the value of the token, so simply returns this
func extractXApiTokenHeader(key, value string) string {
	return value
}

// extractAuthorizationHeader attempts to extract a session id / token
// from the Authorization header within a http request
//
// We are expecting a header format of:
//   Authorization: <scheme> <content>
//
// In the case of the token authorization scheme this would be:
//   Authorization: token WW6ey7SFylhtrrn+DCAz/ov2Z0VJ0/4FBCtBE7p0ARaKE8cAnSfks1...
func extractAuthorizationHeader(key, value string) string {

	// Authorization token is space separated
	parts := strings.Split(value, " ")

	// Invalid if we don't have at least two parts
	if len(parts) < 2 {
		return ""
	}

	// Check our authorization scheme is supported
	if parts[0] != authorizationScheme {
		return ""
	}

	return parts[1]
}
