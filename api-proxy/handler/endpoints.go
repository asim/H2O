package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// jsonResponse represents a generic json object
type jsonResponse map[string]interface{}

func (r jsonResponse) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	return string(b)
}

// EndpointsHandler serves up app-pinning configuration to apps
func EndpointsHandler(srv *HailoServer) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")

		router := srv.Control.Router(r)
		region, version := router.Region()

		if region == nil {
			rw.WriteHeader(int(http.StatusInternalServerError))
			fmt.Fprint(rw, jsonResponse{
				"status":      false,
				"code":        11,
				"dotted_code": "com.hailocab.hailo-2-api.noregions",
				"payload":     "No online regions found",
			})
			return
		}

		appId := r.URL.Query().Get("app")
		urls := region.Urls(appId)

		verbose, _ := strconv.ParseBool(r.URL.Query().Get("verbose"))

		if urls == nil {
			rw.WriteHeader(int(http.StatusInternalServerError))
			fmt.Fprint(rw, jsonResponse{
				"status":      false,
				"code":        11,
				"dotted_code": "com.hailocab.hailo-2-api.noapps",
				"payload":     "No apps found for this region",
			})
			return
		}

		// construct response
		endpoints := make(map[string]interface{})
		endpoints["timestamp"] = version
		for k, v := range urls {
			endpoints[strings.Replace(k, "_", "-", -1)] = v
		}
		rsp := jsonResponse{
			"status":    true,
			"payload":   "OK",
			"endpoints": endpoints,
		}

		// If verbose requested, return enhanced config
		// clients can use this to cache multiple hobs
		if verbose {
			rsp["regions"] = srv.Control.Regions()
			rsp["hobRegions"] = srv.Control.HobRegions()
		}

		// Output final response
		fmt.Fprint(rw, rsp)
	}
}
