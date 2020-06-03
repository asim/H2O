package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"

	jsonschema "github.com/hailo-platform/H2O/go-jsonschema"
	"github.com/hailo-platform/H2O/platform/errors"
	jsonschemaproto "github.com/hailo-platform/H2O/platform/proto/jsonschema"
)

// jsonschemaHandler returns all registered endpoints in json schema format as per ITF draft4
// http://json-schema.org/latest/json-schema-core.html
func jsonschemaHandler(req *Request) (proto.Message, errors.Error) {
	// Get all endpoints
	request := req.Data().(*jsonschemaproto.Request)
	endpoint := request.GetEndpoint()

	endpoints := reg.iterate()
	schemas := make([]*jsonschema.JsonSchema, 0)
	for _, ep := range endpoints {
		if endpoint != "" && endpoint != ep.GetName() {
			continue
		}
		schema, err := marshalEndpoint(ep)
		if err == nil && schema != nil {
			schemas = append(schemas, schema)
		}
	}

	rsp, err := json.Marshal(schemas)
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.kernel.marshal.error", fmt.Sprintf("Unable to unmarshal response data: %v", err.Error()))
	}

	return &jsonschemaproto.Response{
		Jsonschema: proto.String(string(rsp)),
	}, nil
}

// marshalEndpoint marshals a service endpoint into a json schema
func marshalEndpoint(ep *Endpoint) (*jsonschema.JsonSchema, error) {
	reqT, resT := ep.ProtoTypes()
	if reqT == nil && resT == nil {
		return nil, nil
	}

	// Generate the root schema for this endpoint
	rootSchema := jsonschema.New()
	rootSchema.Id = fmt.Sprintf("http://directory.hailoweb.com/service/%s-%v.json#%s", Name, Version, ep.GetName())
	rootSchema.Title = fmt.Sprintf("%s-%v.%s", Name, Version, ep.GetName())
	rootSchema.Description = fmt.Sprintf("%s endpoint schema for %s-%v", ep.GetName(), Name, Version)
	rootSchema.Type = jsonschema.TYPE_OBJECT
	rootSchema.Schema = "http://json-schema.org/draft-04/schema#"

	if reqT != nil {
		// Generate the request skeleton
		reqSchema := jsonschema.New()
		reqSchema.Property = "Request"
		reqSchema.Parent = rootSchema

		if err := reflectEndpoint(reqT, reqSchema); err != nil {
			return nil, err
		}

		if len(reqSchema.Properties) > 0 || len(reqSchema.Definitions) > 0 {
			reqSchema.Type = jsonschema.TYPE_OBJECT
		}

		rootSchema.AddProperty(reqSchema)

	}

	if resT != nil {
		// Generate the response skeleton
		resSchema := jsonschema.New()
		resSchema.Property = "Response"
		resSchema.Parent = rootSchema

		if err := reflectEndpoint(resT, resSchema); err != nil {
			return nil, err
		}
		if len(resSchema.Properties) > 0 || len(resSchema.Definitions) > 0 {
			resSchema.Type = jsonschema.TYPE_OBJECT
		}

		rootSchema.AddProperty(resSchema)
	}

	// All definitions go on the root level
	definitions := make(map[string]*jsonschema.JsonSchema, 20)
	for _, child := range rootSchema.DefinitionsChildren {
		definitions[child.Property] = child
	}
	rootSchema.Definitions = definitions

	return rootSchema, nil
}

// reflectEndpoint traverses a given endpoint and returns a predefined json schema
// based on its data
func reflectEndpoint(typ reflect.Type, schema *jsonschema.JsonSchema) error {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	// Iterate over not built-in types
	if typ.PkgPath() != "" && typ.Kind() == reflect.Struct {
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			tag := f.Tag.Get("protobuf")

			// If there are no proto tags, we assume the field is unused
			if tag == "" {
				continue
			}

			// Parse the proto tags
			priority, propertyName, _, enum := parseProtoTag(tag)

			// Create a property wrapper
			prop := jsonschema.New()
			prop.Property = propertyName
			prop.Parent = schema

			var (
				items   *jsonschema.JsonSchema
				isSlice bool
			)

			typ := f.Type
			// Check if we have a slice
			if typ.Kind() == reflect.Slice {
				typ = typ.Elem()
				isSlice = true
				items = jsonschema.New()
				items.Parent = prop
			}

			// Check if we have a ptr and get the underlying type
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}

			// Add this to the required fields on this level if necessary
			if priority == "req" {
				if err := schema.AddRequired(propertyName); err != nil {
					return err
				}
			}

			// Deal with enums
			if enum != "" {
				enumVals := getProtoEnumValues(typ)

				if isSlice {
					items.Type = jsonschema.TYPE_STRING
					items.Enum = enumVals
					prop.Items = items
					prop.Type = jsonschema.TYPE_ARRAY
				} else {
					prop.Type = jsonschema.TYPE_STRING
					prop.Enum = enumVals
				}
				if err := schema.AddProperty(prop); err != nil {
					return err
				}
				continue
			}

			// Deal with custom types
			if typ.PkgPath() != "" {
				// If this was a slice, create a new property and add the ref as a child
				// Otherwise, add the reference as a normal property
				if isSlice {
					items.Ref = fmt.Sprintf("#/definitions/%s", typ.String())
					prop.Items = items
					prop.Type = jsonschema.TYPE_ARRAY
					if err := schema.AddProperty(prop); err != nil {
						return err
					}

				} else {
					ref := jsonschema.New()
					ref.Ref = fmt.Sprintf("#/definitions/%s", typ.String())
					ref.Parent = schema
					ref.Property = propertyName
					if err := schema.AddProperty(ref); err != nil {
						return err
					}
				}
				// Create a new definition for this object and iterate it
				// NOTE: We always put definitions at root level
				root := findRoot(schema)
				if root == nil {
					root = schema
				}

				def := jsonschema.New()
				def.Property = typ.String()

				def.Parent = root
				def.Type = jsonschema.TYPE_OBJECT
				root.AddDefinitionChild(def)
				reflectEndpoint(typ, def)
				continue
			}

			// We are dealing with a builtin type
			switch typ.Kind() {
			case reflect.String:
				prop.Type = jsonschema.TYPE_STRING

			case reflect.Bool:
				prop.Type = jsonschema.TYPE_BOOLEAN

			case reflect.Int, reflect.Int16, reflect.Int8, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
				prop.Type = jsonschema.TYPE_INTEGER

			case reflect.Float32, reflect.Float64, reflect.Complex128, reflect.Complex64:
				prop.Type = jsonschema.TYPE_NUMBER
			default:
				// We should never get here
				log.Errorf("Unhandled type: %v", typ.Kind())
				continue
			}
			// If we had a slice on this iteration, use the prop type for the nested items scheme
			if isSlice {
				items.Type = prop.Type
				prop.Type = jsonschema.TYPE_ARRAY
				prop.Items = items
			}
			if err := schema.AddProperty(prop); err != nil {
				return err
			}

		}
	}

	return nil
}

// getProtoEnumValues travers an enum type and returns a list of strings
func getProtoEnumValues(t reflect.Type) []string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	values := make([]string, 0)
	for i := 0; ; i++ {
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

// parseProtoTag inspects a proto tag and returns priority, name, default or enum values if found
func parseProtoTag(tag string) (priority string, name string, def string, enum string) {
	arr := strings.Split(tag, ",")
	for i := range arr {
		if i == 2 {
			priority = arr[2]
		}
		if i > 2 {
			parts := strings.Split(arr[i], "=")
			if len(parts) == 2 {
				switch parts[0] {
				case "name":
					name = parts[1]
				case "def":
					def = parts[1]
				case "enum":
					enum = parts[1]
				}
			}
		}

	}

	return
}

// findRoot finds the root node in the linked list
func findRoot(schema *jsonschema.JsonSchema) *jsonschema.JsonSchema {
	// The sole purpose of max depth is to guard against infinite loops
	// Under notmal circumstances it should never be reached
	maxDepth := 100
	s := schema
	for n := 0; n < maxDepth; n++ {
		if s.Parent == nil {
			return s
		}
		s = s.Parent
	}
	log.Errorf("Infinite loop trying to find %s root node", schema.Property)
	return nil
}
