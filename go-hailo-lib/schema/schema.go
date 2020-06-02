package schema

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	ReqiredByDefault = true
)

// Generates json-schema for struct or proto
func Of(s interface{}) *Schema {
	t := reflect.ValueOf(s).Type()

	return &Schema{Schema: typeSchema("", t, reflect.StructTag(""))}
}

func fieldType(t reflect.Type, tag reflect.StructTag) *FieldType {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if strings.Contains(tag.Get("protobuf"), "enum") {
		return ProtoEnum
	}

	if tag.Get("schema-type") != "" {
		return String
	}

	for _, fieldType := range FieldTypes {
		for _, kind := range fieldType.kinds {
			if kind == t.Kind() {
				return fieldType
			}
		}
	}

	return Null
}

func typeSchema(title string, t reflect.Type, tag reflect.StructTag) *Value {
	fieldType := fieldType(t, tag)

	isReadOnly := tag.Get("readOnly") != ""

	value := &Value{
		Title:       title,
		Type:        fieldType,
		Required:    isRequired(tag),
		Enum:        enum(tag),
		Description: tag.Get("description"),
		Format:      Format(tag.Get("format")),
		ReadOnly:    isReadOnly,
	}

	switch fieldType {
	case Map:
		value.Title = t.Name()
		value.Format = "tabs"

		// map items
		value.Items = &Value{
			Type:           Object,
			HeaderTemplate: "{{self.key}}",
			Properties: map[string]*Value{
				"key":   typeSchema("key", t.Key(), reflect.StructTag("")),
				"value": typeSchema("value", t.Elem(), reflect.StructTag("")),
			},
		}

		// the enum annotation for maps is for the key not for the map itself
		value.Items.Properties["key"].Enum = enum(tag)
		value.Enum = make([]string, 0)

		// if there is valueEnum then the value is considered enum
		if tag.Get("valueEnum") != "" {
			value.Items.Properties["value"].Enum = strings.Split(tag.Get("valueEnum"), ",")
		} else {
			value.Items.Properties["value"].Enum = make([]string, 0)
		}
	case Array:
		value.Items = typeSchema(t.Name(), t.Elem(), tag)
		value.Items.HeaderTemplate = tag.Get("headerTemplate")
	case ProtoEnum:
		value.Enum = getProtoEnumValues(t)
	case Object:
		value = structSchema(title, t, tag)
	}

	return value
}

func enum(tag reflect.StructTag) []string {
	if tag.Get("enum") != "" {
		return strings.Split(tag.Get("enum"), ",")
	}
	return make([]string, 0)
}

func structSchema(title string, t reflect.Type, tag reflect.StructTag) *Value {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	value := &Value{
		Title:                title,
		Type:                 Object,
		Required:             isRequired(tag),
		Properties:           make(map[string]*Value),
		AdditionalProperties: false,
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if isUnexported(field) {
			continue
		}

		if field.Tag.Get("hide") != "" {
			continue
		}

		jsonTag := field.Tag.Get("json")

		if jsonTag == "-" {
			continue
		}
		jsonField := field.Name

		if jsonTag != "" {
			jsonField = strings.Split(jsonTag, ",")[0]
		}

		fieldSchema := typeSchema(jsonField, field.Type, field.Tag)
		fieldSchema.PropertyOrder = i + 1

		if field.Anonymous {
			for propName, propSchema := range fieldSchema.Properties {
				value.Properties[propName] = propSchema
			}
		} else {
			value.Properties[jsonField] = fieldSchema
		}
	}

	return value
}

func isUnexported(field reflect.StructField) bool {
	return field.Name[0] >= 'a' && field.Name[0] <= 'z'
}

func isRequired(tag reflect.StructTag) bool {
	protoTag := tag.Get("protobuf")
	if strings.Contains(protoTag, "opt") {
		return false
	}
	jsonTag := tag.Get("json")
	if strings.Contains(jsonTag, "omitempty") {
		return false
	}
	return ReqiredByDefault
}

func getProtoEnumValues(t reflect.Type) []string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	values := make([]string, 0)
	for i := 1; ; i++ {
		val := reflect.New(t)
		val.Elem().SetInt(int64(i))

		result := val.MethodByName("String").Call([]reflect.Value{})
		str := result[0].String()

		if str == fmt.Sprintf("%d", i) {
			break
		}

		values = append(values, str)
	}

	return values
}
