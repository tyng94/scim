package scim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// NewSchemaFromFile reads the file from given filepath and returns a validated schema if no errors take place.
func NewSchemaFromFile(filepath string) (Schema, error) {
	raw, err := ioutil.ReadFile(filepath)
	if err != nil {
		return Schema{}, err
	}

	return NewSchemaFromBytes(raw)
}

// NewSchemaFromString returns a validated schema if no errors take place.
func NewSchemaFromString(s string) (Schema, error) {
	return NewSchemaFromBytes([]byte(s))
}

// NewSchemaFromBytes returns a validated schema if no errors take place.
func NewSchemaFromBytes(raw []byte) (Schema, error) {
	_, err := metaSchema.validate(raw, read)
	if err != nil {
		return Schema{}, err
	}

	var schema schema
	json.Unmarshal(raw, &schema)

	return Schema{schema}, nil
}

// Schema specifies the defined attribute(s) and their characteristics (mutability, returnability, etc).
type Schema struct {
	schema schema
}

// schema specifies the defined attribute(s) and their characteristics (mutability, returnability, etc). For every
// schema URI used in a resource object, there is a corresponding "Schema" resource.
//
// RFC: RFC7643 - https://tools.ietf.org/html/rfc7643#section-7
type schema struct {
	// ID is the unique URI of the schema. REQUIRED.
	ID string
	// Name is the schema's human-readable name. OPTIONAL.
	Name string
	// Description is the schema's human-readable description.  OPTIONAL.
	Description string
	// Attributes is a collection of a complex type that defines service provider attributes and their qualities.
	Attributes attributes
}

// validate validates given bytes based on the schema and validation mode.
func (s schema) validate(raw []byte, mode validationMode) (CoreAttributes, error) {
	var m interface{}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()

	err := d.Decode(&m)
	if err != nil {
		return CoreAttributes{}, err
	}
	return s.Attributes.validate(m, mode)
}

// attribute is a complex type that defines service provider attributes and their qualities via the following set of
// sub-attributes.
//
// RFC: https://tools.ietf.org/html/rfc7643#section-7
type attribute struct {
	// Name is the attribute's name.
	Name string
	// Type is the attribute's data type. Valid values are "string", "boolean", "decimal", "integer", "dateTime",
	// "reference", and "complex".  When an attribute is of type "complex", there SHOULD be a corresponding schema
	// attribute "subAttributes" defined, listing the sub-attributes of the attribute.
	Type attributeType
	// SubAttributes defines a set of sub-attributes when an attribute is of type "complex". "subAttributes" has the
	// same schema sub-attributes as "attributes".
	SubAttributes attributes
	// MultiValued is a boolean value indicating the attribute's plurality.
	MultiValued bool
	// Description is the attribute's human-readable description. When applicable, service providers MUST specify the
	// description.
	Description string
	// Required is a boolean value that specifies whether or not the attribute is required.
	Required bool
	// CanonicalValues is a collection of suggested canonical values that MAY be used (e.g., "work" and "home").
	// OPTIONAL.
	CanonicalValues []string
	// CaseExact is a boolean value that specifies whether or not a string attribute is case sensitive.
	CaseExact bool
	// Mutability is a single keyword indicating the circumstances under which the value of the attribute can be
	// (re)defined.
	Mutability attributeMutability
	// Returned is a single keyword that indicates when an attribute and associated values are returned in response to
	// a GET request or in response to a PUT, POST, or PATCH request.
	Returned attributeReturned
	// Uniqueness is a single keyword value that specifies how the service provider enforces uniqueness of attribute
	// values.
	Uniqueness attributeUniqueness
	// ReferenceTypes is a multi-valued array of JSON strings that indicate the SCIM resource types that may be
	// referenced.
	ReferenceTypes []string
}

func (a attribute) validate(i interface{}, mode validationMode) (CoreAttributes, error) {
	// validate required
	if i == nil {
		if a.Required {
			return CoreAttributes{}, fmt.Errorf("cannot find required value %s", strings.ToLower(a.Name))
		}
		return CoreAttributes{}, nil
	}

	if a.MultiValued {
		arr, ok := i.([]interface{})
		if !ok {
			return CoreAttributes{}, fmt.Errorf("cannot convert %v to a slice", i)
		}

		// empty array = omitted/nil
		if len(arr) == 0 && a.Required {
			return CoreAttributes{}, fmt.Errorf("required array is empty")
		}

		coreAttributes := make([]CoreAttributes, 0)
		for _, sub := range arr {
			attributes, err := a.validateSingular(sub, mode)
			if err != nil {
				return CoreAttributes{}, err
			}
			coreAttributes = append(coreAttributes, attributes)
		}

		if mode != read {
			return CoreAttributes{a.Name: coreAttributes}, nil
		}
		return CoreAttributes{}, nil
	}

	return a.validateSingular(i, mode)
}

func (a attribute) validateSingular(i interface{}, mode validationMode) (CoreAttributes, error) {
	if mode == replace {
		switch a.Mutability {
		case attributeMutabilityImmutable:
			return CoreAttributes{}, fmt.Errorf("immutable field: %s", a.Name)
		case attributeMutabilityReadOnly:
			return CoreAttributes{}, nil
		}
	}

	switch a.Type {
	case attributeTypeBoolean:
		_, ok := i.(bool)
		if !ok {
			return CoreAttributes{}, fmt.Errorf("cannot convert %v to type %s", i, a.Type)
		}
	case attributeTypeComplex:
		if _, err := a.SubAttributes.validate(i, mode); err != nil {
			return CoreAttributes{}, err
		}
	case attributeTypeString, attributeTypeReference:
		_, ok := i.(string)
		if !ok {
			return CoreAttributes{}, fmt.Errorf("cannot convert %v to type %s", i, a.Type)
		}
	case attributeTypeInteger:
		n, ok := i.(json.Number)
		if !ok {
			return CoreAttributes{}, fmt.Errorf("cannot convert %v to a json.Number", i)
		}
		if strings.Contains(n.String(), ".") || strings.Contains(n.String(), "e") {
			return CoreAttributes{}, fmt.Errorf("%s is not an integer value", n)
		}
	default:
		return CoreAttributes{}, fmt.Errorf("not implemented/invalid type: %v", a.Type)
	}

	if mode != read && (a.Returned == attributeReturnedAlways || a.Returned == attributeReturnedDefault) {
		return CoreAttributes{a.Name: i}, nil
	}
	return CoreAttributes{}, nil
}

type attributes []attribute

func (as attributes) validate(i interface{}, mode validationMode) (CoreAttributes, error) {
	coreAttributes := make(CoreAttributes)

	c, ok := i.(map[string]interface{})
	if !ok {
		return CoreAttributes{}, fmt.Errorf("cannot convert %v to type complex", i)
	}

	for _, attribute := range as {
		// validate duplicate
		var hit interface{}
		var found bool
		for k, v := range c {
			if strings.EqualFold(attribute.Name, k) {
				if found {
					return CoreAttributes{}, fmt.Errorf("duplicate key: %s", strings.ToLower(k))
				}
				found = true
				hit = v
			}
		}

		attribute, err := attribute.validate(hit, mode)
		if err != nil {
			return CoreAttributes{}, err
		}

		if mode != read {
			for k, v := range attribute {
				coreAttributes[k] = v
			}
		}
	}
	return coreAttributes, nil
}

type attributeType string

const (
	attributeTypeBinary    attributeType = "binary"
	attributeTypeBoolean                 = "boolean"
	attributeTypeComplex                 = "complex"
	attributeTypeDateTime                = "dateTime"
	attributeTypeDecimal                 = "decimal"
	attributeTypeInteger                 = "integer"
	attributeTypeReference               = "reference"
	attributeTypeString                  = "string"
)

type attributeMutability string

const (
	attributeMutabilityImmutable attributeMutability = "immutable"
	attributeMutabilityReadOnly                      = "readOnly"
	attributeMutabilityReadWrite                     = "readWrite"
	attributeMutabilityWriteOnly                     = "writeOnly"
)

type attributeReturned string

const (
	attributeReturnedAlways  attributeReturned = "always"
	attributeReturnedDefault                   = "default"
	attributeReturnedNever                     = "never"
	attributeReturnedRequest                   = "request"
)

type attributeUniqueness string

const (
	attributeUniquenessGlobal attributeUniqueness = "global"
	attributeUniquenessNone                       = "none"
	attributeUniquenessServer                     = "server"
)

type validationMode int

const (
	// read will validate required and the type, but does not return core attributes.
	read validationMode = iota
	// write will validate required, returnability and the type.
	write
	// replace will validate required, mutability, returnability and type.
	replace
)

var metaSchema schema

func init() {
	if err := json.Unmarshal([]byte(rawMetaSchema), &metaSchema); err != nil {
		panic(err)
	}
}
