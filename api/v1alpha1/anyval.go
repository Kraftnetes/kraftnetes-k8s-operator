package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// AnyValType is an enumeration of the underlying type stored in AnyVal.
type AnyValType int

const (
	AnyValString AnyValType = iota
	AnyValBool
	AnyValNumber
)

// +kubebuilder:validation:Schemaless
// +kubebuilder:pruning:PreserveUnknownFields
// AnyVal is a union type that can hold a boolean, number, or string.
// It allows the Default field in GameDefinitionInput to accept any of these types.
type AnyVal struct {
	// internal type indicator (not serialized)
	Type AnyValType `json:"-"`
	// Only one of these fields will be used depending on the incoming JSON.
	StrVal  string  `json:"-"`
	BoolVal bool    `json:"-"`
	NumVal  float64 `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling for AnyVal.
// It accepts JSON booleans, numbers, or strings.
func (a *AnyVal) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	// If data is a quoted string, unmarshal as string.
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		a.Type = AnyValString
		a.StrVal = s
		return nil
	}
	// Check if the value is a boolean (true or false).
	if string(data) == "true" || string(data) == "false" {
		var b bool
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		a.Type = AnyValBool
		a.BoolVal = b
		return nil
	}
	// Otherwise, try to unmarshal as a number.
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		a.Type = AnyValNumber
		a.NumVal = n
		return nil
	}
	// Fallback: try to unmarshal as a string.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.Type = AnyValString
		a.StrVal = s
		return nil
	}
	return fmt.Errorf("AnyVal must be a boolean, number, or string")
}

// MarshalJSON implements custom marshaling for AnyVal.
func (a AnyVal) MarshalJSON() ([]byte, error) {
	switch a.Type {
	case AnyValString:
		return json.Marshal(a.StrVal)
	case AnyValBool:
		return json.Marshal(a.BoolVal)
	case AnyValNumber:
		return json.Marshal(a.NumVal)
	default:
		return nil, fmt.Errorf("invalid AnyVal type")
	}
}

// String returns the string representation of the stored value.
func (a AnyVal) String() string {
	switch a.Type {
	case AnyValString:
		return a.StrVal
	case AnyValBool:
		return strconv.FormatBool(a.BoolVal)
	case AnyValNumber:
		return strconv.FormatFloat(a.NumVal, 'f', -1, 64)
	default:
		return "<invalid>"
	}
}

// OpenAPISchemaType tells the kube-openapi generator that the base type is "string".
func (AnyVal) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat provides a custom format name.
func (AnyVal) OpenAPISchemaFormat() string { return "any" }

// OpenAPIV3OneOfTypes returns the union types allowed (boolean, number, or string)
// so that the generated CRD's schema indicates this field can accept any of these types.
func (AnyVal) OpenAPIV3OneOfTypes() []string { return []string{"boolean", "number", "string"} }
