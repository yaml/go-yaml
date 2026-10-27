package yaml_test

import (
	"testing"
	"time"

	yaml "go.yaml.in/yaml/v4"
)

// Without the option, plain scalars that look like timestamps resolve
// to time.Time when loaded into an any.
func TestImplicitTimestampsByDefault(t *testing.T) {
	var v map[string]any
	err := yaml.Load([]byte("day: 2026-01-01\nat: 2001-12-15T02:59:43.1Z\n"), &v)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v["day"].(time.Time); !ok {
		t.Errorf("day: got %T, want time.Time", v["day"])
	}
	if _, ok := v["at"].(time.Time); !ok {
		t.Errorf("at: got %T, want time.Time", v["at"])
	}
}

// With WithoutImplicitTimestamps, the same scalars stay strings, while
// an explicit !!timestamp tag still produces a time.Time.
func TestWithoutImplicitTimestamps(t *testing.T) {
	input := "day: 2026-01-01\n" +
		"at: 2001-12-15T02:59:43.1Z\n" +
		"explicit: !!timestamp 2026-01-01\n" +
		"quoted: \"2026-01-01\"\n" +
		"num: 42\n"
	var v map[string]any
	err := yaml.Load([]byte(input), &v, yaml.WithoutImplicitTimestamps())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := v["day"].(string); !ok || got != "2026-01-01" {
		t.Errorf("day: got %T %v, want string 2026-01-01", v["day"], v["day"])
	}
	if got, ok := v["at"].(string); !ok || got != "2001-12-15T02:59:43.1Z" {
		t.Errorf("at: got %T %v, want string 2001-12-15T02:59:43.1Z", v["at"], v["at"])
	}
	if _, ok := v["explicit"].(time.Time); !ok {
		t.Errorf("explicit: got %T, want time.Time", v["explicit"])
	}
	if got, ok := v["quoted"].(string); !ok || got != "2026-01-01" {
		t.Errorf("quoted: got %T %v, want string 2026-01-01", v["quoted"], v["quoted"])
	}
	if got, ok := v["num"].(int); !ok || got != 42 {
		t.Errorf("num: got %T %v, want int 42", v["num"], v["num"])
	}
}

// The option also applies when loading into a Node: the scalar's tag
// is !!str rather than !!timestamp.
func TestWithoutImplicitTimestampsNodeTag(t *testing.T) {
	var doc yaml.Node
	err := yaml.Load([]byte("day: 2026-01-01\n"), &doc, yaml.WithoutImplicitTimestamps())
	if err != nil {
		t.Fatal(err)
	}
	value := doc.Content[0].Content[1]
	if value.Tag != "!!str" {
		t.Errorf("node tag: got %q, want !!str", value.Tag)
	}

	doc = yaml.Node{}
	err = yaml.Load([]byte("day: 2026-01-01\n"), &doc)
	if err != nil {
		t.Fatal(err)
	}
	value = doc.Content[0].Content[1]
	if value.Tag != "!!timestamp" {
		t.Errorf("node tag without option: got %q, want !!timestamp", value.Tag)
	}
}

// A string field keeps working either way.
func TestWithoutImplicitTimestampsStringField(t *testing.T) {
	var v struct {
		Day string `yaml:"day"`
	}
	err := yaml.Load([]byte("day: 2026-01-01\n"), &v, yaml.WithoutImplicitTimestamps())
	if err != nil {
		t.Fatal(err)
	}
	if v.Day != "2026-01-01" {
		t.Errorf("Day: got %q, want 2026-01-01", v.Day)
	}
}
