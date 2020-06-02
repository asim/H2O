package validate

import (
	"fmt"
	"github.com/HailoOSS/go-hailo-lib/multierror"
	"reflect"
)

type Validator struct {
	fieldChecks, methodChecks []check
}

func New() *Validator {
	return &Validator{
		fieldChecks:  make([]check, 0),
		methodChecks: make([]check, 0),
	}
}

type Rule func(v reflect.Value) error

type check struct {
	what string
	rule Rule
}

// CheckField adds a check to this validator to look at struct field `field` and
// test this for every rule in `rules`
func (v *Validator) CheckField(field string, rules ...Rule) *Validator {
	for _, rule := range rules {
		v.fieldChecks = append(v.fieldChecks, check{field, rule})
	}

	return v
}

// CheckMethod adds a check to this validator to look at an expected method
// named `method`, and then test the return value from this method against
// every rule in `rules`
func (v *Validator) CheckMethod(method string, rules ...Rule) *Validator {
	for _, rule := range rules {
		v.methodChecks = append(v.methodChecks, check{method, rule})
	}

	return v
}

// Validate tests all defined checks within this validator against some value `s`
func (v *Validator) Validate(s interface{}) *multierror.MultiError {
	reflV := reflect.ValueOf(s)
	reflVf := reflV

	if t := reflV.Type(); t.Kind() == reflect.Ptr {
		reflVf = reflV.Elem()
	}

	errs := multierror.New()
	for _, check := range v.fieldChecks {
		val := reflVf.FieldByName(check.what)
		if err := check.rule(val); err != nil {
			errs.Add(fmt.Errorf("%s fails validation: %v", check.what, err))
		}
	}
	for _, check := range v.methodChecks {
		m := reflV.MethodByName(check.what)
		if m.IsValid() == false {
			errs.Add(fmt.Errorf("%s() fails validation: method not valid", check.what))
			continue
		}
		// assumptions - hardcoded (a) we do call with no params, (b) 1 thing returned
		vals := m.Call([]reflect.Value{})
		val := vals[0]
		if err := check.rule(val); err != nil {
			errs.Add(fmt.Errorf("%s() fails validation: %v", check.what, err))
		}
	}

	if errs.AnyErrors() {
		return errs
	}

	return nil
}
