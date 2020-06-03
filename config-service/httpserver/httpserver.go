package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/config-service/handler"
	"github.com/hailo-platform/H2O/platform/errors"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
)

const (
	IntialBackoff    = 5 * time.Second
	BackoffIncrement = 5 * time.Second
	MaxBackoff       = 60 * time.Second
)

// Server establishes a listener for serving compiled config for HTTP
func Serve(name, source string, version uint64) {
	// /compile?ids=foo,bar,baz&path=foo.bar.baz
	http.HandleFunc("/compile", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metric := "success"
		defer func() {
			inst.Timing(1.0, metric+".httpcompile", time.Since(start))
		}()

		ids := strings.Split(r.URL.Query().Get("ids"), ",")
		path := strings.Replace(r.URL.Query().Get("path"), ".", "/", -1)

		cfg, hash, pfErr := handler.DoCompile(ids, path)
		if pfErr != nil {
			metric = "error"
			writeError(w, pfErr)
			return
		}

		config := map[string]interface{}{}
		err := json.Unmarshal([]byte(cfg), &config)
		if err != nil {
			metric = "error"
			writeError(w, errors.InternalServerError("com.hailocab.service.config.http.unmarshal", fmt.Sprintf("Failed to unmarshal config, when translating to HTTP response: %v", err)))
			return
		}

		response := map[string]interface{}{
			"config": config,
			"hash":   hash,
		}
		b, err := json.Marshal(response)
		if err != nil {
			metric = "error"
			writeError(w, errors.InternalServerError("com.hailocab.service.config.http.marshal", fmt.Sprintf("Failed to marshal to HTTP response: %v", err)))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(b)
	})

	// root resource
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"about":   name,
			"version": version,
			"docs":    source,
		}
		b, err := json.Marshal(response)
		if err != nil {
			writeError(w, errors.InternalServerError("com.hailocab.service.config.http.marshal", fmt.Sprintf("Failed to marshal to HTTP response: %v", err)))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(b)
	})

	var backoff time.Duration = IntialBackoff

	// Attempt to bind to port, retry and flag healthchecks if failure
	for {

		errorChannel := make(chan error)

		// Need to know if this is successful to clear healthcheck. This blocks so we need to fire it off in a go routine
		// and assume success to clear the error flag if no error is found after a given time.  We then listen on the channel
		// in case that assumption is wrong.
		go func(errChan chan error) {
			errChan <- http.ListenAndServe(":8097", nil)
		}(errorChannel)

		select {
		case err := <-errorChannel:
			log.Criticalf("Failed to start HTTP server for compiled configuration: %v", err)
			SetConnectHealthCheck(err)
		case <-time.After(1 * time.Second):
			// No error after given time, assume successful connection
			SetConnectHealthCheck(nil)

			// Wait on channel in case error happens after timeout at any time
			err := <-errorChannel

			// We have an error.  Notify healthcheck
			if err != nil {
				log.Criticalf("Failed to start HTTP server for compiled configuration: %v", err)
				SetConnectHealthCheck(err)
			}
		}

		time.Sleep(backoff)

		if backoff <= MaxBackoff {
			backoff += BackoffIncrement
		}
		log.Warnf("Retrying http server initialization. Backoff set to %s", backoff.String())
	}
}
