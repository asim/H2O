package auth

import (
	"encoding/base64"
	"github.com/HailoOSS/go-hailo-lib/validate"
	"strconv"
	"strings"
	"time"
)

var userVal *validate.Validator

func init() {
	userVal = validate.New()
	userVal.CheckField("SessId", validate.NotEmpty)
	userVal.CheckField("Mech", validate.NotEmpty)
	userVal.CheckField("Device", validate.NotEmpty)
	userVal.CheckField("Id", validate.NotEmpty)
	userVal.CheckField("Sig", validate.NotEmpty)
	userVal.CheckField("CreatedTs", validate.NotEmpty)
	userVal.CheckField("ExpiryTs", validate.NotEmpty)
}

// FromSessionToken turns a raw session and token pair into a full user object
// that we can query/validate
func FromSessionToken(s, t string) (*User, error) {
	u := &User{SessId: s, Token: []byte(t), Roles: make([]string, 0)}

	// am=admin:d=cli:id=dave:ct=1372942090:et=1372970890:rt=:r=ADMIN:sig=ktVNvY/xeNlhTz0RHEVJIkEHeDh3lJlXzB8rKJk89ZqMeh0TmUYiIIm13UE4Hd4etlt7bJI8HaUTMReSVMFHy3mue/+lQhrwPWSmvIethkHuJX7ReRNJdsJb85ZzE+rNCkortPwVNhARcG5H/kzhq3IcCJxrCkc+fftdFOaXlv4=
	parts := strings.Split(t, ":")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "am":
			u.Mech = kv[1]
		case "d":
			u.Device = kv[1]
		case "id":
			u.Id = kv[1]
		case "ct":
			i, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				continue
			}
			u.CreatedTs = time.Unix(i, 0)
		case "et":
			i, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				continue
			}
			u.ExpiryTs = time.Unix(i, 0)
		case "rt":
			i, err := strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				continue
			}
			u.RenewTs = time.Unix(i, 0)
		case "r":
			u.Roles = strings.Split(kv[1], ",")
		case "sig":
			// sigs are base64 encoded
			u.Sig, _ = base64.StdEncoding.DecodeString(kv[1])
		}
	}

	// extract everything up to :sig= and assign this to Data (this is what is signed)
	if parts := strings.Split(t, ":sig="); len(parts) == 2 {
		u.Data = []byte(parts[0])
	}

	if errs := userVal.Validate(u); errs.AnyErrors() {
		return nil, errs
	}

	// check signature
	if ok, err := verify(u.Sig, u.Data); !ok {
		return nil, err
	}

	return u, nil
}
