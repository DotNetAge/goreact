package common

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractJSON extracts the first JSON object from a string.
// It finds the first '{' and last '}' to extract a JSON object.
// Returns the extracted JSON string or an error if no JSON is found.
func ExtractJSON(response string) (string, error) {
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return "", fmt.Errorf("no JSON found in response")
	}

	return response[jsonStart : jsonEnd+1], nil
}

// ExtractJSONArray extracts the first JSON array from a string.
// It finds the first '[' and last ']' to extract a JSON array.
// Returns the extracted JSON string or an error if no JSON array is found.
func ExtractJSONArray(response string) (string, error) {
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return "", fmt.Errorf("no JSON array found in response")
	}

	return response[jsonStart : jsonEnd+1], nil
}

// ParseJSONObject extracts and parses a JSON object from a string into the provided target.
// This is a convenience function that combines ExtractJSON and json.Unmarshal.
func ParseJSONObject(response string, target any) error {
	jsonStr, err := ExtractJSON(response)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

// ParseJSONArray extracts and parses a JSON array from a string into the provided target.
// This is a convenience function that combines ExtractJSONArray and json.Unmarshal.
func ParseJSONArray(response string, target any) error {
	jsonStr, err := ExtractJSONArray(response)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("failed to parse JSON array: %w", err)
	}

	return nil
}

// MustExtractJSON is like ExtractJSON but panics on error.
// Use this only in tests or when you are certain the response contains JSON.
func MustExtractJSON(response string) string {
	jsonStr, err := ExtractJSON(response)
	if err != nil {
		panic(err)
	}
	return jsonStr
}
