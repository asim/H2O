package handler

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	pprof "net/http/pprof"
	rpprof "runtime/pprof"
	"time"

	// gorilla_handlers "github.com/gorilla/handlers"
	ptomb "gopkg.in/tomb.v2"

	"github.com/hailo-platform/H2O/api-proxy/controlplane"
	"github.com/hailo-platform/H2O/api-proxy/statusmonitor"
	service "github.com/hailo-platform/H2O/platform/server"
)

const faviconStr = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQAQMAAAAlPW0iAAAAA1BMVEX9vSx51WFPAAAADElEQVQImWNgIA0AAAAwAAFDlLdnAAAAAElFTkSuQmCC"

var favicon, _ = base64.StdEncoding.DecodeString(faviconStr)

type HailoServer struct {
	Tomb              *ptomb.Tomb // used to kill the server
	CORSHandler       *CORSHandler
	HTTPServer        *http.Server
	RealIPHandler     *RealIPHandler
	ThrottlingHandler *ThrottlingHandler
	Control           *controlplane.ControlPlane
	Monitor           *statusmonitor.StatusMonitor
}

func (h *HailoServer) Kill(reason error) {
	h.Tomb.Kill(reason)
	h.Control.Kill(reason)
	h.Monitor.Kill(reason)
	h.Tomb.Wait()
	h.Control.Wait()
	h.Monitor.Wait()
}

// Port on which the HTTP server will listen
const listenPort = 8080

// Register our endpoint handlers in an http.ServeMux instance
func initServeMux(s *http.ServeMux, srv *HailoServer) {
	s.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"version":"%v"}`, service.Version)
	})
	s.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(favicon)
	})

	rpprof.StopCPUProfile()
	s.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	s.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	s.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	s.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))

	rpcH := RpcHandler(srv)
	s.HandleFunc("/rpc", rpcH)        // RPC direct to H2 service
	s.HandleFunc("/v2/h2/call", rpcH) // (Deprecated)
	s.HandleFunc("/", Handler(srv))   // Default handler
	// Define health check endpoint
	s.HandleFunc("/v2/az/status", srv.Monitor.Handler)
	s.HandleFunc("/status", statusmonitor.StatusHandler)
	s.HandleFunc("/endpoints", EndpointsHandler(srv))
}

// Creates a new server, with the correct timeouts, throttling, etc.
func NewServer(accessLogWriter io.Writer) *HailoServer {
	srv := &HailoServer{
		Tomb:    new(ptomb.Tomb),
		Control: controlplane.New(),
	}

	srv.Monitor = statusmonitor.NewStatusMonitor()

	h := http.NewServeMux()
	initServeMux(h, srv)
	ch := &CORSHandler{Handler: h}
	pf := NewPCIFilter(ch, srv)
	th := NewThrottlingHandler(pf, srv)
	// gh := gorilla_handlers.LoggingHandler(accessLogWriter, th)
	ih := &RealIPHandler{th}

	srv.HTTPServer = &http.Server{
		Addr:           fmt.Sprintf(":%v", listenPort),
		Handler:        ih,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: http.DefaultMaxHeaderBytes,
	}
	srv.ThrottlingHandler = th
	srv.CORSHandler = ch
	srv.RealIPHandler = ih
	return srv
}
