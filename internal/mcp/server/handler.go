package server

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolBuilder helps construct tools with a fluent API
type ToolBuilder struct {
	name        string
	description string
	properties  map[string]Property
	required    []string
}

// NewTool creates a new tool builder
func NewTool(name string) *ToolBuilder {
	return &ToolBuilder{
		name:       name,
		properties: make(map[string]Property),
	}
}

// Description sets the tool description
func (b *ToolBuilder) Description(desc string) *ToolBuilder {
	b.description = desc
	return b
}

// String adds a string parameter
func (b *ToolBuilder) String(name, description string, required bool) *ToolBuilder {
	b.properties[name] = Property{Type: "string", Description: description}
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// Number adds a number parameter
func (b *ToolBuilder) Number(name, description string, required bool) *ToolBuilder {
	b.properties[name] = Property{Type: "number", Description: description}
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// Integer adds an integer parameter
func (b *ToolBuilder) Integer(name, description string, required bool) *ToolBuilder {
	b.properties[name] = Property{Type: "integer", Description: description}
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// Boolean adds a boolean parameter
func (b *ToolBuilder) Boolean(name, description string, required bool) *ToolBuilder {
	b.properties[name] = Property{Type: "boolean", Description: description}
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// Enum adds an enum (string with choices) parameter
func (b *ToolBuilder) Enum(name, description string, choices []string, required bool) *ToolBuilder {
	b.properties[name] = Property{Type: "string", Description: description, Enum: choices}
	if required {
		b.required = append(b.required, name)
	}
	return b
}

// Build creates the Tool definition
func (b *ToolBuilder) Build() Tool {
	return Tool{
		Name:        b.name,
		Description: b.description,
		InputSchema: InputSchema{
			Type:       "object",
			Properties: b.properties,
			Required:   b.required,
		},
	}
}

// Args helps extract typed arguments from JSON
type Args struct {
	raw json.RawMessage
	m   map[string]any
	err error
}

// ParseArgs parses JSON arguments into an Args helper
func ParseArgs(raw json.RawMessage) *Args {
	a := &Args{raw: raw}
	if len(raw) > 0 {
		a.err = json.Unmarshal(raw, &a.m)
	} else {
		a.m = make(map[string]any)
	}
	return a
}

// Error returns any parsing error
func (a *Args) Error() error {
	return a.err
}

// String gets a string argument
func (a *Args) String(name string) string {
	if v, ok := a.m[name].(string); ok {
		return v
	}
	return ""
}

// StringDefault gets a string argument with default
func (a *Args) StringDefault(name, def string) string {
	if v, ok := a.m[name].(string); ok {
		return v
	}
	return def
}

// Int gets an integer argument
func (a *Args) Int(name string) int {
	if v, ok := a.m[name].(float64); ok {
		return int(v)
	}
	return 0
}

// IntDefault gets an integer argument with default
func (a *Args) IntDefault(name string, def int) int {
	if v, ok := a.m[name].(float64); ok {
		return int(v)
	}
	return def
}

// Float gets a float argument
func (a *Args) Float(name string) float64 {
	if v, ok := a.m[name].(float64); ok {
		return v
	}
	return 0
}

// Bool gets a boolean argument
func (a *Args) Bool(name string) bool {
	if v, ok := a.m[name].(bool); ok {
		return v
	}
	return false
}

// BoolDefault gets a boolean argument with default
func (a *Args) BoolDefault(name string, def bool) bool {
	if v, ok := a.m[name].(bool); ok {
		return v
	}
	return def
}

// Has checks if an argument exists
func (a *Args) Has(name string) bool {
	_, ok := a.m[name]
	return ok
}

// Get returns the raw value
func (a *Args) Get(name string) any {
	return a.m[name]
}

// Unmarshal unmarshals the raw JSON into a struct
func (a *Args) Unmarshal(v any) error {
	if len(a.raw) == 0 {
		return nil
	}
	return json.Unmarshal(a.raw, v)
}

// RequireString returns a string or error if missing
func (a *Args) RequireString(name string) (string, error) {
	if v, ok := a.m[name].(string); ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("required parameter '%s' is missing or empty", name)
}

// RequireInt returns an int or error if missing
func (a *Args) RequireInt(name string) (int, error) {
	if v, ok := a.m[name].(float64); ok {
		return int(v), nil
	}
	return 0, fmt.Errorf("required parameter '%s' is missing", name)
}

// WrapHandler wraps a simple handler function into a ToolHandler
// The wrapped function receives parsed Args and returns a string result
func WrapHandler(fn func(ctx context.Context, args *Args) (string, error)) ToolHandler {
	return func(ctx context.Context, raw json.RawMessage) (*ToolResult, error) {
		args := ParseArgs(raw)
		if args.Error() != nil {
			return ErrorResult(fmt.Sprintf("Invalid arguments: %v", args.Error())), nil
		}

		result, err := fn(ctx, args)
		if err != nil {
			return ErrorResult(err.Error()), nil
		}

		return SuccessResult(result), nil
	}
}

// WrapJSONHandler wraps a handler that returns JSON data
func WrapJSONHandler[T any](fn func(ctx context.Context, args *Args) (T, error)) ToolHandler {
	return func(ctx context.Context, raw json.RawMessage) (*ToolResult, error) {
		args := ParseArgs(raw)
		if args.Error() != nil {
			return ErrorResult(fmt.Sprintf("Invalid arguments: %v", args.Error())), nil
		}

		result, err := fn(ctx, args)
		if err != nil {
			return ErrorResult(err.Error()), nil
		}

		return JSONResult(result)
	}
}

// WrapResourceHandler wraps a simple resource handler
func WrapResourceHandler(mimeType string, fn func(ctx context.Context, uri string) (string, error)) ResourceHandler {
	return func(ctx context.Context, uri string) (*ResourceContent, error) {
		text, err := fn(ctx, uri)
		if err != nil {
			return nil, err
		}
		return &ResourceContent{
			URI:      uri,
			MimeType: mimeType,
			Text:     text,
		}, nil
	}
}
