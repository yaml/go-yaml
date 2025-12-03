package yaml

import (
	"encoding/json"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

type TestUnmarshaler struct {
	Value string
	Array []int
	Map   map[string]int
}

func (t *TestUnmarshaler) UnmarshalJSON(data []byte) error {
	type Alias TestUnmarshaler
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = TestUnmarshaler(aux)
	return nil
}

func Test_unmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  string
	}{
		{
			name: "map with string keys",
			input: map[string]any{
				"Value": "hello",
				"Array": []int{1, 2, 3},
				"Map":   map[string]int{"a": 1, "b": 2},
			},
			expected: &TestUnmarshaler{
				Value: "hello",
				Array: []int{1, 2, 3},
				Map:   map[string]int{"a": 1, "b": 2},
			},
		},
		{
			name: "map with int keys",
			input: map[string]any{
				"Value": "hello",
				"Array": []int{1, 2, 3},
				"Map":   map[int]int{1: 1, 2: 2},
			},
			expected: &TestUnmarshaler{
				Value: "hello",
				Array: []int{1, 2, 3},
				Map:   map[string]int{"1": 1, "2": 2},
			},
		},
		{
			name: "map with any keys",
			input: map[string]any{
				"Value": "hello",
				"Array": []int{1, 2, 3},
				"Map":   map[any]int{1: 1, "b": 2, true: 3},
			},
			expected: &TestUnmarshaler{
				Value: "hello",
				Array: []int{1, 2, 3},
				Map:   map[string]int{"1": 1, "b": 2, "true": 3},
			},
		},
		{
			name: "map with duplicate keys",
			input: map[string]any{
				"Value": "hello",
				"Array": []int{1, 2, 3},
				"Map":   map[any]int{1: 1, "1": 2},
			},
			wantErr: `duplicate key "1" found when converting to JSON object`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			target := &TestUnmarshaler{}
			err := unmarshalJSON(tt.input, target)
			if tt.wantErr != "" {
				assert.ErrorMatchesf(t, tt.wantErr, err, "unmarshalJSON() error")
				return
			}
			assert.NoError(t, err)
			assert.DeepEqualf(t, tt.expected, target, "unmarshalJSON() result")
		})
	}
}
