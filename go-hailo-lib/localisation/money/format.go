package money

import (
	"strings"

	"github.com/HailoOSS/i18n-go/money"
)

// formatMoney for apps -- in the way they expect, eg: 12.34
func FormatMoney(amount int64, currency string) string {
	m := money.New(amount, currency)
	return strings.TrimRight(strings.TrimSuffix(m.String(), currency), " ")
}
