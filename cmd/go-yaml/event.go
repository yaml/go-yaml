// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

// Package main provides YAML event formatting utilities for the go-yaml tool.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

// EventType represents the type of a YAML event
type EventType string

const (
	EventDocumentStart EventType = "DOCUMENT-START"
	EventDocumentEnd   EventType = "DOCUMENT-END"
	EventScalar        EventType = "SCALAR"
	EventSequenceStart EventType = "SEQUENCE-START"
	EventSequenceEnd   EventType = "SEQUENCE-END"
	EventMappingStart  EventType = "MAPPING-START"
	EventMappingEnd    EventType = "MAPPING-END"
)

// Event represents a YAML event
type Event struct {
	Type        EventType
	Value       string
	Anchor      string
	Tag         string
	Style       string
	Implicit    bool
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	HeadComment string
	LineComment string
	FootComment string
}

// EventInfo represents the information about a YAML event for YAML encoding
type EventInfo struct {
	Event    string `yaml:"event"`
	Value    string `yaml:"value,omitempty"`
	Style    string `yaml:"style,omitempty"`
	Tag      string `yaml:"tag,omitempty"`
	Anchor   string `yaml:"anchor,omitempty"`
	Implicit *bool  `yaml:"implicit,omitempty"`
	Explicit *bool  `yaml:"explicit,omitempty"`
	Head     string `yaml:"head,omitempty"`
	Line     string `yaml:"line,omitempty"`
	Foot     string `yaml:"foot,omitempty"`
	Pos      string `yaml:"pos,omitempty"`
}

// ProcessEvents reads YAML from stdin and outputs event information
func ProcessEvents(profuse, compact, unmarshal bool) error {
	if unmarshal {
		return processEventsUnmarshal(profuse, compact)
	}
	return processEventsDecode(profuse, compact)
}

// processEventsDecode uses yaml.Decoder for YAML processing with implicit field augmentation
func processEventsDecode(profuse, compact bool) error {
	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Get implicit flags from libyaml parser
	implicitFlags, err := getDocumentImplicitFlags(input)
	if err != nil {
		return err
	}

	// Use yaml.Loader to get events with comments
	loader, err := yaml.NewLoader(bytes.NewReader(input))
	if err != nil {
		return fmt.Errorf("failed to create loader: %w", err)
	}
	docIndex := 0
	var allEvents []*Event

	for {
		var node yaml.Node
		err := loader.Load(&node)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to load YAML: %w", err)
		}

		// Get events from node (includes comments)
		events := processNodeToEvents(&node, profuse)

		// Augment document start/end events with implicit flags
		if docIndex < len(implicitFlags) {
			for _, event := range events {
				switch event.Type {
				case "DOCUMENT-START":
					event.Implicit = implicitFlags[docIndex].StartImplicit
				case "DOCUMENT-END":
					event.Implicit = implicitFlags[docIndex].EndImplicit
				}
			}
		}

		allEvents = append(allEvents, events...)
		docIndex++
	}

	events := allEvents

	if compact {
		// For compact mode, output each event as a flow style mapping in a sequence
		for _, event := range events {
			info := formatEventInfo(event, profuse)

			// Create a YAML node with flow style for the mapping
			compactNode := &yaml.Node{
				Kind:  yaml.MappingNode,
				Style: yaml.FlowStyle,
			}

			// Add the Event field
			compactNode.Content = append(compactNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "event"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: info.Event})

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
			if info.Tag != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "tag"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Tag})
			}
			if info.Anchor != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "anchor"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Anchor})
			}
			if info.Implicit != nil {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "implicit"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%t", *info.Implicit)})
			}
			if info.Explicit != nil {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "explicit"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%t", *info.Explicit)})
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
			enc, err := yaml.NewDumper(&buf)
			if err != nil {
				return fmt.Errorf("failed to create dumper: %w", err)
			}
			if err := enc.Dump([]*yaml.Node{compactNode}); err != nil {
				enc.Close()
				return fmt.Errorf("failed to dump compact event info: %w", err)
			}
			enc.Close()
			fmt.Print(buf.String())
		}
	} else {
		// For non-compact mode, output each event as a separate mapping
		for _, event := range events {
			info := formatEventInfo(event, profuse)

			var buf bytes.Buffer
			enc, err := yaml.NewDumper(&buf)
			if err != nil {
				return fmt.Errorf("failed to create dumper: %w", err)
			}
			if err := enc.Dump([]*EventInfo{info}); err != nil {
				enc.Close()
				return fmt.Errorf("failed to dump event info: %w", err)
			}
			enc.Close()
			fmt.Print(buf.String())
		}
	}

	return nil
}

// processEventsUnmarshal uses yaml.Unmarshal for YAML processing with implicit field augmentation
func processEventsUnmarshal(profuse, compact bool) error {
	// Read all input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Get implicit flags from libyaml parser
	implicitFlags, err := getDocumentImplicitFlags(input)
	if err != nil {
		return err
	}

	// Split input into documents
	documents := bytes.Split(input, []byte("---"))
	docIndex := 0
	var allEvents []*Event

	for _, doc := range documents {
		// Skip empty documents
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		// Convert to yaml.Node for event processing
		var node yaml.Node
		if err := yaml.Load(doc, &node); err != nil {
			return fmt.Errorf("failed to load YAML to node: %w", err)
		}

		// Get events from node (includes comments)
		events := processNodeToEvents(&node, profuse)

		// Augment document start/end events with implicit flags
		if docIndex < len(implicitFlags) {
			for _, event := range events {
				switch event.Type {
				case "DOCUMENT-START":
					event.Implicit = implicitFlags[docIndex].StartImplicit
				case "DOCUMENT-END":
					event.Implicit = implicitFlags[docIndex].EndImplicit
				}
			}
		}

		allEvents = append(allEvents, events...)
		docIndex++
	}

	events := allEvents

	if compact {
		// For compact mode, output each event as a flow style mapping in a sequence
		for _, event := range events {
			info := formatEventInfo(event, profuse)

			// Create a YAML node with flow style for the mapping
			compactNode := &yaml.Node{
				Kind:  yaml.MappingNode,
				Style: yaml.FlowStyle,
			}

			// Add the Event field
			compactNode.Content = append(compactNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "event"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: info.Event})

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
			if info.Tag != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "tag"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Tag})
			}
			if info.Anchor != "" {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "anchor"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Anchor})
			}
			if info.Implicit != nil {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "implicit"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%t", *info.Implicit)})
			}
			if info.Explicit != nil {
				compactNode.Content = append(compactNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "explicit"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%t", *info.Explicit)})
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
			enc, err := yaml.NewDumper(&buf)
			if err != nil {
				return fmt.Errorf("failed to create dumper: %w", err)
			}
			if err := enc.Dump([]*yaml.Node{compactNode}); err != nil {
				enc.Close()
				return fmt.Errorf("failed to dump compact event info: %w", err)
			}
			enc.Close()
			fmt.Print(buf.String())
		}
	} else {
		// For non-compact mode, output each event as a separate mapping
		for _, event := range events {
			info := formatEventInfo(event, profuse)

			var buf bytes.Buffer
			enc, err := yaml.NewDumper(&buf)
			if err != nil {
				return fmt.Errorf("failed to create dumper: %w", err)
			}
			if err := enc.Dump([]*EventInfo{info}); err != nil {
				enc.Close()
				return fmt.Errorf("failed to dump event info: %w", err)
			}
			enc.Close()
			fmt.Print(buf.String())
		}
	}

	return nil
}

// adjustColumn converts yaml.Node 1-based column to 0-based (libyaml format)
func adjustColumn(col int) int {
	if col > 0 {
		return col - 1
	}
	return 0
}

// processNodeToEvents converts a node to a slice of events for compact output
func processNodeToEvents(node *yaml.Node, profuse bool) []*Event {
	var events []*Event

	// Add document start event
	// yaml.Node uses 1-based columns, but we need 0-based (libyaml format)
	startCol := adjustColumn(node.Column)
	events = append(events, &Event{
		Type:        "DOCUMENT-START",
		StartLine:   node.Line,
		StartColumn: startCol,
		EndLine:     node.Line,
		EndColumn:   startCol,
	})

	// Process the node content
	events = append(events, processNodeToEventsRecursive(node, profuse)...)

	// Add document end event
	events = append(events, &Event{
		Type:        "DOCUMENT-END",
		StartLine:   node.Line,
		StartColumn: startCol,
		EndLine:     node.Line,
		EndColumn:   startCol,
	})

	return events
}

// processNodeToEventsRecursive recursively converts a node to events
func processNodeToEventsRecursive(node *yaml.Node, profuse bool) []*Event {
	var events []*Event

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			events = append(events, processNodeToEventsRecursive(child, profuse)...)
		}
	case yaml.MappingNode:
		startCol := adjustColumn(node.Column)
		events = append(events, &Event{
			Type:        "MAPPING-START",
			StartLine:   node.Line,
			StartColumn: startCol,
			EndLine:     node.Line,
			EndColumn:   startCol,
			Style:       formatStyle(node.Style, profuse),
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) {
				// Key
				keyEvents := processNodeToEventsRecursive(node.Content[i], profuse)
				events = append(events, keyEvents...)
				// Value
				valueEvents := processNodeToEventsRecursive(node.Content[i+1], profuse)
				events = append(events, valueEvents...)
			}
		}
		events = append(events, &Event{
			Type:        "MAPPING-END",
			StartLine:   node.Line,
			StartColumn: startCol,
			EndLine:     node.Line,
			EndColumn:   startCol,
		})
	case yaml.SequenceNode:
		startCol := adjustColumn(node.Column)
		events = append(events, &Event{
			Type:        "SEQUENCE-START",
			StartLine:   node.Line,
			StartColumn: startCol,
			EndLine:     node.Line,
			EndColumn:   startCol,
			Style:       formatStyle(node.Style, profuse),
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
		for _, child := range node.Content {
			childEvents := processNodeToEventsRecursive(child, profuse)
			events = append(events, childEvents...)
		}
		events = append(events, &Event{
			Type:        "SEQUENCE-END",
			StartLine:   node.Line,
			StartColumn: startCol,
			EndLine:     node.Line,
			EndColumn:   startCol,
		})
	case yaml.ScalarNode:
		// Calculate end position for scalars based on value length
		// yaml.Node uses 1-based columns, adjust to 0-based
		startCol := adjustColumn(node.Column)
		endLine := node.Line
		endColumn := startCol
		if node.Value != "" {
			// For single-line values, add the length to the column
			if !strings.Contains(node.Value, "\n") {
				endColumn = startCol + len(node.Value)
			}
		}

		// Filter out default YAML tags
		tag := node.Tag
		// Check if the tag was explicit in the input
		tagWasExplicit := node.Style&yaml.TaggedStyle != 0

		// Show !!str only if it was explicit in the input
		if tag == "!!str" {
			if !tagWasExplicit {
				tag = ""
			}
		}
		// Show all other tags (no filtering)

		events = append(events, &Event{
			Type:        "SCALAR",
			Value:       node.Value,
			Anchor:      node.Anchor,
			Tag:         tag,
			StartLine:   node.Line,
			StartColumn: startCol,
			EndLine:     endLine,
			EndColumn:   endColumn,
			Style:       formatStyle(node.Style, profuse),
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
	case yaml.AliasNode:
		// Generate ALIAS event for alias nodes
		startCol := adjustColumn(node.Column)
		events = append(events, &Event{
			Type:        "ALIAS",
			Value:       node.Value,
			StartLine:   node.Line,
			StartColumn: startCol,
			EndLine:     node.Line,
			EndColumn:   startCol,
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
	}

	return events
}

// formatEventInfo converts an Event to an EventInfo struct for YAML encoding
func formatEventInfo(event *Event, profuse bool) *EventInfo {
	info := &EventInfo{
		Event: string(event.Type),
	}

	if event.Value != "" {
		info.Value = event.Value
	}
	if event.Style != "" {
		info.Style = event.Style
	}
	if event.Tag != "" {
		info.Tag = event.Tag
	}
	if event.Anchor != "" {
		info.Anchor = event.Anchor
	}
	if event.HeadComment != "" {
		info.Head = event.HeadComment
	}
	if event.LineComment != "" {
		info.Line = event.LineComment
	}
	if event.FootComment != "" {
		info.Foot = event.FootComment
	}
	if profuse {
		if event.StartLine == event.EndLine && event.StartColumn == event.EndColumn {
			// Single position
			info.Pos = fmt.Sprintf("%d/%d", event.StartLine, event.StartColumn)
		} else if event.StartLine == event.EndLine {
			// Range on same line
			info.Pos = fmt.Sprintf("%d/%d-%d", event.StartLine, event.StartColumn, event.EndColumn)
		} else {
			// Range across different lines
			info.Pos = fmt.Sprintf("%d/%d-%d/%d", event.StartLine, event.StartColumn, event.EndLine, event.EndColumn)
		}
	}

	// Handle implicit/explicit for document start/end events
	if event.Type == "DOCUMENT-START" || event.Type == "DOCUMENT-END" {
		if profuse {
			// For -E mode: show implicit: true when implicit
			if event.Implicit {
				trueVal := true
				info.Implicit = &trueVal
			}
		} else {
			// For -e mode: show explicit: true when NOT implicit
			if !event.Implicit {
				trueVal := true
				info.Explicit = &trueVal
			}
		}
	}

	return info
}

// DocumentImplicitFlags holds implicit flags for document start and end events
type DocumentImplicitFlags struct {
	StartImplicit bool
	EndImplicit   bool
}

// getDocumentImplicitFlags extracts implicit flags for all documents
func getDocumentImplicitFlags(input []byte) ([]*DocumentImplicitFlags, error) {
	p := libyaml.NewParser()
	if len(input) == 0 {
		input = []byte{'\n'}
	}
	p.SetInputString(input)

	var flags []*DocumentImplicitFlags
	var currentDoc *DocumentImplicitFlags
	var ev libyaml.Event

	for {
		if err := p.Parse(&ev); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		switch ev.Type {
		case libyaml.DOCUMENT_START_EVENT:
			currentDoc = &DocumentImplicitFlags{
				StartImplicit: ev.Implicit,
			}
			flags = append(flags, currentDoc)
		case libyaml.DOCUMENT_END_EVENT:
			if currentDoc != nil {
				currentDoc.EndImplicit = ev.Implicit
			}
		case libyaml.STREAM_END_EVENT:
			ev.Delete()
			return flags, nil
		}

		ev.Delete()
	}
}

// getEventsFromParser parses YAML input and extracts events with implicit field information
func getEventsFromParser(input []byte, profuse bool) ([]*Event, error) {
	p := libyaml.NewParser()
	if len(input) == 0 {
		input = []byte{'\n'}
	}
	p.SetInputString(input)

	var events []*Event
	var ev libyaml.Event

	for {
		if err := p.Parse(&ev); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		event := convertLibyamlEvent(&ev, profuse)
		if event != nil {
			events = append(events, event)
		}

		if ev.Type == libyaml.STREAM_END_EVENT {
			ev.Delete()
			break
		}
		ev.Delete()
	}

	return events, nil
}

// convertLibyamlEvent converts a libyaml event to our Event struct
func convertLibyamlEvent(ev *libyaml.Event, profuse bool) *Event {
	// Skip stream events
	if ev.Type == libyaml.STREAM_START_EVENT || ev.Type == libyaml.STREAM_END_EVENT {
		return nil
	}

	event := &Event{
		StartLine:   ev.StartMark.Line + 1, // libyaml uses 0-based lines
		StartColumn: ev.StartMark.Column,
		EndLine:     ev.EndMark.Line + 1,
		EndColumn:   ev.EndMark.Column,
	}

	switch ev.Type {
	case libyaml.DOCUMENT_START_EVENT:
		event.Type = "DOCUMENT-START"
		event.Implicit = ev.Implicit
	case libyaml.DOCUMENT_END_EVENT:
		event.Type = "DOCUMENT-END"
		event.Implicit = ev.Implicit
	case libyaml.MAPPING_START_EVENT:
		event.Type = "MAPPING-START"
		event.Anchor = string(ev.Anchor)
		event.Tag = string(ev.Tag)
		// Style handling for mapping
		if ev.MappingStyle() == libyaml.FLOW_MAPPING_STYLE {
			event.Style = "Flow"
		}
	case libyaml.MAPPING_END_EVENT:
		event.Type = "MAPPING-END"
	case libyaml.SEQUENCE_START_EVENT:
		event.Type = "SEQUENCE-START"
		event.Anchor = string(ev.Anchor)
		event.Tag = string(ev.Tag)
		// Style handling for sequence
		if ev.SequenceStyle() == libyaml.FLOW_SEQUENCE_STYLE {
			event.Style = "Flow"
		}
	case libyaml.SEQUENCE_END_EVENT:
		event.Type = "SEQUENCE-END"
	case libyaml.SCALAR_EVENT:
		event.Type = "SCALAR"
		event.Value = string(ev.Value)
		event.Anchor = string(ev.Anchor)
		event.Tag = string(ev.Tag)
		event.Implicit = ev.Implicit
		// Style handling for scalar
		switch ev.ScalarStyle() {
		case libyaml.PLAIN_SCALAR_STYLE:
			if profuse {
				event.Style = "Plain"
			}
		case libyaml.DOUBLE_QUOTED_SCALAR_STYLE:
			event.Style = "Double"
		case libyaml.SINGLE_QUOTED_SCALAR_STYLE:
			event.Style = "Single"
		case libyaml.LITERAL_SCALAR_STYLE:
			event.Style = "Literal"
		case libyaml.FOLDED_SCALAR_STYLE:
			event.Style = "Folded"
		}
	case libyaml.ALIAS_EVENT:
		event.Type = "ALIAS"
		event.Anchor = string(ev.Anchor)
	}

	return event
}
