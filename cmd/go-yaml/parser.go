// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Parser wrapper for CLI YAML token/event processing. Provides a simplified
// interface for the command-line tool to access internal parser functionality.

package main

import (
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Parser provides access to the internal YAML Parser for CLI use
type Parser struct {
	parser        libyaml.Parser
	done          bool
	pendingTokens []*Token
	commentsHead  int
}

// NewParser creates a new YAML parser reading from the given reader for CLI use
func NewParser(reader io.Reader) (*Parser, error) {
	p := &Parser{
		parser: libyaml.NewParser(),
	}
	p.parser.SetInputReader(reader)
	return p, nil
}

// Next returns the next token in the YAML stream
func (p *Parser) Next() (*Token, error) {
	// Return pending tokens first
	if len(p.pendingTokens) > 0 {
		token := p.pendingTokens[0]
		p.pendingTokens = p.pendingTokens[1:]
		return token, nil
	}

	if p.done {
		return nil, nil
	}

	var yamlToken libyaml.Token
	if err := p.parser.Scan(&yamlToken); err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("parser error: %w", err)
		}
		p.done = true
		return nil, nil
	}

	token := &Token{
		StartLine:   int(yamlToken.StartMark.Line) + 1,
		StartColumn: int(yamlToken.StartMark.Column),
		EndLine:     int(yamlToken.EndMark.Line) + 1,
		EndColumn:   int(yamlToken.EndMark.Column),
	}

	switch yamlToken.Type {
	case libyaml.STREAM_START_TOKEN:
		token.Type = "STREAM-START"
	case libyaml.STREAM_END_TOKEN:
		token.Type = "STREAM-END"
		p.done = true
	case libyaml.DOCUMENT_START_TOKEN:
		token.Type = "DOCUMENT-START"
	case libyaml.DOCUMENT_END_TOKEN:
		token.Type = "DOCUMENT-END"
	case libyaml.BLOCK_SEQUENCE_START_TOKEN:
		token.Type = "BLOCK-SEQUENCE-START"
	case libyaml.BLOCK_MAPPING_START_TOKEN:
		token.Type = "BLOCK-MAPPING-START"
	case libyaml.BLOCK_END_TOKEN:
		token.Type = "BLOCK-END"
	case libyaml.FLOW_SEQUENCE_START_TOKEN:
		token.Type = "FLOW-SEQUENCE-START"
	case libyaml.FLOW_SEQUENCE_END_TOKEN:
		token.Type = "FLOW-SEQUENCE-END"
	case libyaml.FLOW_MAPPING_START_TOKEN:
		token.Type = "FLOW-MAPPING-START"
	case libyaml.FLOW_MAPPING_END_TOKEN:
		token.Type = "FLOW-MAPPING-END"
	case libyaml.BLOCK_ENTRY_TOKEN:
		token.Type = "BLOCK-ENTRY"
	case libyaml.FLOW_ENTRY_TOKEN:
		token.Type = "FLOW-ENTRY"
	case libyaml.KEY_TOKEN:
		token.Type = "KEY"
	case libyaml.VALUE_TOKEN:
		token.Type = "VALUE"
	case libyaml.ALIAS_TOKEN:
		token.Type = "ALIAS"
		token.Value = string(yamlToken.Value)
	case libyaml.ANCHOR_TOKEN:
		token.Type = "ANCHOR"
		token.Value = string(yamlToken.Value)
	case libyaml.TAG_TOKEN:
		token.Type = "TAG"
		token.Value = string(yamlToken.Value)
	case libyaml.SCALAR_TOKEN:
		token.Type = "SCALAR"
		token.Value = string(yamlToken.Value)
		token.Style = yamlToken.Style.String()
	case libyaml.VERSION_DIRECTIVE_TOKEN:
		token.Type = "VERSION-DIRECTIVE"
	case libyaml.TAG_DIRECTIVE_TOKEN:
		token.Type = "TAG-DIRECTIVE"
	default:
		token.Type = "UNKNOWN"
	}

	// Process comments that should be emitted before this token
	p.processComments(&yamlToken, token)

	// Return first pending token if comments were queued, otherwise return the main token
	if len(p.pendingTokens) > 0 {
		// Add the main token to the end of pending tokens
		p.pendingTokens = append(p.pendingTokens, token)
		// Return the first pending token
		result := p.pendingTokens[0]
		p.pendingTokens = p.pendingTokens[1:]
		return result, nil
	}

	return token, nil
}

// processComments extracts comments from the parser and creates COMMENT tokens
func (p *Parser) processComments(yamlToken *libyaml.Token, mainToken *Token) {
	comments := p.parser.GetPendingComments()

	for p.commentsHead < len(comments) {
		comment := &comments[p.commentsHead]

		// Check if this comment should be emitted before the current token
		// Comments are associated with tokens based on their TokenMark
		if yamlToken.StartMark.Index < comment.TokenMark.Index {
			// This comment is for a future token, stop processing
			break
		}

		// Create comment tokens for head, line, and foot comments
		p.appendCommentTokenIfNotEmpty(comment.Head, "head", comment)
		p.appendCommentTokenIfNotEmpty(comment.Line, "line", comment)
		p.appendCommentTokenIfNotEmpty(comment.Foot, "foot", comment)

		p.commentsHead++
	}
}

// appendCommentTokenIfNotEmpty creates and appends a comment token if the value is not empty.
func (p *Parser) appendCommentTokenIfNotEmpty(value []byte, commentType string, comment *libyaml.Comment) {
	if len(value) > 0 {
		commentToken := &Token{
			Type:        "COMMENT",
			Value:       string(value),
			CommentType: commentType,
			StartLine:   int(comment.StartMark.Line) + 1,
			StartColumn: int(comment.StartMark.Column) + 1,
			EndLine:     int(comment.EndMark.Line) + 1,
			EndColumn:   int(comment.EndMark.Column) + 1,
		}
		p.pendingTokens = append(p.pendingTokens, commentToken)
	}
}

// Close releases the parser resources
func (p *Parser) Close() {
	p.parser.Delete()
}
