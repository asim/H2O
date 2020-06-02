package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	BoolField   bool           `json:"bool_field"`
	StringField string         `json:"string_field"`
	IntField    int64          `json:"int_field"`
	FloatField  float64        `json:"float_field"`
	StructField TestStruct2    `json:"struct_field"`
	StringArray []string       `json:"string_arr"`
	StructArray []*TestStruct2 `json:"struct_array"`
	IntPtr      *int32         `json:"int_ptr"`
	NoJson      int
}

type TestStruct2 struct {
	Field1 string  `json:"field1"`
	Field2 *string `json:"field2"`
}

type TestStruct3 struct {
	TestStruct2
	AnotherField string      `json:"another"`
	Nested       TestStruct2 `json:"nested"`
}

type TestMapType map[string]string

type TagsStruct struct {
	EnumField string            `json:"enumField" enum:"Value1,Value2,Value3" description:"testing enums"`
	Array     []string          `json:"arr" format:"table" headerTemplate:"{{i}}"`
	Map       map[string]string `json:"map"`
}

func TestTags(t *testing.T) {
	schema := Of(&TagsStruct{})
	value, _ := json.MarshalIndent(schema, "", "    ")
	t.Log(string(value))

	value, _ = json.MarshalIndent(schema, "\t\t", "    ")
	assert.Equal(t, string(value), `{
		    "schema": {
		        "type": "object",
		        "properties": {
		            "arr": {
		                "type": "array",
		                "required": true,
		                "items": {
		                    "type": "string",
		                    "required": true,
		                    "title": "",
		                    "format": "table",
		                    "headerTemplate": "{{i}}"
		                },
		                "title": "arr",
		                "propertyOrder": 2,
		                "format": "table"
		            },
		            "enumField": {
		                "type": "string",
		                "required": true,
		                "enum": [
		                    "Value1",
		                    "Value2",
		                    "Value3"
		                ],
		                "title": "enumField",
		                "propertyOrder": 1,
		                "description": "testing enums"
		            },
		            "map": {
		                "type": "map",
		                "required": true,
		                "items": {
		                    "type": "object",
		                    "properties": {
		                        "key": {
		                            "type": "string",
		                            "required": true,
		                            "title": "key"
		                        },
		                        "value": {
		                            "type": "string",
		                            "required": true,
		                            "title": "value"
		                        }
		                    },
		                    "required": false,
		                    "title": "",
		                    "headerTemplate": "{{self.key}}"
		                },
		                "title": "",
		                "propertyOrder": 3,
		                "format": "tabs"
		            }
		        },
		        "required": true,
		        "title": ""
		    }
		}`)
}

func TestInterfaceSchema(t *testing.T) {
	var config struct {
		Field interface{}
	}

	schema := Of(config)
	value, _ := json.MarshalIndent(schema, "", "    ")
	t.Log(string(value))

	value, _ = json.MarshalIndent(schema, "\t\t", "    ")
	assert.Equal(t, string(value), `{
		    "schema": {
		        "type": "object",
		        "properties": {
		            "Field": {
		                "type": "string",
		                "required": true,
		                "title": "Field",
		                "propertyOrder": 1
		            }
		        },
		        "required": true,
		        "title": ""
		    }
		}`)
}

func TestMapSchema(t *testing.T) {
	schema := Of(TestMapType{})
	value, _ := json.MarshalIndent(schema, "", "    ")
	t.Log(string(value))

	value, _ = json.MarshalIndent(schema, "\t\t", "    ")
	assert.Equal(t, string(value), `{
		    "schema": {
		        "type": "map",
		        "required": true,
		        "items": {
		            "type": "object",
		            "properties": {
		                "key": {
		                    "type": "string",
		                    "required": true,
		                    "title": "key"
		                },
		                "value": {
		                    "type": "string",
		                    "required": true,
		                    "title": "value"
		                }
		            },
		            "required": false,
		            "title": "",
		            "headerTemplate": "{{self.key}}"
		        },
		        "title": "TestMapType",
		        "format": "tabs"
		    }
		}`)

	err := schema.Validate(`{
		"key":"value",
		"key2":"value2"
	}`)

	assert.Nil(t, err)

	err = schema.Validate(`[]`)
	assert.NotNil(t, err)
}

func TestEmbeddedFields(t *testing.T) {
	schema := Of(TestStruct3{})
	value, _ := json.MarshalIndent(schema, "", "    ")
	t.Log(string(value))

	value, _ = json.MarshalIndent(schema, "\t\t", "    ")
	assert.Equal(t, string(value), `{
		    "schema": {
		        "type": "object",
		        "properties": {
		            "another": {
		                "type": "string",
		                "required": true,
		                "title": "another",
		                "propertyOrder": 2
		            },
		            "field1": {
		                "type": "string",
		                "required": true,
		                "title": "field1",
		                "propertyOrder": 1
		            },
		            "field2": {
		                "type": "string",
		                "required": true,
		                "title": "field2",
		                "propertyOrder": 2
		            },
		            "nested": {
		                "type": "object",
		                "properties": {
		                    "field1": {
		                        "type": "string",
		                        "required": true,
		                        "title": "field1",
		                        "propertyOrder": 1
		                    },
		                    "field2": {
		                        "type": "string",
		                        "required": true,
		                        "title": "field2",
		                        "propertyOrder": 2
		                    }
		                },
		                "required": true,
		                "title": "nested",
		                "propertyOrder": 3
		            }
		        },
		        "required": true,
		        "title": ""
		    }
		}`)
}

func TestSchemaMarshalling(t *testing.T) {
	schema := Of(TestStruct3{})
	schemaJson := schema.String()

	schema2 := &Schema{}
	err := json.Unmarshal([]byte(schemaJson), schema2)
	assert.Nil(t, err)

	assert.Equal(t, schema.String(), schema2.String())
}

type EnumMap struct {
	Value map[string]string `json:"value" enum:"1,2,3" valueEnum:"4,5,6"`
}

func TestEnumMap(t *testing.T) {
	schema := Of(&EnumMap{})
	value, _ := json.MarshalIndent(schema, "", "    ")
	t.Log(string(value))

	value, _ = json.MarshalIndent(schema, "\t\t", "    ")
	assert.Equal(t, string(value), `{
		    "schema": {
		        "type": "object",
		        "properties": {
		            "value": {
		                "type": "map",
		                "required": true,
		                "items": {
		                    "type": "object",
		                    "properties": {
		                        "key": {
		                            "type": "string",
		                            "required": true,
		                            "enum": [
		                                "1",
		                                "2",
		                                "3"
		                            ],
		                            "title": "key"
		                        },
		                        "value": {
		                            "type": "string",
		                            "required": true,
		                            "enum": [
		                                "4",
		                                "5",
		                                "6"
		                            ],
		                            "title": "value"
		                        }
		                    },
		                    "required": false,
		                    "title": "",
		                    "headerTemplate": "{{self.key}}"
		                },
		                "title": "",
		                "propertyOrder": 1,
		                "format": "tabs"
		            }
		        },
		        "required": true,
		        "title": ""
		    }
		}`)
}

type EnumMap2 struct {
	Value map[string]string `json:"value" valueEnum:"4,5,6"`
}

func TestEnumMap2(t *testing.T) {
	schema := Of(&EnumMap2{})
	value, _ := json.MarshalIndent(schema, "", "    ")
	t.Log(string(value))

	value, _ = json.MarshalIndent(schema, "\t\t", "    ")
	assert.Equal(t, string(value), `{
		    "schema": {
		        "type": "object",
		        "properties": {
		            "value": {
		                "type": "map",
		                "required": true,
		                "items": {
		                    "type": "object",
		                    "properties": {
		                        "key": {
		                            "type": "string",
		                            "required": true,
		                            "title": "key"
		                        },
		                        "value": {
		                            "type": "string",
		                            "required": true,
		                            "enum": [
		                                "4",
		                                "5",
		                                "6"
		                            ],
		                            "title": "value"
		                        }
		                    },
		                    "required": false,
		                    "title": "",
		                    "headerTemplate": "{{self.key}}"
		                },
		                "title": "",
		                "propertyOrder": 1,
		                "format": "tabs"
		            }
		        },
		        "required": true,
		        "title": ""
		    }
		}`)
}

type EnumMap3 struct {
	Value map[string]string `json:"value" enum:"1,2,3"`
}

func TestEnumMap3(t *testing.T) {
	schema := Of(&EnumMap3{})
	value, _ := json.MarshalIndent(schema, "", "    ")
	t.Log(string(value))

	value, _ = json.MarshalIndent(schema, "\t\t", "    ")
	assert.Equal(t, string(value), `{
		    "schema": {
		        "type": "object",
		        "properties": {
		            "value": {
		                "type": "map",
		                "required": true,
		                "items": {
		                    "type": "object",
		                    "properties": {
		                        "key": {
		                            "type": "string",
		                            "required": true,
		                            "enum": [
		                                "1",
		                                "2",
		                                "3"
		                            ],
		                            "title": "key"
		                        },
		                        "value": {
		                            "type": "string",
		                            "required": true,
		                            "title": "value"
		                        }
		                    },
		                    "required": false,
		                    "title": "",
		                    "headerTemplate": "{{self.key}}"
		                },
		                "title": "",
		                "propertyOrder": 1,
		                "format": "tabs"
		            }
		        },
		        "required": true,
		        "title": ""
		    }
		}`)
}

type nestedStruct struct {
	Key   string `json:"key"`
	Value int    `json:"value,omitempty"`
}

type validateStruct struct {
	Name string         `json:"name"`
	Meta []nestedStruct `json:"meta,omitempty"`
}

func TestValidate(t *testing.T) {
	schema := Of(&validateStruct{})

	t.Log(schema.String())

	err := schema.Validate(`{"name":"test"}`)
	assert.Nil(t, err)

	err = schema.Validate(`{"name":"123", "xxx":"yyy"}`)
	assert.NotNil(t, err)

	err = schema.Validate(`{
		"name":"test",
		"meta":[
			{
				"value": 123,
				"xxx": "yyy"
			}
		]
	}`)
	assert.NotNil(t, err)

	err = schema.Validate(`{
		"name":"test",
		"meta":[
			{
				"value": "123"
			}
		]
	}`)

	assert.NotNil(t, err)

	err = schema.Validate(`{
		"name":"test",
		"meta":[
			{
				"key": "123",
				"value": 123
			}
		]
	}`)

	assert.Nil(t, err)
}
