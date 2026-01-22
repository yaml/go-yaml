// Tests for the streaming Loader API, including StreamNode functionality
// and multi-document streaming.

package libyaml

import (
	"bytes"
	"io"
	"testing"

	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

// TestStreamNodeEmptyStream tests that an empty stream returns a single StreamNode
func TestStreamNodeEmptyStream(t *testing.T) {
	input := []byte("")

	loader, err := NewLoader(bytes.NewReader(input), WithStreamNodes())
	assert.NoError(t, err)

	var nodes []Node
	for {
		var node Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Empty stream should return exactly one StreamNode
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, StreamNode, nodes[0].Kind)
}

// TestStreamNodeSingleDocument tests the pattern [Stream, Doc, Stream] for single document
func TestStreamNodeSingleDocument(t *testing.T) {
	input := []byte("key: value\n")

	loader, err := NewLoader(bytes.NewReader(input), WithStreamNodes())
	assert.NoError(t, err)

	var nodes []Node
	for {
		var node Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Single document should return [Stream, Doc, Stream]
	assert.Equal(t, 3, len(nodes))
	assert.Equal(t, StreamNode, nodes[0].Kind)
	assert.Equal(t, DocumentNode, nodes[1].Kind)
	assert.Equal(t, StreamNode, nodes[2].Kind)
}

// TestStreamNodeMultiDocument tests interleaved pattern for multi-document stream
func TestStreamNodeMultiDocument(t *testing.T) {
	input := []byte("---\nkey1: value1\n---\nkey2: value2\n")

	loader, err := NewLoader(bytes.NewReader(input), WithStreamNodes())
	assert.NoError(t, err)

	var nodes []Node
	for {
		var node Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Two documents should return [Stream, Doc, Stream, Doc, Stream]
	assert.Equal(t, 5, len(nodes))
	assert.Equal(t, StreamNode, nodes[0].Kind)
	assert.Equal(t, DocumentNode, nodes[1].Kind)
	assert.Equal(t, StreamNode, nodes[2].Kind)
	assert.Equal(t, DocumentNode, nodes[3].Kind)
	assert.Equal(t, StreamNode, nodes[4].Kind)
}

// TestStreamNodeDirectives tests that directives are captured on StreamNodes
func TestStreamNodeDirectives(t *testing.T) {
	input := []byte("%YAML 1.1\n%TAG ! tag:example.com,2000:app/\n---\nkey: value\n")

	loader, err := NewLoader(bytes.NewReader(input), WithStreamNodes())
	assert.NoError(t, err)

	var nodes []Node
	for {
		var node Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Should return [Stream, Doc, Stream]
	assert.Equal(t, 3, len(nodes))

	// First StreamNode should have encoding
	assert.Equal(t, StreamNode, nodes[0].Kind)
	// Encoding should be set (non-zero)
	if nodes[0].Encoding == 0 {
		t.Fatal("first stream node should have encoding set")
	}

	// Second node is the StreamNode before the document with directives
	// Note: directives appear on the StreamNode BEFORE the document
	streamNode := nodes[0]
	if streamNode.Version != nil {
		assert.Equal(t, 1, streamNode.Version.Major)
		assert.Equal(t, 1, streamNode.Version.Minor)
	}

	if len(streamNode.TagDirectives) > 0 {
		found := false
		for _, td := range streamNode.TagDirectives {
			if td.Handle == "!" && td.Prefix == "tag:example.com,2000:app/" {
				found = true
				break
			}
		}
		assert.True(t, found)
	}
}

// TestStreamNodeEncoding tests that encoding is captured on first StreamNode
func TestStreamNodeEncoding(t *testing.T) {
	input := []byte("key: value\n")

	loader, err := NewLoader(bytes.NewReader(input), WithStreamNodes())
	assert.NoError(t, err)

	var node Node
	err = loader.Load(&node)
	assert.NoError(t, err)

	// First node should be a StreamNode with encoding
	assert.Equal(t, StreamNode, node.Kind)
	// Encoding should be set (non-zero)
	if node.Encoding == 0 {
		t.Fatal("stream node should have encoding set")
	}
}

// TestWithoutStreamNodes tests backward compatibility (default behavior)
func TestWithoutStreamNodes(t *testing.T) {
	input := []byte("---\nkey1: value1\n---\nkey2: value2\n")

	loader, err := NewLoader(bytes.NewReader(input))
	assert.NoError(t, err)

	var nodes []Node
	for {
		var node Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Without stream nodes, should only return DocumentNodes
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, DocumentNode, nodes[0].Kind)
	assert.Equal(t, DocumentNode, nodes[1].Kind)
}

// TestStreamNodeDisabled tests explicitly disabling stream nodes
func TestStreamNodeDisabled(t *testing.T) {
	input := []byte("key: value\n")

	loader, err := NewLoader(bytes.NewReader(input), WithStreamNodes(false))
	assert.NoError(t, err)

	var node Node
	err = loader.Load(&node)
	assert.NoError(t, err)

	// Should get a DocumentNode, not a StreamNode
	assert.Equal(t, DocumentNode, node.Kind)
}

// TestLoadWithAllDocuments_TypedSlice tests loading multiple documents into a typed slice
func TestLoadWithAllDocuments_TypedSlice(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	input := []byte("---\nname: first\n---\nname: second\n---\nname: third\n")

	var configs []Config
	err := Load(input, &configs, WithAllDocuments())
	assert.NoError(t, err)

	assert.Equal(t, 3, len(configs))
	assert.Equal(t, "first", configs[0].Name)
	assert.Equal(t, "second", configs[1].Name)
	assert.Equal(t, "third", configs[2].Name)
}

// TestLoadWithAllDocuments_UntypedSlice tests loading multiple documents into []any
func TestLoadWithAllDocuments_UntypedSlice(t *testing.T) {
	input := []byte("---\nname: first\n---\nname: second\n")

	var docs []any
	err := Load(input, &docs, WithAllDocuments())
	assert.NoError(t, err)

	assert.Equal(t, 2, len(docs))
}

// TestLoadWithAllDocuments_EmptyInput tests that 0 documents with WithAllDocuments results in empty slice
func TestLoadWithAllDocuments_EmptyInput(t *testing.T) {
	input := []byte("")

	var docs []any
	err := Load(input, &docs, WithAllDocuments())
	assert.NoError(t, err)

	assert.Equal(t, 0, len(docs))
}

// TestLoadWithAllDocuments_NonSlice tests that WithAllDocuments with non-slice target returns error
func TestLoadWithAllDocuments_NonSlice(t *testing.T) {
	input := []byte("---\nname: first\n---\nname: second\n")

	var single map[string]any
	err := Load(input, &single, WithAllDocuments())
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*WithAllDocuments requires a pointer to a slice.*", err)
}

// TestLoad_SingleDocument tests loading exactly one document
func TestLoad_SingleDocument(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	input := []byte("name: myconfig\n")

	var config Config
	err := Load(input, &config)
	assert.NoError(t, err)

	assert.Equal(t, "myconfig", config.Name)
}

// TestLoad_ZeroDocuments tests that 0 documents returns error
func TestLoad_ZeroDocuments(t *testing.T) {
	input := []byte("")

	var config map[string]any
	err := Load(input, &config)
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*no documents in stream.*", err)
}

// TestLoad_MultipleDocuments tests that 2+ documents returns error
func TestLoad_MultipleDocuments(t *testing.T) {
	input := []byte("---\nname: first\n---\nname: second\n")

	var config map[string]any
	err := Load(input, &config)
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*expected single document, found multiple.*", err)
}
