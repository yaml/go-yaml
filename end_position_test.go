package yaml_test

import (
	"bytes"
	"fmt"
	"testing"

	yaml "go.yaml.in/yaml/v4"
)

// The assertions here are overwhelmingly "this node spans exactly here", so
// they collapse into three helpers rather than repeating an if-block per field.

func checkKind(t *testing.T, name string, n *yaml.Node, want yaml.Kind) {
	t.Helper()
	if n.Kind != want {
		t.Errorf("%s.Kind = %v, want %v", name, n.Kind, want)
	}
}

func checkStart(t *testing.T, name string, n *yaml.Node, line, column int) {
	t.Helper()
	if n.Line != line || n.Column != column {
		t.Errorf("%s starts at %d:%d, want %d:%d", name, n.Line, n.Column, line, column)
	}
}

func checkEnd(t *testing.T, name string, n *yaml.Node, endLine, endColumn int) {
	t.Helper()
	if n.EndLine != endLine || n.EndColumn != endColumn {
		t.Errorf("%s ends at %d:%d, want %d:%d", name, n.EndLine, n.EndColumn, endLine, endColumn)
	}
}

// findKey returns the value node for the given key path in a decoded mapping.
func findKey(t *testing.T, n *yaml.Node, path ...string) *yaml.Node {
	t.Helper()
	cur := n
	if cur.Kind == yaml.DocumentNode {
		cur = cur.Content[0]
	}
	for _, key := range path {
		if cur.Kind != yaml.MappingNode {
			t.Fatalf("node for key %q is not a mapping", key)
		}
		var next *yaml.Node
		for i := 0; i+1 < len(cur.Content); i += 2 {
			if cur.Content[i].Value == key {
				next = cur.Content[i+1]
				break
			}
		}
		if next == nil {
			t.Fatalf("key %q not found", key)
		}
		cur = next
	}
	return cur
}

// TestNodeEndPosition covers all four node kinds. The end position is recorded
// on two code paths: node() sets it for scalars and aliases (from the event's
// end_mark), while mapping() and sequence() derive it from their last child.
// Across all kinds the end means the same thing -- the position just past the
// last content character -- rather than the start of the following line, which
// is where the collection END token sits after a block dedent.
func TestNodeEndPosition(t *testing.T) {
	//  1: mapping:
	//  2:   a: 1
	//  3:   b: 2
	//  4: sequence:
	//  5:   - first
	//  6:   - second
	//  7: base: &anchor
	//  8:   key: value
	//  9: aliased: *anchor
	// 10: plain: text
	src := `mapping:
  a: 1
  b: 2
sequence:
  - first
  - second
base: &anchor
  key: value
aliased: *anchor
plain: text
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	// Mapping (last-child path): starts at its first entry (line 2, col 3) and
	// ends just past its last value `2` (line 3, col 7) -- the end of the actual
	// content, not the start of the following line.
	mapping := findKey(t, &doc, "mapping")
	checkKind(t, "mapping", mapping, yaml.MappingNode)
	checkStart(t, "mapping", mapping, 2, 3)
	checkEnd(t, "mapping", mapping, 3, 7) // Sequence (last-child path): ends just past its last item `second`
	// (line 6, col 11).
	seq := findKey(t, &doc, "sequence")
	checkKind(t, "seq", seq, yaml.SequenceNode)
	checkStart(t, "seq", seq, 5, 3)
	checkEnd(t, "seq", seq, 6, 11) // Alias (node() path): a reference is a single token, so it starts and ends
	// on the same line -- and its end is the alias token's (cols 10-17), not the
	// span of the anchored mapping it points at.
	alias := findKey(t, &doc, "aliased")
	checkKind(t, "alias", alias, yaml.AliasNode)
	checkStart(t, "alias", alias, 9, 10)
	checkEnd(t, "alias", alias, 9, 17) // Scalar (node() path): starts and ends on the same line (cols 8-12).
	scalar := findKey(t, &doc, "plain")
	checkKind(t, "scalar", scalar, yaml.ScalarNode)
	checkStart(t, "scalar", scalar, 10, 8)
	checkEnd(t, "scalar", scalar, 10, 12)
}

// TestNodeEndPositionLastInDoc covers a collection that is the last element in
// the document: it ends at EOF rather than at a dedent to a following sibling.
// The end must still be its last child's end, not an overshoot to the line
// past the last content -- and it must hold with or without a trailing newline.
func TestNodeEndPositionLastInDoc(t *testing.T) {
	// 1: top: 1
	// 2: block:
	// 3:   x: 10
	// 4:   y: 20
	src := `top: 1
block:
  x: 10
  y: 20
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	block := findKey(t, &doc, "block")
	checkKind(t, "block", block, yaml.MappingNode)
	checkEnd(t, "block", block, 4, 8) // last content line, not the post-EOF line 5 just past `20`

	// The document's root mapping ends where its last child does.
	root := doc.Content[0]
	checkEnd(t, "root", root, 4, 8) // A trailing sequence ends at its last item, again without overshooting EOF.
	var doc2 yaml.Node
	if err := yaml.Unmarshal([]byte("top: 1\nlist:\n  - a\n  - bb\n"), &doc2); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}
	list := findKey(t, &doc2, "list")
	checkKind(t, "list", list, yaml.SequenceNode)
	checkEnd(t, "list", list, 4, 7) // just past `bb`

	// Robust to a missing trailing newline on the final element.
	var doc3 yaml.Node
	if err := yaml.Unmarshal([]byte("top: 1\nblock:\n  x: 10\n  y: 20"), &doc3); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}
	block3 := findKey(t, &doc3, "block")
	checkEnd(t, "block3", block3, 4, 8)
}

// TestNodeEndPositionEmpty covers the fallback for empty collections: with no
// last child to borrow an end from, mapping()/sequence() fall back to the
// END-event mark, which for a flow `{}`/`[]` is just past the closing delimiter.
func TestNodeEndPositionEmpty(t *testing.T) {
	// 1: emptyMap: {}
	// 2: emptySeq: []
	src := `emptyMap: {}
emptySeq: []
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	em := findKey(t, &doc, "emptyMap")
	checkKind(t, "em", em, yaml.MappingNode)
	if len(em.Content) != 0 {
		t.Errorf("len(em.Content) = %d, want 0", len(em.Content))
	}
	if em.Line != 1 {
		t.Errorf("em.Line = %d, want 1", em.Line)
	}
	checkEnd(t, "em", em, 1, 13) // just past `}` (cols 11-12)

	es := findKey(t, &doc, "emptySeq")
	checkKind(t, "es", es, yaml.SequenceNode)
	if len(es.Content) != 0 {
		t.Errorf("len(es.Content) = %d, want 0", len(es.Content))
	}
	if es.Line != 2 {
		t.Errorf("es.Line = %d, want 2", es.Line)
	}
	checkEnd(t, "es", es, 2, 13) // just past `]` (cols 11-12)
}

// TestNodeEndPositionBlockScalar covers literal (|) and folded (>) block
// scalars: multi-line values whose end must land just past the last content
// line, not on the `|`/`>` indicator line and not overshooting to the next key.
func TestNodeEndPositionBlockScalar(t *testing.T) {
	//  1: literal: |
	//  2:   one
	//  3:   two
	//  4: folded: >
	//  5:   alpha
	//  6:   beta
	//  7: after: end
	src := `literal: |
  one
  two
folded: >
  alpha
  beta
after: end
`
	// Block scalars are leaves with no child to borrow an end from, and libyaml's
	// scalar end_mark sits past the trailing line break (at the start of the next
	// line). The scanner captures the position just past the last content
	// character instead, so the span ends on the last content line rather than
	// bleeding into the following node (which matters for any schema ending in a
	// multi-line `description: |`).
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	lit := findKey(t, &doc, "literal")
	checkKind(t, "lit", lit, yaml.ScalarNode)
	if lit.Value != "one\ntwo\n" {
		t.Errorf("lit.Value = %q, want %q", lit.Value, "one\ntwo\n")
	}
	checkEnd(t, "lit", lit, 3, 6) // last content line (`two`) just past `two` (cols 3-5)

	fold := findKey(t, &doc, "folded")
	checkKind(t, "fold", fold, yaml.ScalarNode)
	if fold.Value != "alpha beta\n" {
		t.Errorf("fold.Value = %q, want %q", fold.Value, "alpha beta\n")
	}
	checkEnd(t, "fold", fold, 6, 7) // last content line (`beta`) just past `beta` (cols 3-6)
}

// TestNodeEndPositionTrailingComment verifies a comment does not extend a
// node's end: neither a same-line trailing comment nor a standalone comment
// line after a block becomes part of the preceding node's span.
func TestNodeEndPositionTrailingComment(t *testing.T) {
	//  1: scalar: value  # trailing
	//  2: block:
	//  3:   a: 1
	//  4:   b: 2
	//  5: # standalone comment
	//  6: after: x
	src := `scalar: value  # trailing
block:
  a: 1
  b: 2
# standalone comment
after: x
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	sc := findKey(t, &doc, "scalar")
	checkKind(t, "sc", sc, yaml.ScalarNode)
	checkEnd(t, "sc", sc, 1, 14) // just past `value` (cols 9-13), not the comment

	block := findKey(t, &doc, "block")
	checkKind(t, "block", block, yaml.MappingNode)
	checkEnd(t, "block", block, 4, 7) // last content `b: 2`, not the standalone comment on line 5 just past `2`
}

// TestNodeEndPositionMultiDoc verifies end positions are correct in the second
// document of a stream, where every mark sits at a non-zero line offset.
func TestNodeEndPositionMultiDoc(t *testing.T) {
	//  1: ---
	//  2: a: 1
	//  3: ---
	//  4: c: 3
	//  5: d: 44
	src := "---\na: 1\n---\nc: 3\nd: 44\n"
	dec := yaml.NewDecoder(bytes.NewBufferString(src))
	var d1, d2 yaml.Node
	if err := dec.Decode(&d1); err != nil {
		t.Fatalf("Decode() returned error: %v", err)
	}
	if err := dec.Decode(&d2); err != nil {
		t.Fatalf("Decode() returned error: %v", err)
	}

	root2 := d2.Content[0]
	checkKind(t, "root2", root2, yaml.MappingNode)
	checkEnd(t, "root2", root2, 5, 6) // last content `d: 44` in the 2nd doc just past `44` (cols 4-5)
}

// TestNodeEndPositionMergeKey verifies a mapping using a merge key (<<) ends at
// its own last content, and the merge value (an alias) carries the alias span,
// not the span of the anchored mapping it points at.
func TestNodeEndPositionMergeKey(t *testing.T) {
	//  1: base: &b
	//  2:   x: 1
	//  3:   y: 2
	//  4: merged:
	//  5:   <<: *b
	//  6:   z: 3
	src := `base: &b
  x: 1
  y: 2
merged:
  <<: *b
  z: 3
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	merged := findKey(t, &doc, "merged")
	checkKind(t, "merged", merged, yaml.MappingNode)
	checkEnd(t, "merged", merged, 6, 7) // last content `z: 3` just past `3`

	// merged.Content is [`<<` key, `*b` alias, `z` key, `3` value]; the merge
	// value is the alias on line 5.
	mergeAlias := merged.Content[1]
	checkKind(t, "mergeAlias", mergeAlias, yaml.AliasNode)
	checkEnd(t, "mergeAlias", mergeAlias, 5, 9) // just past `*b` (cols 7-8)
}

// TestNodeEndPositionFlowCollections covers non-empty flow `{}`/`[]`: the end is
// just past the closing delimiter, on the same line.
func TestNodeEndPositionFlowCollections(t *testing.T) {
	// 1: flowMap: {a: 1, b: 2}
	// 2: flowSeq: [10, 20, 30]
	src := `flowMap: {a: 1, b: 2}
flowSeq: [10, 20, 30]
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	// Flow collections end just past their closing delimiter, so the span covers
	// the whole `{...}`/`[...]`. mapping()/sequence() use the END-event mark for
	// flow style (the delimiter is explicit) rather than the last child, which
	// keeps non-empty and empty flow collections consistent.
	fm := findKey(t, &doc, "flowMap")
	checkKind(t, "fm", fm, yaml.MappingNode)
	checkEnd(t, "fm", fm, 1, 22) // just past `}` (col 21)

	fs := findKey(t, &doc, "flowSeq")
	checkKind(t, "fs", fs, yaml.SequenceNode)
	checkEnd(t, "fs", fs, 2, 22) // just past `]` (col 21)
}

// TestNodeEndPositionQuotedScalars covers single- and double-quoted scalars:
// the end is just past the closing quote, not the last content character.
func TestNodeEndPositionQuotedScalars(t *testing.T) {
	// 1: single: 'hello'
	// 2: double: "world"
	src := `single: 'hello'
double: "world"
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	sq := findKey(t, &doc, "single")
	checkKind(t, "sq", sq, yaml.ScalarNode)
	if sq.Value != "hello" {
		t.Errorf("sq.Value = %q, want %q", sq.Value, "hello")
	}
	checkEnd(t, "sq", sq, 1, 16) // just past closing `'` (col 15)

	dq := findKey(t, &doc, "double")
	checkKind(t, "dq", dq, yaml.ScalarNode)
	if dq.Value != "world" {
		t.Errorf("dq.Value = %q, want %q", dq.Value, "world")
	}
	checkEnd(t, "dq", dq, 2, 16) // just past closing `"` (col 15)
}

// TestNodeEndPositionSeqOfMappings covers a block sequence whose items are
// mappings: each item ends at its own last value, and the sequence ends at its
// last item -- neither bleeds into the next item or a following sibling.
func TestNodeEndPositionSeqOfMappings(t *testing.T) {
	// 1: items:
	// 2:   - a: 1
	// 3:     b: 2
	// 4:   - c: 3
	// 5: after: x
	src := `items:
  - a: 1
    b: 2
  - c: 3
after: x
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	items := findKey(t, &doc, "items")
	checkKind(t, "items", items, yaml.SequenceNode)
	if len(items.Content) != 2 {
		t.Errorf("len(items.Content) = %d, want 2", len(items.Content))
	}

	item0 := items.Content[0]
	checkKind(t, "item0", item0, yaml.MappingNode)
	checkEnd(t, "item0", item0, 3, 9) // last value `2` just past `2`

	item1 := items.Content[1]
	checkKind(t, "item1", item1, yaml.MappingNode)
	checkEnd(t, "item1", item1, 4, 9) // last value `3` just past `3`

	// The sequence ends where its last item does, not on the `after` line.
	checkEnd(t, "items", items, 4, 9)
}

// TestNodeEndPositionDeepNesting verifies the end of the innermost leaf
// propagates up to every enclosing mapping.
func TestNodeEndPositionDeepNesting(t *testing.T) {
	// 1: l1:
	// 2:   l2:
	// 3:     l3:
	// 4:       leaf: 1
	src := `l1:
  l2:
    l3:
      leaf: 1
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	for _, path := range [][]string{{"l1"}, {"l1", "l2"}, {"l1", "l2", "l3"}} {
		n := findKey(t, &doc, path...)
		checkKind(t, "n", n, yaml.MappingNode)
		// leaf line, and just past `1`
		checkEnd(t, fmt.Sprintf("node at path %v", path), n, 4, 14)
	}
}

// TestNodeEndPositionBlockScalarChompingAndEOF exercises the block-scalar end
// fix under strip (|-) and keep (|+) chomping, and when the block scalar is the
// last content in the document (ends at EOF, must not overshoot past it).
func TestNodeEndPositionBlockScalarChompingAndEOF(t *testing.T) {
	// 1: strip: |-
	// 2:   a
	// 3:   b
	// 4: tail: |
	// 5:   x
	// 6:   y
	src := `strip: |-
  a
  b
tail: |
  x
  y
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatalf("Unmarshal() returned error: %v", err)
	}

	strip := findKey(t, &doc, "strip")
	checkKind(t, "strip", strip, yaml.ScalarNode)
	if strip.Value != "a\nb" {
		t.Errorf("strip.Value = %q, want %q", strip.Value, "a\nb")
	} // strip: no trailing newline
	checkEnd(t, "strip", strip, 3, 4) // last content line `b` just past `b` (col 3)

	// `tail` is the last node in the document: its end is its last content line
	// (`y` on line 6), not an overshoot to the post-EOF line 7.
	tail := findKey(t, &doc, "tail")
	checkKind(t, "tail", tail, yaml.ScalarNode)
	checkEnd(t, "tail", tail, 6, 4) // just past `y` (col 3)

	// The root mapping inherits the block scalar's end, so it must not overshoot.
	root := doc.Content[0]
	checkEnd(t, "root", root, 6, 4)
}
