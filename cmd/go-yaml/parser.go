package main

import (
	"fmt"
	"io"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Parser provides access to the internal YAML Parser for CLI use
type Parser struct {
	parser libyaml.Parser
	done   bool
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
	if p.done {
		return nil, nil
	}

	var yamlToken libyaml.Token
	if !p.parser.Scan(&yamlToken) {
		if p.parser.ErrorType != libyaml.NO_ERROR {
			return nil,
				fmt.Errorf("parser error: %v", p.parser.Problem)
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

	// Call unfoldComments to process comment information from the parser
	// This moves comments from the comments queue to the parser's comment fields
	p.parser.UnfoldComments(&yamlToken)

	// Access comment information from the parser
	// The parser stores comments in head_comment, line_comment, and foot_comment fields
	if len(p.parser.HeadComment) > 0 {
		token.HeadComment = string(p.parser.HeadComment)
		// Clear the comment after using it to avoid duplication
		p.parser.HeadComment = nil
	}
	if len(p.parser.LineComment) > 0 {
		token.LineComment = string(p.parser.LineComment)
		// Clear the comment after using it to avoid duplication
		p.parser.LineComment = nil
	}
	if len(p.parser.FootComment) > 0 {
		token.FootComment = string(p.parser.FootComment)
		// Clear the comment after using it to avoid duplication
		p.parser.FootComment = nil
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

	return token, nil
}

// Close releases the parser resources
func (p *Parser) Close() {
	p.parser.Delete()
}
