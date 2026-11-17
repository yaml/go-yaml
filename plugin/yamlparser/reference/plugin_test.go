package reference_test

import (
	"reflect"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/go-yaml/plugin/yamlparser/reference"
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
