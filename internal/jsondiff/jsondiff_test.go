// File: ./internal/jsondiff/jsondiff_test.go
package jsondiff

import (
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name           string
		a              interface{}
		b              interface{}
		ignorePatterns []string
		want           bool
		wantErr        bool
	}{
		{
			name: "identical objects",
			a: map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			},
			b: map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			},
			ignorePatterns: nil,
			want:           true,
			wantErr:        false,
		},
		{
			name: "different objects with ignored field",
			a: map[string]interface{}{
				"field1": "value1",
				"field2": "value2",
			},
			b: map[string]interface{}{
				"field1": "value1",
				"field2": "different",
			},
			ignorePatterns: []string{"field2"},
			want:           true,
			wantErr:        false,
		},
		{
			name: "nested objects with ignored field",
			a: map[string]interface{}{
				"field1": "value1",
				"nested": map[string]interface{}{
					"subfield1": "value2",
					"subfield2": "value3",
				},
			},
			b: map[string]interface{}{
				"field1": "value1",
				"nested": map[string]interface{}{
					"subfield1": "value2",
					"subfield2": "different",
				},
			},
			ignorePatterns: []string{"nested.subfield2"},
			want:           true,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DiffOptions{
				IgnorePatterns: tt.ignorePatterns,
			}

			got, err := Compare(tt.a, tt.b, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "simple path",
			pattern:  "field1.field2",
			expected: []string{"field1", "field2"},
		},
		{
			name:     "path with array",
			pattern:  "field1[0].field2",
			expected: []string{"field1", "[0]", "field2"},
		},
		{
			name:     "path with dollar prefix",
			pattern:  "$.field1.field2",
			expected: []string{"field1", "field2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPath(tt.pattern)
			if len(got) != len(tt.expected) {
				t.Errorf("splitPath() got %v, want %v", got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("splitPath() got[%d] = %v, want %v", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
