package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// BoolOrStringType represents the stored type of BoolOrString.
type BoolOrStringType int64

const (
	// Bool indicates that the BoolOrString holds a boolean.
	Bool BoolOrStringType = iota
	// String indicates that the BoolOrString holds a string.
	String
)

// +kubebuilder:validation:Schemaless
// +kubebuilder:pruning:PreserveUnknownFields
// BoolOrString is a type that can hold either a bool or a string.
type BoolOrString struct {
	Type    BoolOrStringType `json:"-"`
	BoolVal bool             `json:"-"`
	StrVal  string           `json:"-"`
}

// FromBool creates a BoolOrString object with a bool value.
func FromBool(val bool) BoolOrString {
	return BoolOrString{Type: Bool, BoolVal: val}
}

// FromString creates a BoolOrString object with a string value.
func FromString(val string) BoolOrString {
	return BoolOrString{Type: String, StrVal: val}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It accepts both JSON booleans and JSON strings.
func (bs *BoolOrString) UnmarshalJSON(value []byte) error {
	if len(value) == 0 {
		return fmt.Errorf("empty value")
	}
	// Check if the value is a quoted string.
	if value[0] == '"' {
		bs.Type = String
		return json.Unmarshal(value, &bs.StrVal)
	}
	// Attempt to unmarshal as a boolean.
	var boolVal bool
	if err := json.Unmarshal(value, &boolVal); err == nil {
		bs.Type = Bool
		bs.BoolVal = boolVal
		return nil
	}
	// As a fallback, try unmarshaling as a string (for unquoted strings, if any).
	var strVal string
	if err := json.Unmarshal(value, &strVal); err == nil {
		bs.Type = String
		bs.StrVal = strVal
		return nil
	}
	return fmt.Errorf("BoolOrString must be a boolean or a string")
}

// MarshalJSON implements the json.Marshaler interface.
func (bs BoolOrString) MarshalJSON() ([]byte, error) {
	switch bs.Type {
	case Bool:
		return json.Marshal(bs.BoolVal)
	case String:
		return json.Marshal(bs.StrVal)
	default:
		return nil, fmt.Errorf("invalid BoolOrString type")
	}
}

// String returns the string representation of the stored value.
func (bs *BoolOrString) String() string {
	if bs == nil {
		return "<nil>"
	}
	if bs.Type == String {
		return bs.StrVal
	}
	return strconv.FormatBool(bs.BoolVal)
}

// OpenAPISchemaType is used by the kube-openapi generator.
func (BoolOrString) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator.
func (BoolOrString) OpenAPISchemaFormat() string { return "bool-or-string" }

// OpenAPIV3OneOfTypes is used by the kube-openapi generator to indicate the union types.
func (BoolOrString) OpenAPIV3OneOfTypes() []string { return []string{"boolean", "string"} }
