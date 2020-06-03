package handler

import (
	log "github.com/cihub/seelog"
	inst "github.com/hailo-platform/H2O/service/instrumentation"
	"net/http"
	"regexp"
	"strings"
)

func extractHobFromRequest(r *http.Request) string {
	hobCode := ""
	for k, vs := range r.URL.Query() {
		switch k {
		case hob:
			hobCode = vs[0]
			continue
		case city:
			if len(vs[0]) >= 3 && len(hobCode) < 3 {
				hobCode = vs[0]
			}
		}
	}
	return hobCode
}

// instrumentHob looks for the hob code and instruments it in graphite
func instrumentHob(r *http.Request, hobCode string) {
	if len(hobCode) > 0 {
		inst.Counter(1.0, "hob."+hobCode, 1)
		return
	}
	hobCode = extractHobFromRequest(r)
	inst.Counter(1.0, "hob."+hobCode, 1)
}

// sanitizeKey this replaces all non alphanumeric characters and converts uppercase chars to lowercase
// this is because graphite does not like funky characters in metric names
// replaces dots as well so only pass in nodes and not a full graphite pass (eg. "handler" is valid, "handler.whatever" is not)
func sanitizeKey(key string) string {
	regStr := "[^a-zA-Z0-9]"
	reg, err := regexp.Compile(regStr)
	if err != nil {
		log.Warnf("Can not compile regex %v, can't sanitize key %v", regStr, key)
		return key
	}
	return strings.ToLower(string(reg.ReplaceAll([]byte(key), []byte("_"))))
}
