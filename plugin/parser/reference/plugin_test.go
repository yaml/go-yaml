package reference_test

import (
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/plugin/parser/reference"
)

func TestPlugin(t *testing.T) {
	var got map[string]any
	err := yaml.Load([]byte("{a: [1, true]}\n"), &got,
		yaml.WithPlugin(reference.New()))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"a": []any{1, true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
