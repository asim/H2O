package i18n

import (
	"testing"

	localisation "github.com/HailoOSS/go-hailo-lib/localisation/hob"
)

func TestPhoneToInternational(t *testing.T) {

	mockCache := &localisation.MockHobsCache{}
	mockCache.On("ReadHob", "NYC").Return(&localisation.Hob{
		Code:  "NYC",
		Phone: localisation.Phone{CallingCode: "+1", TrunkPrefix: "0"},
	})
	mockCache.On("ReadHob", "LON").Return(&localisation.Hob{
		Code:  "LON",
		Phone: localisation.Phone{CallingCode: "+44", TrunkPrefix: "0"},
	})
	mockCache.On("ReadHob", "BUD").Return(&localisation.Hob{
		Code:  "BUD",
		Phone: localisation.Phone{CallingCode: "+36", TrunkPrefix: "06"},
	})
	localisation.Cache = mockCache
	testCases := []struct {
		phone    string
		hobCode  string
		expected string
	}{
		// 00 overrides calling code
		{"0044123 456 789", "NYC", "+44123456789"},
		// + overrides calling code
		{"+4412 34 56 789", "NYC", "+44123456789"},

		// UK
		{"0123456789", "LON", "+44123456789"},
		{"+44123456789", "LON", "+44123456789"},
		{"+44123 456789", "LON", "+44123456789"},
		{"44123456789", "LON", "+44123456789"},
		{"0044123456789", "LON", "+44123456789"},
		{"0044123 456 789", "LON", "+44123456789"},
		{"0123 456789", "LON", "+44123456789"},

		// US
		{"+1555123456", "NYC", "+1555123456"},
		{"555123456", "NYC", "+1555123456"},
		{"555 123 456", "NYC", "+1555123456"},

		// hungary (different length trunk prefix)
		{"0036123456789", "BUD", "+36123456789"},
		{"0036 123 456 789", "BUD", "+36123456789"},
		{"06123456789", "BUD", "+36123456789"},
		{"06 123 45 6789", "BUD", "+36123456789"},

		// short cut if already looks international (don't check HOB)
		// hob FOO hasn't been mocked so test will panic if it doesn't do the right thing
		{"+44123456789", "FOO", "+44123456789"},
		{"+44123 456789", "FOO", "+44123456789"},
		{"0044123456789", "FOO", "+44123456789"},
		{"0044123 456 789", "FOO", "+44123456789"},
	}

	for _, tc := range testCases {
		if got, _ := PhoneToInternational(tc.hobCode, tc.phone); tc.expected != got {
			t.Errorf(`Expected "%s" got "%s"`, tc.expected, got)
		}
	}
}
