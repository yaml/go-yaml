// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"errors"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v4"
)

type migrationFunc func(any, *yaml.Node) (bool, error)

func (f migrationFunc) TryUnmarshalYAML(target any, node *yaml.Node) (bool, error) {
	return f(target, node)
}

type migrationTarget struct {
	Value string
}

type nativeMigrationTarget struct {
	Value string
}

func (v *nativeMigrationTarget) UnmarshalYAML(n *yaml.Node) error {
	v.Value = "native: " + n.Value
	return nil
}

type portableMigrationTarget struct {
	Value string
}

func (v *portableMigrationTarget) UnmarshalYAML(decode func(any) error) error {
	return decode(&v.Value)
}

func TestV4MigrationDispatch(t *testing.T) {
	sentinel := errors.New("migration failed")
	for _, tc := range []struct {
		name      string
		handled   bool
		err       error
		wantCalls []string
	}{
		{"handled", true, nil, []string{"decline", "handle"}},
		{"declined error", false, sentinel, []string{"decline", "handle"}},
		{"handled error", true, sentinel, []string{"decline", "handle"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			decline := migrationFunc(func(any, *yaml.Node) (bool, error) {
				calls = append(calls, "decline")
				return false, nil
			})
			handle := migrationFunc(func(target any, n *yaml.Node) (bool, error) {
				calls = append(calls, "handle")
				if tc.err == nil {
					target.(*migrationTarget).Value = n.Value
				}
				return tc.handled, tc.err
			})
			unreachable := migrationFunc(func(any, *yaml.Node) (bool, error) {
				t.Fatal("dispatch continued after a result")
				return false, nil
			})
			var out migrationTarget
			err := yaml.Load([]byte("hello"), &out,
				yaml.WithPlugin(decline, handle, unreachable))
			if !errors.Is(err, tc.err) {
				t.Fatalf("got %v, want %v", err, tc.err)
			}
			if !reflect.DeepEqual(calls, tc.wantCalls) {
				t.Fatalf("calls: %v", calls)
			}
			if tc.err == nil && out.Value != "hello" {
				t.Fatalf("value: %q", out.Value)
			}
		})
	}
}

func TestV4MigrationPreservesNativeHooks(t *testing.T) {
	p := migrationFunc(func(target any, _ *yaml.Node) (bool, error) {
		switch target.(type) {
		case *nativeMigrationTarget, *portableMigrationTarget:
			t.Fatal("migration overrode a supported hook")
		}
		return false, nil
	})
	var native nativeMigrationTarget
	if err := yaml.Load([]byte("hello"), &native, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	if native.Value != "native: hello" {
		t.Fatal(native.Value)
	}
	var portable portableMigrationTarget
	if err := yaml.Load([]byte("hello"), &portable, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	if portable.Value != "hello" {
		t.Fatal(portable.Value)
	}
}

func TestV4MigrationDeclinesToOrdinaryConstruction(t *testing.T) {
	p := migrationFunc(func(any, *yaml.Node) (bool, error) { return false, nil })
	out := migrationTarget{Value: "before"}
	if err := yaml.Load([]byte("value: after"), &out, yaml.WithPlugin(p)); err != nil {
		t.Fatal(err)
	}
	if out.Value != "after" {
		t.Fatal(out)
	}
}
