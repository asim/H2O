package localisation

import (
	"math/rand"
	"regexp"
	"time"
)

var hobRe = regexp.MustCompile(`^([A-Z]{3})`)

// ExtractHobFromID takes an ID of the form LON1234 and returns LON
func ExtractHobFromID(id string) string {
	if match := hobRe.FindStringSubmatch(id); match != nil {
		return match[0]
	} else {
		return ""
	}
}

// ExtractCityFromID is backwards compatible version of ExtractHobFromID
func ExtractCityFromID(id string) string {
	return ExtractHobFromID(id)
}

func jitter(j time.Duration) time.Duration {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return time.Duration(int64(rng.Float64() * float64(j)))
}
