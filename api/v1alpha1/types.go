package v1alpha1

import (
    "encoding/json"
    "fmt"
    "strconv"

    "k8s.io/apimachinery/pkg/util/intstr"
)

// BoolOrString is a custom type that can hold either a boolean or a string value.
// This is modeled similarly to k8s.io/apimachinery/pkg/util/intstr.IntOrString but for booleans.
//
// Example usages:
//   filebrowser: true
//   filebrowser: "true"
//   filebrowser: "some-placeholder"
type BoolOrString struct {
	Type    ValueType `json:"-"`
	BoolVal bool      `json:"-"`
	StrVal  string    `json:"-"`
}

// ValueType indicates whether BoolOrString holds a boolean or a string.
type ValueType int

const (
    Bool ValueType = iota
    String
)

// UnmarshalJSON allows BoolOrString to accept either `true`/`false` (bool) or `"someString"` in JSON.
func (b *BoolOrString) UnmarshalJSON(data []byte) error {
    // 1) Try parsing as bool
    var boolVal bool
    if err := json.Unmarshal(data, &boolVal); err == nil {
        b.Type = Bool
        b.BoolVal = boolVal
        return nil
    }

    // 2) Otherwise, parse as string
    var strVal string
    if err := json.Unmarshal(data, &strVal); err == nil {
        b.Type = String
        b.StrVal = strVal
        return nil
    }

    return fmt.Errorf("BoolOrString: invalid input %q: must be bool or string", string(data))
}

// MarshalJSON emits a JSON bool if we hold a bool, otherwise a JSON string.
func (b BoolOrString) MarshalJSON() ([]byte, error) {
    switch b.Type {
    case Bool:
        return json.Marshal(b.BoolVal)
    case String:
        return json.Marshal(b.StrVal)
    }
    return []byte("null"), nil
}

// AsBool tries to convert the underlying data to a real bool. If it's a bool, we use that.
// If it's a string, we attempt to parse it (e.g. "true", "false", "1", etc.).
func (b BoolOrString) AsBool() (bool, error) {
    switch b.Type {
    case Bool:
        return b.BoolVal, nil
    case String:
        parsed, err := strconv.ParseBool(b.StrVal)
        if err != nil {
            return false, fmt.Errorf("cannot parse string %q as bool: %w", b.StrVal, err)
        }
        return parsed, nil
    }
    return false, fmt.Errorf("unknown type for BoolOrString")
}

// IsString tells you if this field was originally defined as a string in the YAML.
func (b BoolOrString) IsString() bool {
    return b.Type == String
}

// IsBool tells you if this field was originally defined as a bool in the YAML.
func (b BoolOrString) IsBool() bool {
    return b.Type == Bool
}

// IntOrString is re-exported for convenience from "k8s.io/apimachinery/pkg/util/intstr".
// This type can hold either an integer or a string.
// e.g. containerPort: 25565  or  containerPort: "25565"
type IntOrString = intstr.IntOrString
