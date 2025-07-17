package engine

import (
	"encoding/json"
	"fmt"
)

// LogAsJSON takes any data structure, marshals it to indented JSON, and prints it.
// This is a generic utility for logging different configuration types.
func LogAsJSON(v interface{}, prefix string) error {
	jsonData, err := json.MarshalIndent(v, prefix, "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}
