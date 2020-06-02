package validate

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	regexpCurrencyCode  = regexp.MustCompile("^[A-Z]{3}$")
	regexpHob           = regexp.MustCompile("^[A-Z]{3}$")
	regexpOptionalHobId = regexp.MustCompile("^([A-Z]{3}[0-9]+)?$")
)

func Chain(validator *Validator) func(v reflect.Value) error {
	return func(v reflect.Value) error {
		if err := validator.Validate(v.Interface()); err.AnyErrors() {
			return errors.New(err.Error())
		}

		return nil
	}
}

func NotEmpty(v reflect.Value) error {
	if v.Kind() == reflect.Invalid {
		return fmt.Errorf("cannot validate an invalid type")
	}

	if v.Kind() == reflect.Ptr && v.IsNil() {
		return fmt.Errorf("must not be nil")
	}
	// time?
	if t, ok := v.Interface().(time.Time); ok {
		if t.IsZero() {
			return fmt.Errorf("must not be zero time")
		}
	}

	if v.Kind() == reflect.String && strings.TrimSpace(v.String()) == "" {
		return fmt.Errorf("must not be empty string")
	}

	if v.Kind() == reflect.Slice {
		if v.IsNil() {
			return fmt.Errorf("must not be nil")
		} else if v.Len() == 0 {
			return fmt.Errorf("must not be zero length slice")
		}
	}

	return nil
}

func City(v reflect.Value) error {
	return Hob(v)
}

func Hob(v reflect.Value) error {
	return Regexp(regexpHob)(v)
}

func CurrencyCode(v reflect.Value) error {
	return Regexp(regexpCurrencyCode)(v)
}

func CityId(v reflect.Value) error {
	return HobId(v)
}

func HobId(v reflect.Value) error {
	return Regexp(regexpOptionalHobId)(v)
}

func OneOf(allowed ...interface{}) Rule {
	return func(v reflect.Value) error {
		for _, a := range allowed {
			if reflect.DeepEqual(v.Interface(), a) {
				return nil
			}
		}
		return fmt.Errorf("must be one of %v", allowed)
	}
}

func NotOneOf(disallowed ...interface{}) Rule {
	return func(v reflect.Value) error {
		for _, d := range disallowed {
			if reflect.DeepEqual(v.Interface(), d) {
				return fmt.Errorf("must not be one of %v", disallowed)
			}
		}
		return nil
	}
}

func Regexp(re *regexp.Regexp) func(v reflect.Value) error {
	return func(v reflect.Value) error {
		if v.Kind() != reflect.String {
			return fmt.Errorf("must be a string")
		}

		if !re.MatchString(v.String()) {
			return fmt.Errorf("must be %s", re.String())
		}

		return nil
	}
}

// StringLength validates that a value is a string between min/max length (rune count)
func StringLength(min, max int) Rule {
	return func(v reflect.Value) error {
		if v.Kind() != reflect.String {
			return fmt.Errorf("must be a string")
		}
		if l := utf8.RuneCountInString(v.String()); l < min || l > max {
			return fmt.Errorf("must be between %v and %v characters in length", min, max)
		}
		return nil
	}
}

func isFloatBetween(v *reflect.Value, min, max float64) error {
	k := v.Kind()

	if k != reflect.Float32 && k != reflect.Float64 {
		return fmt.Errorf("must be a floating point number")
	}

	fValue := v.Float()
	if fValue < min || fValue > max {
		return fmt.Errorf("must be between %v and %v", min, max)
	}

	return nil
}

func isTimestampBetween(v *reflect.Value, min, max time.Time) error {
	k := v.Kind()

	if k != reflect.Int64 {
		return fmt.Errorf("timestamp must be int64")
	}

	t := time.Unix(v.Int(), 0)

	if t.Before(min) || t.After(max) {
		return fmt.Errorf("must be between %s and %v", min, max)
	}

	return nil
}

func Longitude(v reflect.Value) error {
	return isFloatBetween(&v, -180, 180)
}

func Latitude(v reflect.Value) error {
	return isFloatBetween(&v, -90, 90)
}
