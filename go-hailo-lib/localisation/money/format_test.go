package money

import (
	"testing"
)

func TestFormatMoney(t *testing.T) {
	testCases := []struct {
		amount   int64
		currency string
		expected string
	}{
		{
			amount:   1000,
			currency: "GBP",
			expected: "10.00",
		},
	}

	for i, tc := range testCases {
		if actual := FormatMoney(tc.amount, tc.currency); tc.expected != actual {
			t.Errorf("Incorrect money (%d): expected %s got %s", i, tc.expected, actual)
		}
	}
}
