package filters

import (
	"encoding/json"
	"testing"

	"github.com/HailoOSS/pongo2"
	"github.com/stretchr/testify/assert"
)

func helperTestFilter(assert *assert.Assertions, filterFunc pongo2.FilterFunction, value, param interface{}, expected string, message string) {

	in, err := filterFunc(pongo2.AsValue(value), pongo2.AsValue(param))
	assert.Nil(err)
	assert.Equal(expected, in.String(), message)
}

func TestPassthrough(t *testing.T) {
	assert := assert.New(t)

	helperTestFilter(assert, Passthrough, "AVALUE", "", "AVALUE", "Values should passthrough")
	helperTestFilter(assert, Passthrough, "AVALUE", "APARAM", "AVALUE", "Values should passthrough")
}

func TestConvertKilometersToMiles(t *testing.T) {
	assert := assert.New(t)
	helperTestFilter(assert, ConvertKilometersToMiles, "", "", "0.0", "Wrong km to miles convertion it seems")
	helperTestFilter(assert, ConvertKilometersToMiles, "1", "", "0.6", "Wrong km to miles convertion it seems")
	helperTestFilter(assert, ConvertKilometersToMiles, "12", "", "7.5", "Wrong km to miles convertion it seems")
	helperTestFilter(assert, ConvertKilometersToMiles, "23.2", "", "14.4", "Wrong km to miles convertion it seems")
	helperTestFilter(assert, ConvertKilometersToMiles, "23", "", "14.3", "Wrong km to miles convertion it seems")
}

func TestCurrencySymbol(t *testing.T) {
	assert := assert.New(t)

	helperTestFilter(assert, CurrencySymbol, "", "", "", "Wrong currency symbol")
	helperTestFilter(assert, CurrencySymbol, "EUR", "", "€", "Wrong currency symbol")
	helperTestFilter(assert, CurrencySymbol, "GBP", "", "£", "Wrong currency symbol")
}

func TestFormatCurrencyAmount(t *testing.T) {
	assert := assert.New(t)

	formatCurrency := FormatCurrencyAmount("en_GB")

	helperTestFilter(assert, formatCurrency, "", "", "0.00", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "100", "", "1.00", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "121312312312323", "", "1,213,123,123,123.23", "Wrong currency amount")
	// helperTestFilter(assert, formatCurrency, "3510005", "", "1,213,123,123,123.23", "Wrong currency amount")

	formatCurrency = FormatCurrencyAmount("es_ES")

	helperTestFilter(assert, formatCurrency, "", "", "0,00", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "123", "", "1,23", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "121312312312323", "", "1.213.123.123.123,23", "Wrong currency amount")

	formatCurrency = FormatCurrencyAmount("ja_JP")

	helperTestFilter(assert, formatCurrency, "", "JPY", "0", "Wrong currency format")
	helperTestFilter(assert, formatCurrency, "10370", "JPY", "10,370", "Wrong currency format")
}

func TestFormatDecimalAmount(t *testing.T) {
	assert := assert.New(t)

	formatAmount := FormatDecimalAmount("en_GB")

	helperTestFilter(assert, formatAmount, "15.0", "GBP", "15.00", "Wrong currency amount")
	helperTestFilter(assert, formatAmount, "15", "GBP", "15.00", "Wrong currency amount")
	helperTestFilter(assert, formatAmount, "15.0", "JPY", "15", "Wrong currency amount")

	formatAmount = FormatDecimalAmount("ja_JP")
	helperTestFilter(assert, formatAmount, "15.0", "", "15", "Wrong currency amount")
	helperTestFilter(assert, formatAmount, "15.0", 3, "15.000", "Wrong currency amount")
}

func TestFormatShortCurrencyAmount(t *testing.T) {
	assert := assert.New(t)

	formatCurrency := FormatShortCurrencyAmount("en_GB")

	helperTestFilter(assert, formatCurrency, "", "", "0", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "100", "", "1", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "1501", "", "15.01", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "1500", "", "15", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "121312312312323", "", "1,213,123,123,123.23", "Wrong currency amount")

	formatCurrency = FormatShortCurrencyAmount("es_ES")

	helperTestFilter(assert, formatCurrency, "", "", "0", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "123", "", "1,23", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "-1500", "", "-15", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "121312312312323", "", "1.213.123.123.123,23", "Wrong currency amount")

	formatCurrency = FormatShortCurrencyAmount("ja_JP")

	helperTestFilter(assert, formatCurrency, "", "JPY", "0", "Wrong currency format")
	helperTestFilter(assert, formatCurrency, "14500", "JPY", "14,500", "Wrong currency format")
	helperTestFilter(assert, formatCurrency, "10370", "JPY", "10,370", "Wrong currency format")
}

func TestLocalizedFormatCurrency(t *testing.T) {
	assert := assert.New(t)

	formatCurrency := LocalizedFormatCurrency("GBP", "en_GB")

	helperTestFilter(assert, formatCurrency, "", "", "£0.00", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "101", "", "£1.01", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "121312312312323", "", "£1,213,123,123,123.23", "Wrong currency amount")

	formatCurrency = LocalizedFormatCurrency("EUR", "es_ES")

	helperTestFilter(assert, formatCurrency, "", "", "0,00 €", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "123", "", "1,23 €", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "121312312312323", "", "1.213.123.123.123,23 €", "Wrong currency amount")

	formatCurrency = LocalizedFormatCurrency("JPY", "ja_JP")

	helperTestFilter(assert, formatCurrency, "", "", "¥0", "Wrong currency amount")
	helperTestFilter(assert, formatCurrency, "10370", "", "¥10,370", "Wrong currency amount")

	formatCurrency = LocalizedFormatCurrency("EUR", "es_ES")

	helperTestFilter(assert, formatCurrency, "", "", "0,00 €", "Wrong currency amount")

}

func TestSimpleDateFormatter(t *testing.T) {
	assert := assert.New(t)

	ff := SimpleDateFormatter("Europe/London")

	helperTestFilter(assert, ff, "1282367908", "", "", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "Mon 2 January", "Sat 21 August", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "2 Jan 2006", "21 Aug 2010", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "02/01/2006 15:04", "21/08/2010 06:18", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "02/01/2006", "21/08/2010", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "03:04PM", "06:18AM", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "15:04", "06:18", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "2006-01-02 15:04:05", "2010-08-21 06:18:28", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "2006/01/02", "2010/08/21", "Dates should match")
}

func TestLocalizedDateFormatter(t *testing.T) {
	assert := assert.New(t)

	ff := LocalizedDateFormatter("en_GB", "Europe/London")

	helperTestFilter(assert, ff, "1282367908", "", "", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "15:04", "06:18", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "3:04PM", "6:18AM", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "Mon 2 Jan", "Sat 21 Aug", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "Mon 2 January", "Sat 21 August", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "2 Jan 2006", "21 Aug 2010", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "2 January 2006", "21 August 2010", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "2006/01/02", "2010/08/21", "Dates should match")
	helperTestFilter(assert, ff, "1282367908", "2006/01/02 15:04", "2010/08/21 06:18", "Dates should match")
	helperTestFilter(assert, ff, "2015-10-27 23:18:03.0 UTC", "2006/01/02 15:04", "2015/10/27 23:18", "Dates should match")
}

func TestMaskAccountNumber(t *testing.T) {
	assert := assert.New(t)

	testMasking(assert, "123456", nil, "123456")
	testMasking(assert, "1234567", nil, "****567")
	testMasking(assert, "1234567", pongo2.AsValue(3), "***4567")
	testMasking(assert, "1234567", pongo2.AsValue(8), "*******")
}

func testMasking(assert *assert.Assertions, in string, n *pongo2.Value, exp string) {
	out, err := MaskAccountNumber(pongo2.AsValue(in), n)
	assert.Nil(err)
	assert.Equal(exp, out.String())
}

func TestUnmarshal(t *testing.T) {
	tc := `{ "Hailo Account": "Cuenta Hailo", "Hailo Cash": "Efectivo Hailo", "Street Card": "Tarjeta no Hailo", "Scrub (C)": "Anulación pasajero", "Hailo Account: REFUND": "Cuenta Hailo: REEMBOLSO", "Pay With Hailo": "Pagar con Hailo"}`
	out := map[string]interface{}{}
	err := json.Unmarshal([]byte(tc), &out)
	if err != nil {
		t.Errorf("Failed parse: %s", err)
	}
	out2, err2 := UnmarshalJson(pongo2.AsValue(tc), nil)
	if err2 != nil {
		t.Errorf("Failed parse: %s", err2)
	}
	if len(out2.Interface().(map[string]interface{})) == 0 {
		t.Errorf("OUT %+v", out2)
	}
	tc10 := "{\"s\":\"-\",\"l\":[4,9,12]}"
	out12, err12 := UnmarshalJson(pongo2.AsValue(tc10), nil)
	if err12 != nil {
		t.Errorf("Failed parse: %s", err12)
	}
	if len(out12.Interface().(map[string]interface{})) == 0 {
		t.Errorf("OUT %+v", out12)
	}
}

func TestLookupMap(t *testing.T) {
	mm := map[string]interface{}{
		"Sax Rohmer":   "Fu Manchu",
		"Conan Doyle":  "Sherlock Holmes",
		"Iris Murdoch": "Under The Net",
	}
	val, err := LookupMap(pongo2.AsValue(mm), pongo2.AsValue("Sax Rohmer"))
	if err != nil {
		t.Errorf("Failed in LookupMap: error: %s", err)
	}
	if val == nil || val.String() != "Fu Manchu" {
		t.Errorf("Failed to look up 'Sax Rohmer'")
	}
	val, err = LookupMap(pongo2.AsValue(mm), pongo2.AsValue("William Gibson"))
	if err != nil {
		t.Errorf("Should have returned \"\" not error in LookupMap: error: %s", err)
	}
	if val.String() != "" {
		t.Errorf("Should not have looked up 'William Gibson'")
	}
}

type disCase struct {
	in, expected, sym string
	locs              []int
}

func TestDoInsertSymbol(t *testing.T) {
	cases := []disCase{
		disCase{"hello", "hello", "-", []int{}},
		disCase{"whatdayisit?", "what-day-is-it-?", "-", []int{4, 7, 9, 11}},
		disCase{"whatdayisit?", "what day is it?", " ", []int{4, 7, 9}},
		disCase{"whatdayisit?", "what day is it ?", " ", []int{4, 7, 9, 11}},
		disCase{"hi", "hi", "-", []int{4, 7, 9, 11}},
		disCase{"whatdayisit", "-what-day-is-it", "-", []int{0, 4, 7, 9, 11}},
		disCase{"whatdayisit", "what-day-is-it", "-", []int{4, 7, 9, 11, 13}},
		disCase{"****************1088", "****-****-**-******1088", "-", []int{4, 8, 10}},
	}
	for i, c := range cases {
		if res := doInsertSymbol(c.in, c.sym, c.locs); c.expected != res {
			t.Errorf("Case[%d]: expected: %s got: %s", i, c.expected, res)
		}
	}
}

func TestInsertSymbol(t *testing.T) {
	cases := []disCase{
		disCase{"hello", "hello", "-", []int{}},
		disCase{"whatdayisit?", "what-day-is-it-?", "-", []int{4, 7, 9, 11}},
		disCase{"whatdayisit?", "what day is it?", " ", []int{4, 7, 9}},
		disCase{"whatdayisit?", "what day is it ?", " ", []int{4, 7, 9, 11}},
		disCase{"hi", "hi", "-", []int{4, 7, 9, 11}},
		disCase{"whatdayisit", "-what-day-is-it", "-", []int{0, 4, 7, 9, 11}},
		disCase{"whatdayisit", "what-day-is-it", "-", []int{4, 7, 9, 11, 13}},
		disCase{"****************1088", "****-****-**-******1088", "-", []int{4, 8, 10}},
	}
	for i, c := range cases {
		m := map[string]interface{}{"s": c.sym, "l": c.locs}
		if res, err := InsertSymbol(pongo2.AsValue(c.in), pongo2.AsValue(m)); err != nil || c.expected != res.String() {
			t.Errorf("Case[%d]: expected: %s got: %s", i, c.expected, res.String())
		}
	}
}

func TestEscapeEntities(t *testing.T) {
	assert := assert.New(t)

	helperTestFilter(assert, EscapeEntities, "Jack&Jill", "", "Jack&amp;Jill", "Ampersand should be escaped")
	helperTestFilter(assert, EscapeEntities, "Jack & Jill", "", "Jack&nbsp;&amp;&nbsp;Jill", "Ampersand should be escaped and space")
}
