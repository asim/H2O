package handler

import (
	"net/http"
	"regexp"
)

var allowedCorsOrigins = [...]*regexp.Regexp{
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?hailoweb.com(:\d+)?$`),
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?hailoapp.com(:\d+)?$`),
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?HailoOSS.com(:\d+)?$`),
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?hailovpn.com(:\d+)?$`),
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?elasticride.com(:\d+)?$`),
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?elasticride.local(:\d+)?$`),
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?elasticride.dev(:\d+)?$`),
	regexp.MustCompile(`^https?://(?:[-\w\.]+\.)?hailopay.com(:\d+)?$`),
}

type CORSHandler struct {
	Handler http.Handler
}

func (h *CORSHandler) isAllowedCORSOrigin(origin string) bool {
	for _, re := range allowedCorsOrigins {
		if re.MatchString(origin) {
			return true
		}
	}

	return false
}

func (h *CORSHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" && h.isAllowedCORSOrigin(origin) {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
		rw.Header().Set("Access-Control-Allow-Methods", "DELETE, GET, HEAD, OPTIONS, POST, PUT")

		// Allow all headers a client wants to send
		if wantedHeaders := r.Header.Get("Access-Control-Request-Headers"); wantedHeaders != "" {
			rw.Header().Set("Access-Control-Allow-Headers", wantedHeaders)
		}

		// Set the CORS security policy cache if this is an OPTIONS request
		if r.Method == "OPTIONS" {
			rw.Header().Set("Access-Control-Max-Age", "3600")
			return
		}
	}

	h.Handler.ServeHTTP(rw, r)
}
