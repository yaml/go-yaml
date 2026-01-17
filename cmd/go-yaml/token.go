// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

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
	CommentType string // For COMMENT tokens: "head", "line", or "foot"
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
	Token string `yaml:"token"`
	Value string `yaml:"value,omitempty"`
	Style string `yaml:"style,omitempty"`
	Head  string `yaml:"head,omitempty"`
	Line  string `yaml:"line,omitempty"`
	Foot  string `yaml:"foot,omitempty"`
	Pos   string `yaml:"pos,omitempty"`
}

// ProcessTokens reads YAML from reader and outputs token information using the internal scanner
func ProcessTokens(reader io.Reader, profuse, compact, unmarshal bool) error {
	if unmarshal {
		return processTokensUnmarshal(reader, profuse, compact)
	}
	return processTokensWithParser(reader, profuse, compact)
}

// processTokensDecode uses Loader.Load for YAML processing
func processTokensDecode(profuse, compact bool) error {
	loader, err := yaml.NewLoader(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to create loader: %w", err)
	}
	firstDoc := true

	for {
		var node yaml.Node
		err := loader.Load(&node)
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
					&yaml.Node{Kind: yaml.ScalarNode, Value: "token"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Token})

				// Add other fields if they exist
				if info.Value != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "value"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Value})
				}
				if info.Style != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "style"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Style})
				}
				if info.Head != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "head"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Head})
				}
				if info.Line != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "line"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Line})
				}
				if info.Foot != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "foot"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Foot})
				}
				if info.Pos != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "pos"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Pos})
				}

				var buf bytes.Buffer
				dumper, err := yaml.NewDumper(&buf)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump([]*yaml.Node{compactNode}); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump compact token info: %w", err)
				}
				dumper.Close()
				fmt.Print(buf.String())
			}
		} else {
			// For non-compact mode, output each token as a separate mapping
			for _, token := range tokens {
				info := formatTokenInfo(token, profuse)

				var buf bytes.Buffer
				dumper, err := yaml.NewDumper(&buf)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump([]*TokenInfo{info}); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump token info: %w", err)
				}
				dumper.Close()
				fmt.Print(buf.String())
			}
		}
	}

	return nil
}

// processTokensWithParser uses the internal parser for token processing
func processTokensWithParser(reader io.Reader, profuse, compact bool) error {
	p, err := NewParser(reader)
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

		info := formatTokenInfo(token, profuse)

		if compact {
			// For compact mode, output each token as a flow style mapping in a sequence
			compactNode := &yaml.Node{
				Kind:  yaml.MappingNode,
				Style: yaml.FlowStyle,
			}

			// Add the Token field
			compactNode.Content = append(compactNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "token"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: info.Token})

			// Add other fields if they exist
			if info.Value != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "value"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Value})
			}
			if info.Style != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "style"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Style})
			}
			if info.Head != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "head"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Head})
			}
			if info.Line != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "line"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Line})
			}
			if info.Foot != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "foot"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Foot})
			}
			if info.Pos != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "pos"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Pos})
			}

			var buf bytes.Buffer
			dumper, err := yaml.NewDumper(&buf)
			if err != nil {
				return fmt.Errorf("failed to create dumper: %w", err)
			}
			if err := dumper.Dump([]*yaml.Node{compactNode}); err != nil {
				dumper.Close()
				return fmt.Errorf("failed to dump compact token info: %w", err)
			}
			dumper.Close()
			fmt.Print(buf.String())
		} else {
			// For non-compact mode, output each token as a separate mapping
			var buf bytes.Buffer
			dumper, err := yaml.NewDumper(&buf)
			if err != nil {
				return fmt.Errorf("failed to create dumper: %w", err)
			}
			if err := dumper.Dump([]*TokenInfo{info}); err != nil {
				dumper.Close()
				return fmt.Errorf("failed to dump token info: %w", err)
			}
			dumper.Close()
			fmt.Print(buf.String())
		}
	}

	return nil
}

// processTokensUnmarshal uses [yaml.Unmarshal] for YAML processing
func processTokensUnmarshal(reader io.Reader, profuse, compact bool) error {
	// Read all input from reader
	input, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
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

		// For unmarshal mode, use `any` first to avoid preserving comments
		var data any
		if err := yaml.Load(doc, &data); err != nil {
			return fmt.Errorf("failed to load YAML: %w", err)
		}

		// Convert to yaml.Node for token processing
		var node yaml.Node
		if err := yaml.Load(doc, &node); err != nil {
			return fmt.Errorf("failed to load YAML to node: %w", err)
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
					&yaml.Node{Kind: yaml.ScalarNode, Value: "token"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Token})

				// Add other fields if they exist
				if info.Value != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "value"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Value})
				}
				if info.Style != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "style"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Style})
				}
				if info.Head != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "head"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Head})
				}
				if info.Line != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "line"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Line})
				}
				if info.Foot != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "foot"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Foot})
				}
				if info.Pos != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "pos"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Pos})
				}

				var buf bytes.Buffer
				dumper, err := yaml.NewDumper(&buf)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump([]*yaml.Node{compactNode}); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump compact token info: %w", err)
				}
				dumper.Close()
				fmt.Print(buf.String())
			}
		} else {
			// For non-compact mode, output each token as a separate mapping
			for _, token := range tokens {
				info := formatTokenInfo(token, profuse)

				var buf bytes.Buffer
				dumper, err := yaml.NewDumper(&buf)
				if err != nil {
					return fmt.Errorf("failed to create dumper: %w", err)
				}
				if err := dumper.Dump([]*TokenInfo{info}); err != nil {
					dumper.Close()
					return fmt.Errorf("failed to dump token info: %w", err)
				}
				dumper.Close()
				fmt.Print(buf.String())
			}
		}
	}

	return nil
}

// formatTokenInfo converts a [Token] to a [TokenInfo] struct for YAML encoding
func formatTokenInfo(token *Token, profuse bool) *TokenInfo {
	info := &TokenInfo{
		Token: token.Type,
	}

	// For COMMENT tokens, use the CommentType to determine which field to populate
	if token.Type == "COMMENT" && token.CommentType != "" && token.Value != "" {
		switch token.CommentType {
		case "head":
			info.Head = token.Value
		case "line":
			info.Line = token.Value
		case "foot":
			info.Foot = token.Value
		}
	} else {
		// For non-COMMENT tokens
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
	}

	if profuse {
		if token.StartLine == token.EndLine && token.StartColumn == token.EndColumn {
			// Single position
			info.Pos = fmt.Sprintf("%d:%d", token.StartLine, token.StartColumn)
		} else if token.StartLine == token.EndLine {
			// Range on same line
			info.Pos = fmt.Sprintf("%d:%d-%d", token.StartLine, token.StartColumn, token.EndColumn)
		} else {
			// Range across different lines
			info.Pos = fmt.Sprintf("%d:%d-%d:%d", token.StartLine, token.StartColumn, token.EndLine, token.EndColumn)
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
			Style:       formatStyle(node.Style, false),
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
