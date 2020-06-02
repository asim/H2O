package handler

import (
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/HailoOSS/go-hailo-lib/validate"
)

var (
	userValidator *validate.Validator
	regexpRole    = regexp.MustCompile("^([\\dA-Z]+)(\\.[\\dA-Za-z-]+)*$")
)

func init() {
	userValidator = validate.New()
	userValidator.CheckField("App", validate.NotEmpty)
	userValidator.CheckField("Uid", validate.NotEmpty)
	userValidator.CheckField("Created", validate.NotEmpty)
	userValidator.CheckField("Created", sensibleTime)
	userValidator.CheckField("Roles", validRoles)
}

// sensibleTime checks for a reasonable time - hailo wasn't around before 2010
func sensibleTime(v reflect.Value) error {
	if t, ok := v.Interface().(time.Time); ok {
		if !t.IsZero() && t.Before(time.Unix(1262304000, 0)) {
			return fmt.Errorf("must not be before 2010: %v", t)
		}
	}

	return nil
}

// validRoles makes sure roles are [A-Z]
func validRoles(v reflect.Value) error {
	r := v.Interface().([]string)
	for _, role := range r {
		if !regexpRole.MatchString(role) {
			return fmt.Errorf("Invalid role - must be like FOO.BAR")
		}
	}
	return nil
}
