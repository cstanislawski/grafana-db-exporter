// File: ./internal/jsondiff/jsondiff.go
package jsondiff

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// DiffOptions contains configuration for the diffing process
type DiffOptions struct {
	// IgnorePatterns is a list of JSONPath patterns to ignore during comparison
	IgnorePatterns []string
}

// Compare compares two JSON documents, ignoring fields specified by the patterns
func Compare(a, b interface{}, opts DiffOptions) (bool, error) {
	// Convert both objects to JSON strings for comparison
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false, fmt.Errorf("failed to marshal first object: %w", err)
	}

	bJSON, err := json.Marshal(b)
	if err != nil {
		return false, fmt.Errorf("failed to marshal second object: %w", err)
	}

	// Create copies for modification
	aResult := gjson.Parse(string(aJSON))
	bResult := gjson.Parse(string(bJSON))

	// Apply ignore patterns to both objects
	aCleaned := applyIgnorePatterns(aResult, opts.IgnorePatterns)
	bCleaned := applyIgnorePatterns(bResult, opts.IgnorePatterns)

	return aCleaned.String() == bCleaned.String(), nil
}

// applyIgnorePatterns removes specified paths from the JSON object
func applyIgnorePatterns(data gjson.Result, patterns []string) gjson.Result {
	// If no patterns are specified, return the original data
	if len(patterns) == 0 {
		return data
	}

	// Convert to a map for manipulation
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(data.Raw), &obj); err != nil {
		return data
	}

	// Apply each ignore pattern
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Remove the field specified by the pattern
		removeField(obj, splitPath(pattern))
	}

	// Convert back to JSON
	result, err := json.Marshal(obj)
	if err != nil {
		return data
	}

	return gjson.Parse(string(result))
}

// splitPath splits a JSONPath pattern into segments
func splitPath(pattern string) []string {
	// Remove leading $ if present
	if strings.HasPrefix(pattern, "$.") {
		pattern = pattern[2:]
	} else if strings.HasPrefix(pattern, "$") {
		pattern = pattern[1:]
	}

	// Split on dots, handling escaped dots and brackets
	var segments []string
	current := ""
	inBracket := false

	for _, char := range pattern {
		switch char {
		case '[':
			if current != "" {
				segments = append(segments, current)
				current = ""
			}
			inBracket = true
			current += string(char)
		case ']':
			inBracket = false
			current += string(char)
			segments = append(segments, current)
			current = ""
		case '.':
			if inBracket {
				current += string(char)
			} else if current != "" {
				segments = append(segments, current)
				current = ""
			}
		default:
			current += string(char)
		}
	}

	if current != "" {
		segments = append(segments, current)
	}

	return segments
}

// removeField removes a field from an object based on the path segments
func removeField(obj map[string]interface{}, segments []string) {
	if len(segments) == 0 {
		return
	}

	segment := segments[0]
	if len(segments) == 1 {
		// Handle array index notation
		if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
			return // Skip array modifications for safety
		}
		delete(obj, segment)
		return
	}

	// Handle nested objects
	if next, ok := obj[segment].(map[string]interface{}); ok {
		removeField(next, segments[1:])
	}
}
