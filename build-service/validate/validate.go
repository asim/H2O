package validate

import (
	"fmt"
	"reflect"
	"strings"
)

// Validate will inspect the struct, s using reflection
// and check that it meets the specified validation rules
// defined as struct tags.
func Validate(s interface{}) []error {
	errors := make([]error, 0)

	v := reflect.ValueOf(s)
	t := v.Type()

	if t.Kind() == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
	}

	nonblank := make([]string, 0)
	urls := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		validate := f.Tag.Get("validate")
		if validate != "" {
			parts := strings.Split(validate, ",")
			for _, p := range parts {
				if p == "nonblank" {
					nonblank = append(nonblank, f.Name)
				}
				if p == "url" {
					urls = append(urls, f.Name)
				}
			}
		}
	}

	for _, nb := range nonblank {
		f := v.FieldByName(nb)
		s := f.String()

		if s == "" {
			errors = append(errors, fmt.Errorf("%s cannot be blank", nb))
		}
	}

	return errors
}
