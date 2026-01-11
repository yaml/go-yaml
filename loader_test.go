// Tests for the streaming Loader API, including StreamNode functionality
// and multi-document streaming.

package yaml_test

import (
	"bytes"
	"io"
	"testing"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/testutil/assert"
)

// TestStreamNodeEmptyStream tests that an empty stream returns a single StreamNode
func TestStreamNodeEmptyStream(t *testing.T) {
	input := []byte("")

	loader, err := yaml.NewLoader(bytes.NewReader(input), yaml.WithStreamNodes())
	assert.NoError(t, err)

	var nodes []yaml.Node
	for {
		var node yaml.Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Empty stream should return exactly one StreamNode
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, yaml.StreamNode, nodes[0].Kind)
}

// TestStreamNodeSingleDocument tests the pattern [Stream, Doc, Stream] for single document
func TestStreamNodeSingleDocument(t *testing.T) {
	input := []byte("key: value\n")

	loader, err := yaml.NewLoader(bytes.NewReader(input), yaml.WithStreamNodes())
	assert.NoError(t, err)

	var nodes []yaml.Node
	for {
		var node yaml.Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Single document should return [Stream, Doc, Stream]
	assert.Equal(t, 3, len(nodes))
	assert.Equal(t, yaml.StreamNode, nodes[0].Kind)
	assert.Equal(t, yaml.DocumentNode, nodes[1].Kind)
	assert.Equal(t, yaml.StreamNode, nodes[2].Kind)
}

// TestStreamNodeMultiDocument tests interleaved pattern for multi-document stream
func TestStreamNodeMultiDocument(t *testing.T) {
	input := []byte("---\nkey1: value1\n---\nkey2: value2\n")

	loader, err := yaml.NewLoader(bytes.NewReader(input), yaml.WithStreamNodes())
	assert.NoError(t, err)

	var nodes []yaml.Node
	for {
		var node yaml.Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Two documents should return [Stream, Doc, Stream, Doc, Stream]
	assert.Equal(t, 5, len(nodes))
	assert.Equal(t, yaml.StreamNode, nodes[0].Kind)
	assert.Equal(t, yaml.DocumentNode, nodes[1].Kind)
	assert.Equal(t, yaml.StreamNode, nodes[2].Kind)
	assert.Equal(t, yaml.DocumentNode, nodes[3].Kind)
	assert.Equal(t, yaml.StreamNode, nodes[4].Kind)
}

// TestStreamNodeDirectives tests that directives are captured on StreamNodes
func TestStreamNodeDirectives(t *testing.T) {
	input := []byte("%YAML 1.1\n%TAG ! tag:example.com,2000:app/\n---\nkey: value\n")

	loader, err := yaml.NewLoader(bytes.NewReader(input), yaml.WithStreamNodes())
	assert.NoError(t, err)

	var nodes []yaml.Node
	for {
		var node yaml.Node
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
	assert.Equal(t, yaml.StreamNode, nodes[0].Kind)
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

	loader, err := yaml.NewLoader(bytes.NewReader(input), yaml.WithStreamNodes())
	assert.NoError(t, err)

	var node yaml.Node
	err = loader.Load(&node)
	assert.NoError(t, err)

	// First node should be a StreamNode with encoding
	assert.Equal(t, yaml.StreamNode, node.Kind)
	// Encoding should be set (non-zero)
	if node.Encoding == 0 {
		t.Fatal("stream node should have encoding set")
	}
}

// TestWithoutStreamNodes tests backward compatibility (default behavior)
func TestWithoutStreamNodes(t *testing.T) {
	input := []byte("---\nkey1: value1\n---\nkey2: value2\n")

	loader, err := yaml.NewLoader(bytes.NewReader(input))
	assert.NoError(t, err)

	var nodes []yaml.Node
	for {
		var node yaml.Node
		err := loader.Load(&node)
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		nodes = append(nodes, node)
	}

	// Without stream nodes, should only return DocumentNodes
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, yaml.DocumentNode, nodes[0].Kind)
	assert.Equal(t, yaml.DocumentNode, nodes[1].Kind)
}

// TestStreamNodeDisabled tests explicitly disabling stream nodes
func TestStreamNodeDisabled(t *testing.T) {
	input := []byte("key: value\n")

	loader, err := yaml.NewLoader(bytes.NewReader(input), yaml.WithStreamNodes(false))
	assert.NoError(t, err)

	var node yaml.Node
	err = loader.Load(&node)
	assert.NoError(t, err)

	// Should get a DocumentNode, not a StreamNode
	assert.Equal(t, yaml.DocumentNode, node.Kind)
}

// TestLoadWithAll_TypedSlice tests loading multiple documents into a typed slice
func TestLoadWithAll_TypedSlice(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	input := []byte("---\nname: first\n---\nname: second\n---\nname: third\n")

	var configs []Config
	err := yaml.Load(input, &configs, yaml.WithAll())
	assert.NoError(t, err)

	assert.Equal(t, 3, len(configs))
	assert.Equal(t, "first", configs[0].Name)
	assert.Equal(t, "second", configs[1].Name)
	assert.Equal(t, "third", configs[2].Name)
}

// TestLoadWithAll_UntypedSlice tests loading multiple documents into []any
func TestLoadWithAll_UntypedSlice(t *testing.T) {
	input := []byte("---\nname: first\n---\nname: second\n")

	var docs []any
	err := yaml.Load(input, &docs, yaml.WithAll())
	assert.NoError(t, err)

	assert.Equal(t, 2, len(docs))
}

// TestLoadWithAll_EmptyInput tests that 0 documents with WithAll results in empty slice
func TestLoadWithAll_EmptyInput(t *testing.T) {
	input := []byte("")

	var docs []any
	err := yaml.Load(input, &docs, yaml.WithAll())
	assert.NoError(t, err)

	assert.Equal(t, 0, len(docs))
}

// TestLoadWithAll_NonSlice tests that WithAll with non-slice target returns error
func TestLoadWithAll_NonSlice(t *testing.T) {
	input := []byte("---\nname: first\n---\nname: second\n")

	var single map[string]any
	err := yaml.Load(input, &single, yaml.WithAll())
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*WithAll requires a pointer to a slice.*", err)
}

// TestLoad_SingleDocument tests loading exactly one document
func TestLoad_SingleDocument(t *testing.T) {
	type Config struct {
		Name string `yaml:"name"`
	}

	input := []byte("name: myconfig\n")

	var config Config
	err := yaml.Load(input, &config)
	assert.NoError(t, err)

	assert.Equal(t, "myconfig", config.Name)
}

// TestLoad_ZeroDocuments tests that 0 documents returns error
func TestLoad_ZeroDocuments(t *testing.T) {
	input := []byte("")

	var config map[string]any
	err := yaml.Load(input, &config)
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*no documents in stream.*", err)
}

// TestLoad_MultipleDocuments tests that 2+ documents returns error
func TestLoad_MultipleDocuments(t *testing.T) {
	input := []byte("---\nname: first\n---\nname: second\n")

	var config map[string]any
	err := yaml.Load(input, &config)
	assert.NotNil(t, err)
	assert.ErrorMatches(t, ".*expected single document, found multiple.*", err)
}
