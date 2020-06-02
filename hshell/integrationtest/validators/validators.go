package validators

import (
	"github.com/bitly/go-simplejson"
	"github.com/HailoOSS/hshell/integrationtest/request"
	"regexp"
)

func JsonValidator(m map[string]bool) request.CustomValidationFunc {

	var jsonValFunc request.CustomValidationFunc = func(b []byte) bool {
		js, err := simplejson.NewJson(b)
		if err != nil {
			return false
		}

		for name, val := range m {
			jsonVal, ok := js.CheckGet(name)
			if !ok {
				return false
			}

			stat, err := jsonVal.Bool()
			if err != nil {
				return false
			}
			if stat != val {
				return false
			}
		}

		return true
	}
	return jsonValFunc
}

func RegexValidator(regex string) request.CustomValidationFunc {
	var regexValFunc request.CustomValidationFunc = func(b []byte) bool {
		matched, _ := regexp.MatchString(regex, string(b))
		return matched
	}
	return regexValFunc
}
