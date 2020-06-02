package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/HailoOSS/gojsonschema"
)

type FieldType struct {
	name  string
	kinds []reflect.Kind
}

func (field *FieldType) String() string {
	return field.name
}

func NewFieldType(name string, kinds ...reflect.Kind) *FieldType {
	return &FieldType{
		name:  name,
		kinds: kinds,
	}
}

var (
	String    = NewFieldType("string", reflect.String, reflect.Interface)
	ProtoEnum = NewFieldType("string")
	Enum      = NewFieldType("string")
	Number    = NewFieldType("number",
		reflect.Int32, reflect.Uint32, reflect.Int64,
		reflect.Uint64, reflect.Float32, reflect.Float64,
		reflect.Int16, reflect.Int32, reflect.Uint16,
		reflect.Uint32, reflect.Int8, reflect.Uint8,
		reflect.Int, reflect.Uint,
	)
	Map     = NewFieldType("map", reflect.Map)
	Object  = NewFieldType("object", reflect.Struct)
	Array   = NewFieldType("array", reflect.Array, reflect.Slice)
	Boolean = NewFieldType("boolean", reflect.Bool)
	Null    = NewFieldType("null")

	FieldTypes = []*FieldType{String, Enum, ProtoEnum, Number, Map, Object, Array, Boolean, Null}
)

type Format string

const (
	FormatTabs  = Format("tabs")
	FormatTable = Format("table")
)

func (fieldType *FieldType) MarshalJSON() ([]byte, error) {
	return json.Marshal(fieldType.String())
}

func (fieldType *FieldType) UnmarshalJSON(data []byte) error {
	for _, possibleType := range FieldTypes {
		possibleJson, _ := possibleType.MarshalJSON()

		if string(possibleJson) == string(data) {
			*fieldType = *possibleType
			return nil
		}
	}

	return fmt.Errorf("Unknown field type: %s", string(data))
}

type Value struct {
	Type                 *FieldType        `json:"type"`
	Properties           map[string]*Value `json:"properties,omitempty"`
	Required             bool              `json:"required"`
	Items                *Value            `json:"items,omitempty"`
	Enum                 []string          `json:"enum,omitempty"`
	Title                string            `json:"title"`
	PropertyOrder        int               `json:"propertyOrder,omitempty"`
	Format               Format            `json:"format,omitempty"`
	Description          string            `json:"description,omitempty"`
	HeaderTemplate       string            `json:"headerTemplate,omitempty"`
	ReadOnly             bool              `json:"readOnly,omitempty"`
	AdditionalProperties bool              `json:"additionalProperties,omitempty"`
}

type Schema struct {
	Schema *Value `json:"schema"`
}

func (schema *Schema) String() string {
	schemaJson, _ := json.Marshal(schema)
	return string(schemaJson)
}

type JsonSchema struct {
	Type                 *FieldType             `json:"type"`
	Properties           map[string]*JsonSchema `json:"properties,omitempty"`
	Items                *JsonSchema            `json:"items,omitempty"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

// change map type which is used in json-editor and is not standard in json schema to object with additionalProperties.
func fixMaps(value *JsonSchema) {
	if value == nil {
		return
	} else if value.Type.String() == "map" {
		value.Type = Object
		value.AdditionalProperties = true
		value.Items = nil
		value.Properties = nil
	} else {
		for _, property := range value.Properties {
			fixMaps(property)
		}
		fixMaps(value.Items)
	}
}

func (schema *Schema) Validate(config string) error {
	// convert the schema to json
	schemaBytes, err := json.Marshal(schema.Schema)
	if err != nil {
		return err
	}

	// convert the json to basic schema, to strip unecessary fields
	jsonSchema := &JsonSchema{}
	err = json.Unmarshal(schemaBytes, jsonSchema)
	if err != nil {
		return err
	}

	// fix the maps type
	fixMaps(jsonSchema)

	// convert the basic schema to json
	schemaBytes, err = json.Marshal(jsonSchema)

	// contruct validator
	loader := gojsonschema.NewStringLoader(string(schemaBytes))
	schemaDocument, err := gojsonschema.NewSchema(loader)
	if err != nil {
		return err
	}

	// validate
	result, err := schemaDocument.Validate(gojsonschema.NewStringLoader(config))

	if err != nil {
		return err
	}

	// check for errors
	if result.Valid() {
		return nil
	} else {
		errors := "Errors validating the config : "
		// Loop through errors
		for _, desc := range result.Errors() {
			errors += desc.String() + "; "
		}

		return fmt.Errorf(errors)
	}
}
