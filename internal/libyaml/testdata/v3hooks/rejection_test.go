// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v3hooks_test

import (
	"errors"
	"strings"
	"testing"

	v3 "go.yaml.in/yaml/v3"
	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/datatest"
	gopkgv3 "gopkg.in/yaml.v3"
)

type V3 struct {
	Value string
}

func (*V3) UnmarshalYAML(*v3.Node) error {
	panic("v3 hook must not run")
}

type GopkgV3 struct {
	Value string
}

func (*GopkgV3) UnmarshalYAML(*gopkgv3.Node) error {
	panic("gopkg v3 hook must not run")
}

type nested[T any] struct {
	Item T
}

type inline[T any] struct {
	Item T `yaml:",inline"`
}

type aliased[T any] struct {
	Base any
	Item T
}

func destination[T any](shape string) any {
	switch shape {
	case "direct":
		return new(T)
	case "pointer":
		return new(*T)
	case "nested":
		return new(nested[T])
	case "nested-pointer":
		return new(nested[*T])
	case "slice":
		return new([]T)
	case "map":
		return new(map[string]T)
	case "inline":
		return new(inline[T])
	case "inline-pointer":
		return new(inline[*T])
	case "deep-inline":
		return new(inline[inline[T]])
	case "alias":
		return new(aliased[T])
	default:
		panic("unknown destination shape: " + shape)
	}
}

func TestV3HookRejection(t *testing.T) {
	cases, err := datatest.LoadTestCasesFromFile("rejection.yaml",
		func(in []byte) (any, error) {
			var out any
			err := yaml.Load(in, &out)
			return out, err
		})
	if err != nil {
		t.Fatal(err)
	}
	apis := map[string]func([]byte, any) error{
		"Load": func(in []byte, out any) error { return yaml.Load(in, out) },
		"V2": func(in []byte, out any) error {
			return yaml.Load(in, out, yaml.WithV2Defaults())
		},
		"V3": func(in []byte, out any) error {
			return yaml.Load(in, out, yaml.WithV3Defaults())
		},
		"V4": func(in []byte, out any) error {
			return yaml.Load(in, out, yaml.WithV4Defaults())
		},
		"Unmarshal": yaml.Unmarshal,
		"Loader": func(in []byte, out any) error {
			loader, err := yaml.NewLoader(strings.NewReader(string(in)))
			if err != nil {
				return err
			}
			return loader.Load(out)
		},
		"Decoder": func(in []byte, out any) error {
			return yaml.NewDecoder(strings.NewReader(string(in))).Decode(out)
		},
		"Node.Load": func(in []byte, out any) error {
			var n yaml.Node
			if err := yaml.Load(in, &n); err != nil {
				return err
			}
			return n.Load(out)
		},
		"Node.Decode": func(in []byte, out any) error {
			var n yaml.Node
			if err := yaml.Load(in, &n); err != nil {
				return err
			}
			return n.Decode(out)
		},
	}
	for _, tc := range cases {
		for pkg, dest := range map[string]func(string) any{
			"go.yaml.in/yaml/v3": destination[V3],
			"gopkg.in/yaml.v3":   destination[GopkgV3],
		} {
			for name, load := range apis {
				t.Run(tc["name"].(string)+"/"+pkg+"/"+name,
					func(t *testing.T) {
						err := load([]byte(tc["yaml"].(string)),
							dest(tc["shape"].(string)))
						if err == nil {
							t.Fatal("expected migration error")
						}
						for _, text := range []string{
							"UnmarshalYAML(*" + pkg + ".Node)",
							"v4 cannot invoke v3 Node hooks",
							"*go.yaml.in/yaml/v4.Node",
							"UnmarshalYAML(func(any) error)",
						} {
							if !strings.Contains(err.Error(), text) {
								t.Errorf("error %q missing %q", err, text)
							}
						}
						var detail *yaml.LoadError
						if !errors.As(err, &detail) {
							t.Fatalf("expected LoadError: %v", err)
						}
						if detail.Stage != yaml.ConstructorStage ||
							detail.Mark.Line != tc["line"].(int) ||
							detail.Mark.Column <= 0 {
							t.Errorf("unexpected stage or position: %+v", detail)
						}
					})
			}
		}
	}
}

type wrongSignature struct {
	Value string
}

func (*wrongSignature) UnmarshalYAML(*v3.Node) bool {
	panic("wrong signature must not run")
}

func TestWrongSignatureIsOrdinaryStruct(t *testing.T) {
	var out wrongSignature
	if err := yaml.Load([]byte("value: input"), &out); err != nil {
		t.Fatal(err)
	}
	if out.Value != "input" {
		t.Fatalf("unexpected destination: %+v", out)
	}
}

func TestNullDoesNotInvokeV3Hooks(t *testing.T) {
	for name, out := range map[string]any{
		"v3": new(V3), "gopkg": new(GopkgV3),
	} {
		t.Run(name, func(t *testing.T) {
			if err := yaml.Load([]byte("null"), out); err != nil {
				t.Fatal(err)
			}
		})
	}
}
