package yaml_test

import (
	"testing"

	yaml "go.yaml.in/yaml/v4"
)

func loadMappingValue(t *testing.T, src, key string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Load([]byte(src), &doc); err != nil {
		t.Fatal(err)
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	t.Fatalf("key %q not found in %q", key, src)
	return nil
}

func pairValues(pairs [][2]*yaml.Node) [][2]string {
	out := make([][2]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, [2]string{p[0].Value, p[1].Value})
	}
	return out
}

func checkPairs(t *testing.T, node *yaml.Node, want [][2]string) {
	t.Helper()
	got := pairValues(node.Pairs())
	if len(got) != len(want) {
		t.Fatalf("Pairs: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Pairs: got %v, want %v", got, want)
		}
	}
}

// The example from the issue: the raw Content of target holds a "<<"
// key and "a"; Pairs applies the merges, with the earlier source in
// the sequence winning and the explicit key winning over both.
func TestNodePairsAppliesMergeKeys(t *testing.T) {
	src := `
base: &b {a: 1, b: 2}
over: &o {b: 20, c: 3}
target:
  <<: [*o, *b]
  a: 100
`
	target := loadMappingValue(t, src, "target")
	checkPairs(t, target, [][2]string{{"b", "20"}, {"c", "3"}, {"a", "100"}})
}

// An explicit key wins over a merged one even when it is written
// before the merge key.
func TestNodePairsExplicitKeyBeforeMerge(t *testing.T) {
	src := `
target:
  a: 100
  <<: {a: 1, b: 2}
`
	target := loadMappingValue(t, src, "target")
	checkPairs(t, target, [][2]string{{"a", "100"}, {"b", "2"}})
}

// A single mapping as the merge value, and a merge inside the merged
// mapping, are both expanded.
func TestNodePairsNestedMerge(t *testing.T) {
	src := `
inner: &i {x: 1, y: 2}
mid: &m
  <<: *i
  y: 20
  z: 30
target:
  <<: *m
  z: 300
`
	target := loadMappingValue(t, src, "target")
	checkPairs(t, target, [][2]string{{"x", "1"}, {"y", "20"}, {"z", "300"}})
}

// Pairs is nil for nodes that are not mappings.
func TestNodePairsNonMapping(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Load([]byte("- a\n- b\n"), &doc); err != nil {
		t.Fatal(err)
	}
	if pairs := doc.Content[0].Pairs(); pairs != nil {
		t.Errorf("sequence Pairs: got %v, want nil", pairs)
	}
	if pairs := doc.Content[0].Content[0].Pairs(); pairs != nil {
		t.Errorf("scalar Pairs: got %v, want nil", pairs)
	}
	var nilNode *yaml.Node
	if pairs := nilNode.Pairs(); pairs != nil {
		t.Errorf("nil Pairs: got %v, want nil", pairs)
	}
}

// A mapping that merges itself through an alias must terminate.
func TestNodePairsSelfMerge(t *testing.T) {
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "a"}
	one := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}
	mergeKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!merge", Value: "<<"}
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: m}
	m.Content = []*yaml.Node{mergeKey, alias, key, one}
	checkPairs(t, m, [][2]string{{"a", "1"}})
}
