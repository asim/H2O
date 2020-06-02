package i18n

import (
	"fmt"
	"strings"

	localisation "github.com/HailoOSS/go-hailo-lib/localisation/hob"
)

func PhoneToInternational(hobCode, phone string) (string, error) {
	// remove spaces
	phone = strings.Replace(phone, " ", "", -1)

	// looks international
	if strings.HasPrefix(phone, "+") {
		return phone, nil
	}

	// 00 international, change to +
	if strings.HasPrefix(phone, "00") {
		return "+" + phone[2:], nil
	}
	hob, err := localisation.GetHob(hobCode)
	if err != nil {
		return "", fmt.Errorf("Invalid HOB: %s err:%v", hobCode, err)
	}

	callingCode := hob.Phone.CallingCode
	if len(callingCode) == 0 {
		return "", fmt.Errorf("Missing calling code")
	}

	// strip + from calling code
	callingCode = strings.TrimLeft(callingCode, "+")

	// strip trunk prefix (e.g. initial zeros)
	phone = strings.TrimLeft(phone, hob.Phone.TrunkPrefix)

	// if we have callingCode, all good so add +
	if strings.HasPrefix(phone, callingCode) {
		return "+" + phone, nil
	}

	// prefix with calling code
	return "+" + callingCode + phone, nil
}
