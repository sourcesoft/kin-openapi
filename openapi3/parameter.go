package openapi3

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-openapi/jsonpointer"
	"github.com/sourcesoft/kin-openapi/jsoninfo"
)

type ParametersMap map[string]*ParameterRef

var _ jsonpointer.JSONPointable = (*ParametersMap)(nil)

func (p ParametersMap) JSONLookup(token string) (interface{}, error) {
	ref, ok := p[token]
	if ref == nil || ok == false {
		return nil, fmt.Errorf("object has no field %q", token)
	}

	if ref.Ref != "" {
		return &Ref{Ref: ref.Ref}, nil
	}
	return ref.Value, nil
}

// Parameters is specified by OpenAPI/Swagger 3.0 standard.
type Parameters []*ParameterRef

var _ jsonpointer.JSONPointable = (*Parameters)(nil)

func (p Parameters) JSONLookup(token string) (interface{}, error) {
	index, err := strconv.Atoi(token)
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(p) {
		return nil, fmt.Errorf("index out of bounds array[0,%d] index '%d'", len(p), index)
	}

	ref := p[index]

	if ref != nil && ref.Ref != "" {
		return &Ref{Ref: ref.Ref}, nil
	}
	return ref.Value, nil
}

func NewParameters() Parameters {
	return make(Parameters, 0, 4)
}

func (parameters Parameters) GetByInAndName(in string, name string) *Parameter {
	for _, item := range parameters {
		if v := item.Value; v != nil {
			if v.Name == name && v.In == in {
				return v
			}
		}
	}
	return nil
}

func (parameters Parameters) Validate(c context.Context) error {
	dupes := make(map[string]struct{})
	for _, item := range parameters {
		if v := item.Value; v != nil {
			key := v.In + ":" + v.Name
			if _, ok := dupes[key]; ok {
				return fmt.Errorf("more than one %q parameter has name %q", v.In, v.Name)
			}
			dupes[key] = struct{}{}
		}

		if err := item.Validate(c); err != nil {
			return err
		}
	}
	return nil
}

// Parameter is specified by OpenAPI/Swagger 3.0 standard.
type Parameter struct {
	ExtensionProps
	Name            string      `json:"name,omitempty" yaml:"name,omitempty"`
	In              string      `json:"in,omitempty" yaml:"in,omitempty"`
	Description     string      `json:"description,omitempty" yaml:"description,omitempty"`
	Style           string      `json:"style,omitempty" yaml:"style,omitempty"`
	Explode         *bool       `json:"explode,omitempty" yaml:"explode,omitempty"`
	AllowEmptyValue bool        `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	AllowReserved   bool        `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
	Deprecated      bool        `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	Required        bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema          *SchemaRef  `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example         interface{} `json:"example,omitempty" yaml:"example,omitempty"`
	Examples        Examples    `json:"examples,omitempty" yaml:"examples,omitempty"`
	Content         Content     `json:"content,omitempty" yaml:"content,omitempty"`
}

var _ jsonpointer.JSONPointable = (*Parameter)(nil)

const (
	ParameterInPath   = "path"
	ParameterInQuery  = "query"
	ParameterInHeader = "header"
	ParameterInCookie = "cookie"
)

func NewPathParameter(name string) *Parameter {
	return &Parameter{
		Name:     name,
		In:       ParameterInPath,
		Required: true,
	}
}

func NewQueryParameter(name string) *Parameter {
	return &Parameter{
		Name: name,
		In:   ParameterInQuery,
	}
}

func NewHeaderParameter(name string) *Parameter {
	return &Parameter{
		Name: name,
		In:   ParameterInHeader,
	}
}

func NewCookieParameter(name string) *Parameter {
	return &Parameter{
		Name: name,
		In:   ParameterInCookie,
	}
}

func (parameter *Parameter) WithDescription(value string) *Parameter {
	parameter.Description = value
	return parameter
}

func (parameter *Parameter) WithRequired(value bool) *Parameter {
	parameter.Required = value
	return parameter
}

func (parameter *Parameter) WithSchema(value *Schema) *Parameter {
	if value == nil {
		parameter.Schema = nil
	} else {
		parameter.Schema = &SchemaRef{
			Value: value,
		}
	}
	return parameter
}

func (parameter *Parameter) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(parameter)
}

func (parameter *Parameter) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, parameter)
}

func (parameter Parameter) JSONLookup(token string) (interface{}, error) {
	switch token {
	case "schema":
		if parameter.Schema != nil {
			if parameter.Schema.Ref != "" {
				return &Ref{Ref: parameter.Schema.Ref}, nil
			}
			return parameter.Schema.Value, nil
		}
	case "name":
		return parameter.Name, nil
	case "in":
		return parameter.In, nil
	case "description":
		return parameter.Description, nil
	case "style":
		return parameter.Style, nil
	case "explode":
		return parameter.Explode, nil
	case "allowEmptyValue":
		return parameter.AllowEmptyValue, nil
	case "allowReserved":
		return parameter.AllowReserved, nil
	case "deprecated":
		return parameter.Deprecated, nil
	case "required":
		return parameter.Required, nil
	case "example":
		return parameter.Example, nil
	case "examples":
		return parameter.Examples, nil
	case "content":
		return parameter.Content, nil
	}

	v, _, err := jsonpointer.GetForToken(parameter.ExtensionProps, token)
	return v, err
}

// SerializationMethod returns a parameter's serialization method.
// When a parameter's serialization method is not defined the method returns
// the default serialization method corresponding to a parameter's location.
func (parameter *Parameter) SerializationMethod() (*SerializationMethod, error) {
	switch parameter.In {
	case ParameterInPath, ParameterInHeader:
		style := parameter.Style
		if style == "" {
			style = SerializationSimple
		}
		explode := false
		if parameter.Explode != nil {
			explode = *parameter.Explode
		}
		return &SerializationMethod{Style: style, Explode: explode}, nil
	case ParameterInQuery, ParameterInCookie:
		style := parameter.Style
		if style == "" {
			style = SerializationForm
		}
		explode := true
		if parameter.Explode != nil {
			explode = *parameter.Explode
		}
		return &SerializationMethod{Style: style, Explode: explode}, nil
	default:
		return nil, fmt.Errorf("unexpected parameter's 'in': %q", parameter.In)
	}
}

func (parameter *Parameter) Validate(c context.Context) error {
	if parameter.Name == "" {
		return errors.New("parameter name can't be blank")
	}
	in := parameter.In
	switch in {
	case
		ParameterInPath,
		ParameterInQuery,
		ParameterInHeader,
		ParameterInCookie:
	default:
		return fmt.Errorf("parameter can't have 'in' value %q", parameter.In)
	}

	// Validate a parameter's serialization method.
	sm, err := parameter.SerializationMethod()
	if err != nil {
		return err
	}
	var smSupported bool
	switch {
	case parameter.In == ParameterInPath && sm.Style == SerializationSimple && !sm.Explode,
		parameter.In == ParameterInPath && sm.Style == SerializationSimple && sm.Explode,
		parameter.In == ParameterInPath && sm.Style == SerializationLabel && !sm.Explode,
		parameter.In == ParameterInPath && sm.Style == SerializationLabel && sm.Explode,
		parameter.In == ParameterInPath && sm.Style == SerializationMatrix && !sm.Explode,
		parameter.In == ParameterInPath && sm.Style == SerializationMatrix && sm.Explode,

		parameter.In == ParameterInQuery && sm.Style == SerializationForm && sm.Explode,
		parameter.In == ParameterInQuery && sm.Style == SerializationForm && !sm.Explode,
		parameter.In == ParameterInQuery && sm.Style == SerializationSpaceDelimited && sm.Explode,
		parameter.In == ParameterInQuery && sm.Style == SerializationSpaceDelimited && !sm.Explode,
		parameter.In == ParameterInQuery && sm.Style == SerializationPipeDelimited && sm.Explode,
		parameter.In == ParameterInQuery && sm.Style == SerializationPipeDelimited && !sm.Explode,
		parameter.In == ParameterInQuery && sm.Style == SerializationDeepObject && sm.Explode,

		parameter.In == ParameterInHeader && sm.Style == SerializationSimple && !sm.Explode,
		parameter.In == ParameterInHeader && sm.Style == SerializationSimple && sm.Explode,

		parameter.In == ParameterInCookie && sm.Style == SerializationForm && !sm.Explode,
		parameter.In == ParameterInCookie && sm.Style == SerializationForm && sm.Explode:
		smSupported = true
	}
	if !smSupported {
		e := fmt.Errorf("serialization method with style=%q and explode=%v is not supported by a %s parameter", sm.Style, sm.Explode, in)
		return fmt.Errorf("parameter %q schema is invalid: %v", parameter.Name, e)
	}

	if (parameter.Schema == nil) == (parameter.Content == nil) {
		e := errors.New("parameter must contain exactly one of content and schema")
		return fmt.Errorf("parameter %q schema is invalid: %v", parameter.Name, e)
	}
	if schema := parameter.Schema; schema != nil {
		if err := schema.Validate(c); err != nil {
			return fmt.Errorf("parameter %q schema is invalid: %v", parameter.Name, err)
		}
	}
	if content := parameter.Content; content != nil {
		if err := content.Validate(c); err != nil {
			return fmt.Errorf("parameter %q content is invalid: %v", parameter.Name, err)
		}
	}
	return nil
}
