package yaml_test

import (
	"context"
	"sync"
	"testing"

	yaml "go.yaml.in/yaml/v4"
)

type ctxKey struct{}

// withCtx records the context it was decoded with, and the value it found.
type withCtx struct {
	Name string
	seen string
}

func (w *withCtx) UnmarshalYAML(ctx context.Context, n *yaml.Node) error {
	type bis withCtx
	if err := n.Decode((*bis)(w)); err != nil {
		return err
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		w.seen = v
	}
	return nil
}

func TestUnmarshalWithContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey{}, "from-caller")
	var w withCtx
	if err := yaml.UnmarshalWithContext(ctx, []byte("name: n\n"), &w); err != nil {
		t.Fatal(err)
	}
	if w.Name != "n" || w.seen != "from-caller" {
		t.Fatalf("got name=%q seen=%q", w.Name, w.seen)
	}
}

// Unmarshal still works on a type implementing only the context form; it just
// receives a background context.
func TestUnmarshalWithoutContext(t *testing.T) {
	var w withCtx
	if err := yaml.Unmarshal([]byte("name: n\n"), &w); err != nil {
		t.Fatal(err)
	}
	if w.Name != "n" || w.seen != "" {
		t.Fatalf("got name=%q seen=%q", w.Name, w.seen)
	}
}

// Each decode sees its own context, with no state shared between them.
func TestUnmarshalWithContextConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		want := string(rune('a' + i%26))
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), ctxKey{}, want)
			var w withCtx
			if err := yaml.UnmarshalWithContext(ctx, []byte("name: n\n"), &w); err != nil {
				t.Error(err)
				return
			}
			if w.seen != want {
				t.Errorf("got %q, want %q", w.seen, want)
			}
		}()
	}
	wg.Wait()
}
