// Package main provides YAML token formatting utilities for the go-yaml tool.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Token represents a YAML token with comment information
type Token struct {
	Type        string
	Value       string
	Style       string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	HeadComment string
	LineComment string
	FootComment string
}

// TokenInfo represents the information about a YAML token for YAML encoding
type TokenInfo struct {
	Token string `yaml:"Token"`
	Value string `yaml:"Value,omitempty"`
	Style string `yaml:"Style,omitempty"`
	Head  string `yaml:"Head,omitempty"`
	Line  string `yaml:"Line,omitempty"`
	Foot  string `yaml:"Foot,omitempty"`
	Pos   string `yaml:"Pos,omitempty"`
}

// ProcessTokens reads YAML from stdin and outputs token information using the internal scanner
func ProcessTokens(profuse, compact, unmarshal bool) error {
	if unmarshal {
		return processTokensUnmarshal(profuse, compact)
	}
	return processTokensWithParser(profuse, compact)
}

// processTokensDecode uses Decoder.Decode for YAML processing
func processTokensDecode(profuse, compact bool) error {
	decoder := yaml.NewDecoder(os.Stdin)
	firstDoc := true

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode YAML: %w", err)
		}

		// Add document separator for all documents except the first
		if !firstDoc {
			fmt.Println("---")
		}
		firstDoc = false

		tokens := processNodeToTokens(&node, profuse)

		if compact {
			// For compact mode, output each token as a flow style mapping in a sequence
			for _, token := range tokens {
				info := formatTokenInfo(token, profuse)

				// Create a YAML node with flow style for the mapping
				compactNode := &yaml.Node{
					Kind:  yaml.MappingNode,
					Style: yaml.FlowStyle,
				}

				// Add the Token field
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Token"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Token})

				// Add other fields if they exist
				if info.Value != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Value"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Value})
				}
				if info.Style != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Style"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Style})
				}
				if info.Head != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Head"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Head})
				}
				if info.Line != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Line"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Line})
				}
				if info.Foot != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Foot"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Foot})
				}
				if info.Pos != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Pos"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Pos})
				}

				var buf bytes.Buffer
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				if err := enc.Encode([]*yaml.Node{compactNode}); err != nil {
					enc.Close()
					return fmt.Errorf("failed to marshal compact token info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		} else {
			// For non-compact mode, output each token as a separate mapping
			for _, token := range tokens {
				info := formatTokenInfo(token, profuse)

				var buf bytes.Buffer
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				if err := enc.Encode([]*TokenInfo{info}); err != nil {
					enc.Close()
					return fmt.Errorf("failed to marshal token info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		}
	}

	return nil
}

// processTokensWithParser uses the internal parser for token processing
func processTokensWithParser(profuse, compact bool) error {
	p, err := yaml.NewParser(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	defer p.Close()

	for {
		token, err := p.Next()
		if err != nil {
			return fmt.Errorf("failed to get next token: %w", err)
		}
		if token == nil {
			break
		}

		// Convert parser token to our token format
		ourToken := &Token{
			Type:        token.Type,
			Value:       token.Value,
			Style:       token.Style,
			StartLine:   token.StartLine,
			StartColumn: token.StartCol,
			EndLine:     token.EndLine,
			EndColumn:   token.EndCol,
		}

		info := formatTokenInfo(ourToken, profuse)

		if compact {
			// For compact mode, output each token as a flow style mapping in a sequence
			compactNode := &yaml.Node{
				Kind:  yaml.MappingNode,
				Style: yaml.FlowStyle,
			}

			// Add the Token field
			compactNode.Content = append(compactNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "Token"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: info.Token})

			// Add other fields if they exist
			if info.Value != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Value"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Value})
			}
			if info.Style != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Style"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Style})
			}
			if info.Head != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Head"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Head})
			}
			if info.Line != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Line"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Line})
			}
			if info.Foot != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Foot"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Foot})
			}
			if info.Pos != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Pos"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Pos})
			}

			var buf bytes.Buffer
			enc := yaml.NewEncoder(&buf)
			enc.SetIndent(2)
			if err := enc.Encode([]*yaml.Node{compactNode}); err != nil {
				enc.Close()
				return fmt.Errorf("failed to marshal compact token info: %w", err)
			}
			enc.Close()
			fmt.Print(buf.String())
		} else {
			// For non-compact mode, output each token as a separate mapping
			var buf bytes.Buffer
			enc := yaml.NewEncoder(&buf)
			enc.SetIndent(2)
			if err := enc.Encode([]*TokenInfo{info}); err != nil {
				enc.Close()
				return fmt.Errorf("failed to marshal token info: %w", err)
			}
			enc.Close()
			fmt.Print(buf.String())
		}
	}

	return nil
}

// processTokensUnmarshal uses yaml.Unmarshal for YAML processing
func processTokensUnmarshal(profuse, compact bool) error {
	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Split input into documents
	documents := bytes.Split(input, []byte("---"))
	firstDoc := true

	for _, doc := range documents {
		// Skip empty documents
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		// Add document separator for all documents except the first
		if !firstDoc {
			fmt.Println("---")
		}
		firstDoc = false

		// For unmarshal mode, use interface{} first to avoid preserving comments
		var data interface{}
		if err := yaml.Unmarshal(doc, &data); err != nil {
			return fmt.Errorf("failed to unmarshal YAML: %w", err)
		}

		// Convert to yaml.Node for token processing
		var node yaml.Node
		if err := yaml.Unmarshal(doc, &node); err != nil {
			return fmt.Errorf("failed to unmarshal YAML to node: %w", err)
		}

		tokens := processNodeToTokens(&node, profuse)

		if compact {
			// For compact mode, output each token as a flow style mapping in a sequence
			for _, token := range tokens {
				info := formatTokenInfo(token, profuse)

				// Create a YAML node with flow style for the mapping
				compactNode := &yaml.Node{
					Kind:  yaml.MappingNode,
					Style: yaml.FlowStyle,
				}

				// Add the Token field
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Token"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Token})

				// Add other fields if they exist
				if info.Value != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Value"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Value})
				}
				if info.Style != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Style"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Style})
				}
				if info.Head != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Head"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Head})
				}
				if info.Line != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Line"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Line})
				}
				if info.Foot != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Foot"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Foot})
				}
				if info.Pos != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Pos"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Pos})
				}

				var buf bytes.Buffer
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				if err := enc.Encode([]*yaml.Node{compactNode}); err != nil {
					enc.Close()
					return fmt.Errorf("failed to marshal compact token info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		} else {
			// For non-compact mode, output each token as a separate mapping
			for _, token := range tokens {
				info := formatTokenInfo(token, profuse)

				var buf bytes.Buffer
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				if err := enc.Encode([]*TokenInfo{info}); err != nil {
					enc.Close()
					return fmt.Errorf("failed to marshal token info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		}
	}

	return nil
}

// formatTokenInfo converts a Token to a TokenInfo struct for YAML encoding
func formatTokenInfo(token *Token, profuse bool) *TokenInfo {
	info := &TokenInfo{
		Token: token.Type,
	}

	if token.Value != "" {
		info.Value = token.Value
	}
	if token.Style != "" && token.Style != "Plain" {
		info.Style = token.Style
	}
	if token.HeadComment != "" {
		info.Head = token.HeadComment
	}
	if token.LineComment != "" {
		info.Line = token.LineComment
	}
	if token.FootComment != "" {
		info.Foot = token.FootComment
	}
	if profuse {
		if token.StartLine == token.EndLine && token.StartColumn == token.EndColumn {
			info.Pos = fmt.Sprintf("%d;%d", token.StartLine, token.StartColumn)
		} else {
			info.Pos = fmt.Sprintf("%d;%d-%d;%d", token.StartLine, token.StartColumn, token.EndLine, token.EndColumn)
		}
	}

	return info
}

// processNodeToTokens converts a node to a slice of tokens
func processNodeToTokens(node *yaml.Node, profuse bool) []*Token {
	var tokens []*Token

	// Add stream start token
	tokens = append(tokens, &Token{
		Type: "STREAM-START",
	})

	// Add document start token
	tokens = append(tokens, &Token{
		Type: "DOCUMENT-START",
	})

	// Process the node content
	tokens = append(tokens, processNodeToTokensRecursive(node, profuse)...)

	// Add document end token
	tokens = append(tokens, &Token{
		Type: "DOCUMENT-END",
	})

	// Add stream end token
	tokens = append(tokens, &Token{
		Type: "STREAM-END",
	})

	return tokens
}

// processNodeToTokensRecursive recursively converts a node to tokens
func processNodeToTokensRecursive(node *yaml.Node, profuse bool) []*Token {
	var tokens []*Token

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			tokens = append(tokens, processNodeToTokensRecursive(child, profuse)...)
		}
	case yaml.MappingNode:
		tokens = append(tokens, &Token{
			Type:        "BLOCK-MAPPING-START",
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) {
				// Key
				tokens = append(tokens, &Token{
					Type:        "KEY",
					StartLine:   node.Content[i].Line,
					StartColumn: node.Content[i].Column,
					EndLine:     node.Content[i].Line,
					EndColumn:   node.Content[i].Column,
				})
				keyTokens := processNodeToTokensRecursive(node.Content[i], profuse)
				tokens = append(tokens, keyTokens...)
				// Value
				tokens = append(tokens, &Token{
					Type:        "VALUE",
					StartLine:   node.Content[i+1].Line,
					StartColumn: node.Content[i+1].Column,
					EndLine:     node.Content[i+1].Line,
					EndColumn:   node.Content[i+1].Column,
				})
				valueTokens := processNodeToTokensRecursive(node.Content[i+1], profuse)
				tokens = append(tokens, valueTokens...)
			}
		}
		tokens = append(tokens, &Token{
			Type:        "BLOCK-END",
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
		})
	case yaml.SequenceNode:
		tokens = append(tokens, &Token{
			Type:        "BLOCK-SEQUENCE-START",
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
		for _, child := range node.Content {
			tokens = append(tokens, &Token{
				Type:        "BLOCK-ENTRY",
				StartLine:   child.Line,
				StartColumn: child.Column,
				EndLine:     child.Line,
				EndColumn:   child.Column,
			})
			childTokens := processNodeToTokensRecursive(child, profuse)
			tokens = append(tokens, childTokens...)
		}
		tokens = append(tokens, &Token{
			Type:        "BLOCK-END",
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
		})
	case yaml.ScalarNode:
		// Check for anchor before the scalar
		if node.Anchor != "" {
			tokens = append(tokens, &Token{
				Type:        "ANCHOR",
				Value:       node.Anchor,
				StartLine:   node.Line,
				StartColumn: node.Column,
				EndLine:     node.Line,
				EndColumn:   node.Column,
			})
		}

		// Check for tag before the scalar
		// Check if the tag was explicit in the input
		tagWasExplicit := node.Style&yaml.TaggedStyle != 0

		// Show tag if it exists and either:
		// - it's not !!str, or
		// - it's !!str and was explicit in the input
		if node.Tag != "" && (node.Tag != "!!str" || tagWasExplicit) {
			tokens = append(tokens, &Token{
				Type:        "TAG",
				Value:       node.Tag,
				StartLine:   node.Line,
				StartColumn: node.Column,
				EndLine:     node.Line,
				EndColumn:   node.Column,
			})
		}

		// Calculate end position for scalars based on value length
		endLine := node.Line
		endColumn := node.Column
		if node.Value != "" {
			// For single-line values, add the length to the column
			if !strings.Contains(node.Value, "\n") {
				endColumn += len(node.Value)
			} else {
				// For multi-line values, we'd need more complex logic
				// For now, just use the start position
				endColumn = node.Column
			}
		}
		tokens = append(tokens, &Token{
			Type:        "SCALAR",
			Value:       node.Value,
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     endLine,
			EndColumn:   endColumn,
			Style:       formatStyle(node.Style),
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
	case yaml.AliasNode:
		// Generate ALIAS token for alias nodes
		tokens = append(tokens, &Token{
			Type:        "ALIAS",
			Value:       node.Value,
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
	}

	return tokens
}
