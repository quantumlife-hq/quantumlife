// Package proactive implements proactive recommendation and nudge systems.
package proactive

import (
	"encoding/json"
	"strings"
)

// encodeJSON converts a value to JSON string
func encodeJSON(v interface{}) (string, error) {
	if v == nil {
		return "{}", nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "{}", err
	}
	return string(data), nil
}

// decodeJSON parses a JSON string into a value
func decodeJSON(s string, v interface{}) error {
	if s == "" || s == "{}" || s == "[]" {
		return nil
	}
	return json.Unmarshal([]byte(s), v)
}

// sanitizeID creates a safe ID from a string
func sanitizeID(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, byte(c))
		}
	}
	if len(result) > 20 {
		result = result[:20]
	}
	return strings.ToLower(string(result))
}
