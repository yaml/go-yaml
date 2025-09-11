// Package main provides YAML event formatting utilities for the go-yaml tool.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"go.yaml.in/yaml/v4"
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
	Event  string `yaml:"Event"`
	Value  string `yaml:"Value,omitempty"`
	Style  string `yaml:"Style,omitempty"`
	Tag    string `yaml:"Tag,omitempty"`
	Anchor string `yaml:"Anchor,omitempty"`
	Head   string `yaml:"Head,omitempty"`
	Line   string `yaml:"Line,omitempty"`
	Foot   string `yaml:"Foot,omitempty"`
	Pos    string `yaml:"Pos,omitempty"`
}

// ProcessEvents reads YAML from stdin and outputs event information
func ProcessEvents(profuse, compact, unmarshal bool) error {
	if unmarshal {
		return processEventsUnmarshal(profuse, compact)
	}
	return processEventsDecode(profuse, compact)
}

// processEventsDecode uses Decoder.Decode for YAML processing
func processEventsDecode(profuse, compact bool) error {
	decoder := yaml.NewDecoder(os.Stdin)
	firstDoc := true

	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to decode YAML: %w", err)
		}

		// Add document separator for all documents except the first
		if !firstDoc {
			fmt.Println("---")
		}
		firstDoc = false

		events := processNodeToEvents(&node, profuse)

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
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Event"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Event})

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
				if info.Tag != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Tag"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Tag})
				}
				if info.Anchor != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Anchor"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Anchor})
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
					return fmt.Errorf("failed to marshal compact event info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		} else {
			// For non-compact mode, output each event as a separate mapping
			for _, event := range events {
				info := formatEventInfo(event, profuse)

				var buf bytes.Buffer
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				if err := enc.Encode([]*EventInfo{info}); err != nil {
					enc.Close()
					return fmt.Errorf("failed to marshal event info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		}
	}

	return nil
}

// processEventsUnmarshal uses yaml.Unmarshal for YAML processing
func processEventsUnmarshal(profuse, compact bool) error {
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

		// Convert to yaml.Node for event processing
		var node yaml.Node
		if err := yaml.Unmarshal(doc, &node); err != nil {
			return fmt.Errorf("failed to unmarshal YAML to node: %w", err)
		}

		events := processNodeToEvents(&node, profuse)

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
					&yaml.Node{Kind: yaml.ScalarNode, Value: "Event"},
					&yaml.Node{Kind: yaml.ScalarNode, Value: info.Event})

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
				if info.Tag != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Tag"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Tag})
				}
				if info.Anchor != "" {
					compactNode.Content = append(compactNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: "Anchor"},
						&yaml.Node{Kind: yaml.ScalarNode, Value: info.Anchor})
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
					return fmt.Errorf("failed to marshal compact event info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		} else {
			// For non-compact mode, output each event as a separate mapping
			for _, event := range events {
				info := formatEventInfo(event, profuse)

				var buf bytes.Buffer
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				if err := enc.Encode([]*EventInfo{info}); err != nil {
					enc.Close()
					return fmt.Errorf("failed to marshal event info: %w", err)
				}
				enc.Close()
				fmt.Print(buf.String())
			}
		}
	}

	return nil
}

// processNodeToEvents converts a node to a slice of events for compact output
func processNodeToEvents(node *yaml.Node, profuse bool) []*Event {
	var events []*Event

	// Add document start event
	events = append(events, &Event{
		Type:        "DOCUMENT-START",
		StartLine:   node.Line,
		StartColumn: node.Column,
		EndLine:     node.Line,
		EndColumn:   node.Column,
	})

	// Process the node content
	events = append(events, processNodeToEventsRecursive(node, profuse)...)

	// Add document end event
	events = append(events, &Event{
		Type:        "DOCUMENT-END",
		StartLine:   node.Line,
		StartColumn: node.Column,
		EndLine:     node.Line,
		EndColumn:   node.Column,
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
		events = append(events, &Event{
			Type:        "MAPPING-START",
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
			Style:       formatStyle(node.Style),
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
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
		})
	case yaml.SequenceNode:
		events = append(events, &Event{
			Type:        "SEQUENCE-START",
			StartLine:   node.Line,
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
			Style:       formatStyle(node.Style),
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
			StartColumn: node.Column,
			EndLine:     node.Line,
			EndColumn:   node.Column,
		})
	case yaml.ScalarNode:
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
			StartColumn: node.Column,
			EndLine:     endLine,
			EndColumn:   endColumn,
			Style:       formatStyle(node.Style),
			HeadComment: node.HeadComment,
			LineComment: node.LineComment,
			FootComment: node.FootComment,
		})
	case yaml.AliasNode:
		// Generate ALIAS event for alias nodes
		events = append(events, &Event{
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
			info.Pos = fmt.Sprintf("%d;%d", event.StartLine, event.StartColumn)
		} else {
			info.Pos = fmt.Sprintf("%d;%d-%d;%d", event.StartLine, event.StartColumn, event.EndLine, event.EndColumn)
		}
	}

	return info
}
