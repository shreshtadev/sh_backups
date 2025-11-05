package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DetailItem represents an individual error object in the 'detail' array.
type DetailItem struct {
	Type  string   `json:"type"`
	Loc   []string `json:"loc"`
	Msg   string   `json:"msg"`
	Input any      `json:"input"`
}

// APIError is the top-level structure for the JSON error response.
type APIError struct {
	Detail interface{} `json:"detail"` // Use interface{} to hold the unmarshaled value
}

// ErrorResponse is the core structure for unmarshaling the HTTP body.
type ErrorResponse struct {
	// RawDetail holds the raw JSON bytes of the 'detail' field.
	// We use json.RawMessage to delay unmarshaling this specific field.
	RawDetail json.RawMessage `json:"detail"`
	Detail    string          // This will hold the final, formatted error message.
}

// UnmarshalJSON implements the json.Unmarshaler interface, handling array or string details.
func (e *ErrorResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	rawDetail, ok := raw["detail"]
	if !ok {
		e.Detail = "Unknown error format: missing 'detail' field"
		return nil
	}

	// 1. Attempt to unmarshal as a string
	var detailString string
	if err := json.Unmarshal(rawDetail, &detailString); err == nil {
		e.Detail = detailString
		return nil
	}

	// 2. Attempt to unmarshal as an array of DetailItem
	var detailArray []DetailItem
	if err := json.Unmarshal(rawDetail, &detailArray); err == nil {
		var messages []string
		for _, item := range detailArray {
			messages = append(messages, fmt.Sprintf("Type: %s, Location: %s, Message: %s",
				item.Type,
				strings.Join(item.Loc, "->"),
				item.Msg,
			))
		}
		e.Detail = strings.Join(messages, "; ")
		return nil
	}

	// 3. Fallback
	e.Detail = fmt.Sprintf("Unrecognized detail format: %s", string(rawDetail))
	return nil
}

// --------------------------------------------------------------------------

// ParseErrorBody takes the HTTP status and the raw JSON body and returns the formatted error message.
func ParseErrorBody(status string, bodyBytes []byte) string {
	var errResp ErrorResponse

	// Unmarshal the flexible error response
	if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
		// If unmarshaling the whole structure fails, return the raw body as a string
		return fmt.Sprintf("HTTP status %s, failed to parse error JSON: %s", status, string(bodyBytes))
	}

	// Return the cleanly formatted error message
	return fmt.Sprintf("HTTP Error %s: %s", status, errResp.Detail)
}
