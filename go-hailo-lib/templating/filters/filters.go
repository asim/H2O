package filters

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	c "github.com/HailoOSS/i18n-go/currency"
	loc "github.com/HailoOSS/i18n-go/locale"
	"github.com/HailoOSS/i18n-go/money"

	"github.com/HailoOSS/monday"
	"github.com/HailoOSS/pongo2"
)

const (
	kmToMilesConstant float64 = 0.621371
)

// Passthrough does nothing to the supplied input - just passes it through
func Passthrough(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return in, nil // nothing to do here, just to keep track of the safe application
}

// LocalizedFormatCurrency returns a filter that formats currency for display WITH symbol
func LocalizedFormatCurrency(currency string, locale string) pongo2.FilterFunction {
	return func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		fAmount, err := strconv.ParseFloat(in.String(), 64)
		var amount int64
		if err != nil {
			amount = 0
		}
		amount = int64(fAmount)
		currencyParam := param.String()
		if currencyParam != "" {
			currency = currencyParam
		}

		money := money.Money{
			C: currency,
			M: amount,
		}

		return pongo2.AsValue(money.Format(locale)), nil
	}
}

// FormatCurrencyAmount returns a filter that formats currency amount for display WITHOUT a currency symbol
func FormatCurrencyAmount(locale string) pongo2.FilterFunction {
	return func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

		fAmount, err := strconv.ParseFloat(in.String(), 64)
		var amount int64
		if err != nil {
			amount = 0
		}

		amount = int64(fAmount)
		currency := param.String()

		money := money.Money{
			M: amount,
			C: currency,
		}

		return pongo2.AsValue(money.FormatNoSymbol(locale)), nil
	}
}

// FormatDecimalAmount returns a filter that formats decimal amount for display WITHOUT a currency symbol
func FormatDecimalAmount(locale string) pongo2.FilterFunction {
	l := loc.Get(locale)
	cName := "GBP"
	if l != nil {
		cName = l.CurrencyCode
	}
	currency := c.Get(cName)
	defaultDigits := currency.DecimalDigits
	return func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

		log.Tracef("[FormatDecimalAmount] 000 IN: %s PARAM: %s LOCALE: %d", in.String(), param.String(), locale)

		if len(in.String()) == 0 {
			return pongo2.AsValue(""), nil
		}

		fAmount, err := strconv.ParseFloat(in.String(), 64)
		if err != nil {
			return nil, &pongo2.Error{
				Sender:   "filterFormatDecimalAmount",
				ErrorMsg: fmt.Sprintf("Error formatting value - not parseable '%v': error: %s", in, err),
			}
		}

		digits := defaultDigits
		if param.IsInteger() {
			digits = param.Integer()
			log.Tracef("[FormatDecimalAmount] IN: %s PARAM: %s LOCALE: %s DIGITS: %d", in.String(), param.String(), locale, digits)
		} else if param.IsString() && len(param.String()) > 0 {
			cName = param.String()
			currency := c.Get(cName)
			log.Tracef("[FormatDecimalAmount] IN: %s PARAM: %s LOCALE: %d CURRENCY: %s DIGITS: %d", in.String(), param.String(), locale, cName, digits)
			digits = currency.DecimalDigits
		}

		log.Tracef("[FormatDecimalAmount] IN: %s PARAM: %s LOCALE: %d DIGITS: %d", in.String(), param.String(), locale, digits)

		if digits > 0 {
			return pongo2.AsValue(strconv.FormatFloat(fAmount, 'f', digits, 64)), nil
		}
		return pongo2.AsValue(strconv.FormatInt(int64(fAmount), 10)), nil
	}
}

// FormatShortCurrencyAmount returns a filter that formats currency amount for display WITHOUT a currency symbol in a bizarrely complex way
func FormatShortCurrencyAmount(locale string) pongo2.FilterFunction {
	return func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

		fAmount, err := strconv.ParseFloat(in.String(), 64)
		var amount int64
		if err != nil {
			amount = 0
		}

		amount = int64(fAmount)
		currency := param.String()

		money := money.Money{
			M: amount,
			C: currency,
		}

		lce := loc.Get(locale)
		moneyAmt := money.FormatNoSymbol(locale)
		if lce != nil &&
			len(moneyAmt) > lce.CurrencyDecimalDigits+1 &&
			hasZeroCents(moneyAmt, lce.CurrencyDecimalDigits) {
			moneyAmt = moneyAmt[:len(moneyAmt)-(lce.CurrencyDecimalDigits+1)]
		}
		return pongo2.AsValue(moneyAmt), nil
	}
}

func hasZeroCents(amt string, centDigits int) bool {
	if centDigits < 1 {
		return false
	}

	for i := 1; i <= centDigits; i++ {
		digit, _ := strconv.ParseInt(string(amt[len(amt)-i]), 10, 64)
		if digit != 0 {
			return false
		}
	}
	return true
}

// TTFormats is a list of time formats as a list of strings used for parsing times supplied as strings
var TTFormats = []string{"2006-01-02 15:04:05.0 MST", "2006-01-02 15:04:05.0", "2006-01-02"}

// LocalizedDateFormatter returns a formatter for the context locale and time-zone that attempts to parse the input
// using a list of formats until a successful parse is achieved.
func LocalizedDateFormatter(locale string, timezone string) pongo2.FilterFunction {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		log.Errorf("Failed to load timezone: %v", err)
		location, _ = time.LoadLocation("Europe/London")
	}

	return func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		tstampEpoch, err := strconv.ParseInt(in.String(), 10, 64)
		timestamp := time.Unix(tstampEpoch, 0)

		if err != nil {
			timestamp = ParseTimeFromString(in.String())
		}

		format := param.String()

		// Find monday.Locale for locale string
		var mLocale monday.Locale
		mLocale = monday.LocaleEnGB
		for _, l := range monday.ListLocales() {
			if string(l) == locale {
				mLocale = l
				break
			}
		}

		out := monday.Format(timestamp.In(location), format, mLocale)

		log.Debugf("[LocalizedDateFormatter] Location: %s (%s) Monday fmt: %s Time fmt: %s (%s)", locale, timezone, out, timestamp.Format(format), timestamp.In(location).Format(format))

		return pongo2.AsValue(out), nil
	}
}

// ParseTimeFromString parses a date from the supplied value using an optional supplied location
func ParseTimeFromString(in string, loc ...*time.Location) time.Time {
	var timestamp time.Time
	var err error
	for _, tf := range TTFormats {
		if len(loc) > 0 {
			timestamp, err = time.ParseInLocation(tf, in, loc[0])
		} else {
			timestamp, err = time.Parse(tf, in)
		}
		if err == nil {
			break
		}
		log.Warnf("Failed to parse as text: %v in: %s using: %s", err, in, tf)
		timestamp = time.Unix(0, 0)
	}
	return timestamp
}

// SimpleDateFormatter given a time-zone deduces an appropriate location and returns a filter that prses a date from a
// supplied string or timestamp.
func SimpleDateFormatter(timezone string) pongo2.FilterFunction {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		log.Errorf("Failed to load timezone: %v", err)
		location, _ = time.LoadLocation("Europe/London")
	}

	return func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		tstampEpoch, err := strconv.ParseInt(in.String(), 10, 64)
		timestamp := time.Unix(tstampEpoch, 0)
		format := param.String()

		if err != nil {
			log.Warnf("Failed to parse timestamp: %v", err)
			timestamp = time.Unix(0, 0)
		}

		return pongo2.AsValue(monday.Format(timestamp.In(location), format, monday.LocaleEnUS)), nil
	}
}

// CurrencySymbol returns a filter that retruns a currency symbol for the supplied currency isocode.
func CurrencySymbol(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	code := in.String()
	if code == "" {
		return pongo2.AsValue(""), nil
	}
	currencyObj := c.Get(code)
	return pongo2.AsValue(currencyObj.Symbol), nil
}

// Capitalize capitalizes the supplied string
func Capitalize(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if !in.IsString() {
		return pongo2.AsValue(""), nil
	}
	return pongo2.AsValue(strings.Title(strings.ToLower(in.String()))), nil
}

// EscapeEntities applies HTML escaping tot he supplied input string.
func EscapeEntities(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	output := in.String()
	output = strings.Replace(output, "&", "&amp;", -1)
	output = strings.Replace(output, ">", "&gt;", -1)
	output = strings.Replace(output, "<", "&lt;", -1)
	output = strings.Replace(output, "\"", "&quot;", -1)
	output = strings.Replace(output, "'", "&#39;", -1)
	output = strings.Replace(output, " ", "&nbsp;", -1)
	return pongo2.AsValue(output), nil
}

// Split splits a string based on a separator token.
func Split(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if !in.CanSlice() {
		return in, nil
	}
	s := in.String()
	sep := param.String()

	result := make([]string, 0, len(s))
	for _, c := range s {
		result = append(result, string(c))
	}
	return pongo2.AsValue(strings.Split(s, sep)), nil
}

// UnmarshalJson interprets the input value as a JSON map or array and returns a Go map or array.
func UnmarshalJson(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()

	if s != "" {
		var result []interface{}
		err := json.Unmarshal([]byte(s), &result)
		if err == nil {
			log.Debugf("[UnmarshalJson] JSON OUT LIST ===== %s", result)
			return pongo2.AsValue(result), nil
		}
		var result2 map[string]interface{}
		err2 := json.Unmarshal([]byte(s), &result2)
		if err2 == nil {
			log.Debugf("[UnmarshalJson] JSON OUT MAP ===== %s", result2)
			return pongo2.AsValue(result2), nil
		}
		return nil, &pongo2.Error{
			Sender:   "filterUnmarshalJson",
			ErrorMsg: fmt.Sprintf("Error unmarshaling value '%v' %s", err, s),
		}
	}

	return pongo2.AsValue([]string{}), nil
}

// ConvertKilometersToMiles converts the supplied value in Km as Miles
func ConvertKilometersToMiles(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	distance, err := strconv.ParseFloat(in.String(), 64)
	if err != nil {
		distance = 0
	}

	return pongo2.AsValue(fmt.Sprintf("%01.1f", float64(distance)*kmToMilesConstant)), nil
}

// MaskAccountNumber applies various maskings to a number (or string>) supplied as a string, e.g. to card number.
func MaskAccountNumber(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if in == nil || len(in.String()) < 7 {
		return in, nil
	}
	n, l := 4, len(in.String())
	if param != nil && param.IsInteger() && param.Integer() > 0 {
		n = param.Integer()
		if n > l {
			n = l
		}
	}
	masked := strings.Repeat("*", n)
	if n < l {
		masked = masked + in.String()[n:]
	}
	return pongo2.AsValue(masked), nil
}

// LookupMap returns a value extracted from the map supplied as the value for the key supplied as param.
func LookupMap(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	log.Debugf("[LookupMap] IN PARAM: %+v %+v", *in, *param)
	if !in.Contains(param) {
		return pongo2.AsValue(""), nil
	}
	ii := in.Interface()
	log.Debugf("[LookupMap] Lookup value: %v %s", ii, param.String())
	switch ii.(type) {
	case map[string]interface{}:
		vv := reflect.ValueOf(ii).MapIndex(reflect.ValueOf(param.String()))
		log.Debugf("Found value: %v", vv.Interface())
		return pongo2.AsValue(vv.Interface()), nil
	default:
		return nil, &pongo2.Error{
			Sender:   "filterLookupMap",
			ErrorMsg: fmt.Sprintf("Error looking up value - lookup error '%v' in '%v'", param, in),
		}
	}
}

// InsertSymbol sort of inserts a symbol
func InsertSymbol(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	log.Debugf("[InsertSymbol] IN PARAM: %+v %+v", *in, *param)
	if !in.IsString() {
		return nil, &pongo2.Error{
			Sender:   "filterInsertSymbol",
			ErrorMsg: fmt.Sprintf("Target should be string: %+v", in),
		}
	}
	i := in.String()
	pi := param.Interface()
	pm, ok := pi.(map[string]interface{})
	if !ok {
		return nil, &pongo2.Error{
			Sender:   "filterInsertSymbol",
			ErrorMsg: fmt.Sprintf("Param should be map[string]interface{}: %+v %T", pi, pi),
		}
	}
	ps := "-"
	switch t := pm["s"].(type) {
	default:
		return nil, &pongo2.Error{
			Sender:   "filterInsertSymbol",
			ErrorMsg: fmt.Sprintf("['s'] should be type string: %+v %T", pm["s"], t),
		}
	case string:
		ps = pm["s"].(string)
	}
	pl := []int{}
	switch t := pm["l"].(type) {
	default:
		return nil, &pongo2.Error{
			Sender:   "filterInsertSymbol",
			ErrorMsg: fmt.Sprintf("['l'] should be type []int: %+v %T", pm["l"], t),
		}
	case []int:
		pl = pm["l"].([]int)
	case []interface{}:
		ipl := pm["l"].([]interface{})
		pl = []int{}
		for _, ip := range ipl {
			pl = append(pl, int(ip.(float64)))
		}
	}
	out := doInsertSymbol(i, ps, pl)
	log.Debugf("[InsertSymbol] Re-formatted value: %s", out)
	return pongo2.AsValue(out), nil
}

func doInsertSymbol(i, ps string, pl []int) string {
	if len(pl) == 0 || pl[0] >= len(i) {
		return i
	}
	out := ""
	j := -1
	for k, l := range pl {
		if k == 0 {
			out = i[:l] + ps
		} else if l >= len(i) {
			out = out + i[j:]
			return out
		} else {
			out = out + i[j:l] + ps
		}
		j = l
	}
	if j < len(i) {
		out = out + i[j:]
	}
	return out
}
