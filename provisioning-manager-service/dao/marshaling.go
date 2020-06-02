package dao

import (
	"encoding/json"
)

func flatten(i interface{}) string {
	b, _ := json.Marshal(i)
	return string(b)
}

func unflattenServiceRunLevels(s string) [6]bool {
	var levels [6]bool

	if s == "" || s == "null" {
		// default to returning a running service
		for i, _ := range levels {
			levels[i] = true
		}
		return levels
	}

	if err := json.Unmarshal([]byte(s), &levels); err != nil {
		// default to returning a running service
		for i, _ := range levels {
			levels[i] = true
		}
		return levels
	}

	return levels
}
