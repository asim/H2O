package handler

import (
	"net"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
)

// RealIPHandler ensures that the request's RemoteAddr field is correctly populated with the actual client's IP address,
// rather than that of any intermediary.
type RealIPHandler struct {
	Handler http.Handler
}

func (h *RealIPHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if addr := r.Header.Get("X-Forwarded-For"); addr != "" {
		host := strings.SplitN(addr, ", ", 2)[0]
		port := r.Header.Get("X-Forwarded-Port")
		if port == "" {
			var err error
			_, port, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				log.Warnf("[RealIPHandler] Error splitting host/port: %s", err.Error())
				h.Handler.ServeHTTP(rw, r)
				return
			}

		}
		r.RemoteAddr = net.JoinHostPort(host, port)
	}

	h.Handler.ServeHTTP(rw, r)
}
