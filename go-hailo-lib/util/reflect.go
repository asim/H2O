package util

import (
	"reflect"
)

func FieldNamesAsByteSlice(x interface{}) [][]byte {
	typ := reflect.TypeOf(x)
	if typ.Kind() != reflect.Struct {
		return [][]byte{}
	}

	fieldNames := make([][]byte, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		var fieldName string
		if field.Tag.Get("name") != "" {
			fieldName = field.Tag.Get("name")
		} else if field.Tag.Get("key") != "" {
			fieldName = field.Tag.Get("key")
		} else {
			fieldName = field.Name
		}

		fieldNames[i] = []byte(fieldName)
	}
	return fieldNames
}
