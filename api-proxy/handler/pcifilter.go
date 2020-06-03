package handler

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	log "github.com/cihub/seelog"
	ptomb "gopkg.in/tomb.v2"

	"github.com/hailo-platform/H2O/service/config"
)

type PCIFilter struct {
	sync.RWMutex
	sensitivePaths  map[string]bool
	currentProxyURL *url.URL
	handler         http.Handler
	proxy           http.Handler
}

func NewPCIFilter(h http.Handler, srv *HailoServer) *PCIFilter {
	f := &PCIFilter{
		handler: h,
	}
	f.loadConfig()
	ch := config.SubscribeChanges()
	srv.Tomb.Go(func() error {
		return f.listenConfig(ch, srv.Tomb)
	})
	return f
}

func (f *PCIFilter) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if f.IsSensitiveRequest(r) {
		http.Error(rw, "Sensitive requests should be sent to the secure API", http.StatusBadRequest)
	} else {
		f.handler.ServeHTTP(rw, r)
	}
}

func (f *PCIFilter) listenConfig(ch <-chan bool, tomb *ptomb.Tomb) error {
	for {
		select {
		case <-ch:
			log.Tracef("[PCIFilter] Received config change")
			f.loadConfig()
		case <-tomb.Dying():
			log.Tracef("[PCIFilter] Dying in response to tomb death")
			return nil
		}
	}
}

func (f *PCIFilter) loadConfig() {
	f.Lock()
	defer f.Unlock()

	encryptionProxyURL, _ := url.Parse(config.AtPath("pci", "encryptionProxyUrl").AsString(""))
	if encryptionProxyURL != f.currentProxyURL {
		log.Debugf("[PCIFilter] Switching encryption proxy URL to %s", encryptionProxyURL.String())
		f.proxy = httputil.NewSingleHostReverseProxy(encryptionProxyURL)
		f.currentProxyURL = encryptionProxyURL
	}

	f.sensitivePaths = map[string]bool{}
	for _, path := range config.AtPath("pci", "sensitivePaths").AsStringArray() {
		f.sensitivePaths[path] = true
	}
}

func (f *PCIFilter) IsSensitiveRequest(req *http.Request) bool {
	f.RLock()
	defer f.RUnlock()
	methodAndPath := fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	return f.sensitivePaths[methodAndPath] && req.Header.Get("X-Encrypted") != "true"
}

func (f *PCIFilter) Forward(rw http.ResponseWriter, r *http.Request) {
	f.RLock()
	p := f.proxy
	log.Tracef("[PCIFilter] Forwarding to %s", f.currentProxyURL.String())
	f.RUnlock()
	p.ServeHTTP(rw, r)
}
