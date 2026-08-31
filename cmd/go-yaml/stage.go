// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
	"go.yaml.in/yaml/v4/internal/libyaml"
)

type stageKind string

const (
	stageYAML  stageKind = "yaml"
	stageToken stageKind = "token"
	stageEvent stageKind = "event"
	stageNode  stageKind = "node"
)

type structuredInput struct {
	stage  stageKind
	tokens []*TokenInfo
	events []*EventInfo
	nodes  []*yaml.Node
}

func parseStage(value string) (stageKind, error) {
	switch strings.ToLower(value) {
	case "y", "yaml":
		return stageYAML, nil
	case "t", "token", "tokens":
		return stageToken, nil
	case "e", "event", "events":
		return stageEvent, nil
	case "n", "node", "nodes":
		return stageNode, nil
	default:
		return "", fmt.Errorf("unknown input stage %q (use token, event, node, or yaml)", value)
	}
}

func detectStructuredInput(data []byte, forced string) (*structuredInput, error) {
	if forced != "" {
		stage, err := parseStage(forced)
		if err != nil {
			return nil, err
		}
		return decodeStructuredInput(data, stage)
	}

	if tokens, err := decodeTokenContract(data); err == nil {
		return &structuredInput{stage: stageToken, tokens: tokens}, nil
	}
	if events, err := decodeEventContract(data); err == nil {
		return &structuredInput{stage: stageEvent, events: events}, nil
	}
	if nodes, err := decodeNodeContract(data); err == nil {
		return &structuredInput{stage: stageNode, nodes: nodes}, nil
	}
	return &structuredInput{stage: stageYAML}, nil
}

func decodeStructuredInput(data []byte, stage stageKind) (*structuredInput, error) {
	switch stage {
	case stageYAML:
		return &structuredInput{stage: stageYAML}, nil
	case stageToken:
		tokens, err := decodeTokenContract(data)
		return &structuredInput{stage: stageToken, tokens: tokens}, err
	case stageEvent:
		events, err := decodeEventContract(data)
		return &structuredInput{stage: stageEvent, events: events}, err
	case stageNode:
		nodes, err := decodeNodeContract(data)
		return &structuredInput{stage: stageNode, nodes: nodes}, err
	default:
		return nil, fmt.Errorf("unsupported input stage %q", stage)
	}
}

func decodeTokenContract(data []byte) ([]*TokenInfo, error) {
	var tokens []*TokenInfo
	if err := yaml.Load(data, &tokens, yaml.WithKnownFields()); err != nil {
		return nil, fmt.Errorf("invalid token input: %w", err)
	}
	if len(tokens) < 2 {
		return nil, errors.New("invalid token input: expected a token stream")
	}
	var real []*TokenInfo
	for i, token := range tokens {
		if token == nil || !validTokenType(token.Token) {
			return nil, fmt.Errorf("invalid token input at item %d", i+1)
		}
		if token.Token != "COMMENT" {
			real = append(real, token)
		}
	}
	if len(real) < 2 || real[0].Token != "STREAM-START" ||
		real[len(real)-1].Token != "STREAM-END" {
		return nil, errors.New("invalid token input: stream boundaries are required")
	}
	return tokens, nil
}

func validTokenType(value string) bool {
	_, ok := tokenTypes[value]
	return ok || value == "COMMENT"
}

var tokenTypes = map[string]libyaml.TokenType{
	"STREAM-START":         libyaml.STREAM_START_TOKEN,
	"STREAM-END":           libyaml.STREAM_END_TOKEN,
	"VERSION-DIRECTIVE":    libyaml.VERSION_DIRECTIVE_TOKEN,
	"TAG-DIRECTIVE":        libyaml.TAG_DIRECTIVE_TOKEN,
	"DOCUMENT-START":       libyaml.DOCUMENT_START_TOKEN,
	"DOCUMENT-END":         libyaml.DOCUMENT_END_TOKEN,
	"BLOCK-SEQUENCE-START": libyaml.BLOCK_SEQUENCE_START_TOKEN,
	"BLOCK-MAPPING-START":  libyaml.BLOCK_MAPPING_START_TOKEN,
	"BLOCK-END":            libyaml.BLOCK_END_TOKEN,
	"FLOW-SEQUENCE-START":  libyaml.FLOW_SEQUENCE_START_TOKEN,
	"FLOW-SEQUENCE-END":    libyaml.FLOW_SEQUENCE_END_TOKEN,
	"FLOW-MAPPING-START":   libyaml.FLOW_MAPPING_START_TOKEN,
	"FLOW-MAPPING-END":     libyaml.FLOW_MAPPING_END_TOKEN,
	"BLOCK-ENTRY":          libyaml.BLOCK_ENTRY_TOKEN,
	"FLOW-ENTRY":           libyaml.FLOW_ENTRY_TOKEN,
	"KEY":                  libyaml.KEY_TOKEN,
	"VALUE":                libyaml.VALUE_TOKEN,
	"ALIAS":                libyaml.ALIAS_TOKEN,
	"ANCHOR":               libyaml.ANCHOR_TOKEN,
	"TAG":                  libyaml.TAG_TOKEN,
	"SCALAR":               libyaml.SCALAR_TOKEN,
}

func decodeEventContract(data []byte) ([]*EventInfo, error) {
	var events []*EventInfo
	if err := yaml.Load(data, &events, yaml.WithKnownFields()); err != nil {
		return nil, fmt.Errorf("invalid event input: %w", err)
	}
	if len(events) < 2 {
		return nil, errors.New("invalid event input: expected an event stream")
	}
	for i, event := range events {
		if event == nil || !validEventType(event.Event) {
			return nil, fmt.Errorf("invalid event input at item %d", i+1)
		}
	}
	if events[0].Event != "STREAM-START" || events[len(events)-1].Event != "STREAM-END" {
		return nil, errors.New("invalid event input: stream boundaries are required")
	}
	return events, nil
}

func validEventType(value string) bool {
	switch EventType(value) {
	case EventStreamStart, EventStreamEnd, EventDocumentStart, EventDocumentEnd,
		EventScalar, EventSequenceStart, EventSequenceEnd, EventMappingStart,
		EventMappingEnd, "ALIAS", EventTailComment:
		return true
	default:
		return false
	}
}

func decodeNodeContract(data []byte) ([]*yaml.Node, error) {
	if nodes, err := decodeDetailedNodes(data); err == nil {
		return nodes, nil
	}
	return decodeCompactNodes(data)
}

func decodeDetailedNodes(data []byte) ([]*yaml.Node, error) {
	var one NodeInfo
	if err := yaml.Load(data, &one, yaml.WithKnownFields()); err == nil && one.Kind != "" {
		node, err := nodeInfoToNode(&one)
		if err != nil {
			return nil, err
		}
		nodes := []*yaml.Node{node}
		if err := relinkAliases(nodes); err != nil {
			return nil, err
		}
		return nodes, nil
	}

	var many []*NodeInfo
	if err := yaml.Load(data, &many, yaml.WithKnownFields()); err != nil {
		return nil, fmt.Errorf("invalid node input: %w", err)
	}
	if len(many) == 0 {
		return nil, errors.New("invalid node input: empty node list")
	}
	nodes := make([]*yaml.Node, len(many))
	for i, info := range many {
		if info == nil || info.Kind == "" {
			return nil, fmt.Errorf("invalid node input at item %d", i+1)
		}
		node, err := nodeInfoToNode(info)
		if err != nil {
			return nil, fmt.Errorf("invalid node input at item %d: %w", i+1, err)
		}
		nodes[i] = node
	}
	if err := relinkAliases(nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func nodeInfoToNode(info *NodeInfo) (*yaml.Node, error) {
	kind, err := parseNodeKind(info.Kind)
	if err != nil {
		return nil, err
	}
	style, err := parseNodeStyle(info.Style)
	if err != nil {
		return nil, err
	}
	node := &yaml.Node{
		Kind:        kind,
		Style:       style,
		Anchor:      info.Anchor,
		Tag:         info.Tag,
		Value:       info.Value,
		HeadComment: info.Head,
		LineComment: info.Line,
		FootComment: info.Foot,
	}
	for _, child := range info.Content {
		if child == nil {
			return nil, errors.New("null node content is not allowed")
		}
		content, err := nodeInfoToNode(child)
		if err != nil {
			return nil, err
		}
		node.Content = append(node.Content, content)
	}
	if kind == yaml.StreamNode {
		node.Stream = &yaml.Stream{Encoding: parseNodeEncoding(info.Encoding)}
		if info.Version != "" {
			major, minor, err := parseVersion(info.Version)
			if err != nil {
				return nil, err
			}
			node.Stream.Version = &yaml.VersionDirective{Major: major, Minor: minor}
		}
		for _, directive := range info.TagDirectives {
			node.Stream.TagDirectives = append(node.Stream.TagDirectives,
				yaml.TagDirective{Handle: directive.Handle, Prefix: directive.Prefix})
		}
	}
	return node, nil
}

func parseNodeKind(kind string) (yaml.Kind, error) {
	switch strings.ToLower(kind) {
	case "document":
		return yaml.DocumentNode, nil
	case "sequence":
		return yaml.SequenceNode, nil
	case "mapping":
		return yaml.MappingNode, nil
	case "scalar":
		return yaml.ScalarNode, nil
	case "alias":
		return yaml.AliasNode, nil
	case "stream":
		return yaml.StreamNode, nil
	default:
		return 0, fmt.Errorf("unknown node kind %q", kind)
	}
}

func parseNodeStyle(style string) (yaml.Style, error) {
	switch strings.ToLower(style) {
	case "", "plain":
		return 0, nil
	case "double":
		return yaml.DoubleQuotedStyle, nil
	case "single":
		return yaml.SingleQuotedStyle, nil
	case "literal":
		return yaml.LiteralStyle, nil
	case "folded":
		return yaml.FoldedStyle, nil
	case "flow":
		return yaml.FlowStyle, nil
	default:
		return 0, fmt.Errorf("unknown node style %q", style)
	}
}

func decodeCompactNodes(data []byte) ([]*yaml.Node, error) {
	var contract yaml.Node
	if err := yaml.Load(data, &contract); err != nil {
		return nil, fmt.Errorf("invalid node input: %w", err)
	}
	if len(contract.Content) != 1 {
		return nil, errors.New("invalid node input: expected one contract document")
	}
	root := contract.Content[0]
	if root.Kind == yaml.SequenceNode {
		var nodes []*yaml.Node
		for i, item := range root.Content {
			node, err := compactNodeToNode(item)
			if err != nil {
				return nil, fmt.Errorf("invalid compact node at item %d: %w", i+1, err)
			}
			nodes = append(nodes, wrapDocument(node))
		}
		if err := relinkAliases(nodes); err != nil {
			return nil, err
		}
		return nodes, nil
	}
	node, err := compactNodeToNode(root)
	if err != nil {
		return nil, fmt.Errorf("invalid compact node input: %w", err)
	}
	nodes := []*yaml.Node{wrapDocument(node)}
	if err := relinkAliases(nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func compactNodeToNode(contract *yaml.Node) (*yaml.Node, error) {
	if contract.Kind != yaml.MappingNode || len(contract.Content)%2 != 0 {
		return nil, errors.New("node must be a mapping")
	}
	node := &yaml.Node{}
	var shape string
	for i := 0; i < len(contract.Content); i += 2 {
		key, value := contract.Content[i].Value, contract.Content[i+1]
		switch key {
		case "anchor":
			node.Anchor = value.Value
		case "tag":
			node.Tag = value.Value
		case "head":
			node.HeadComment = value.Value
		case "line":
			node.LineComment = value.Value
		case "foot":
			node.FootComment = value.Value
		case "mapping", "sequence", "plain", "double", "single",
			"literal", "folded", "alias", "stream":
			if shape != "" {
				return nil, errors.New("node has multiple shape fields")
			}
			shape = key
			if err := fillCompactNode(node, key, value); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown node field %q", key)
		}
	}
	if shape == "" {
		return nil, errors.New("node has no shape field")
	}
	return node, nil
}

func fillCompactNode(node *yaml.Node, shape string, value *yaml.Node) error {
	switch shape {
	case "mapping", "sequence":
		if value.Kind != yaml.SequenceNode {
			return fmt.Errorf("%s content must be a sequence", shape)
		}
		if shape == "mapping" {
			node.Kind, node.Tag = yaml.MappingNode, defaultTag(node.Tag, "!!map")
			if len(value.Content)%2 != 0 {
				return errors.New("mapping content must contain key/value node pairs")
			}
		} else {
			node.Kind, node.Tag = yaml.SequenceNode, defaultTag(node.Tag, "!!seq")
		}
		for _, item := range value.Content {
			child, err := compactNodeToNode(item)
			if err != nil {
				return err
			}
			node.Content = append(node.Content, child)
		}
	case "plain", "double", "single", "literal", "folded":
		node.Kind, node.Value = yaml.ScalarNode, value.Value
		style, _ := parseNodeStyle(shape)
		node.Style = style
	case "alias":
		node.Kind, node.Value = yaml.AliasNode, value.Value
	case "stream":
		node.Kind = yaml.StreamNode
		node.Stream = &yaml.Stream{}
	default:
		return fmt.Errorf("unsupported compact node shape %q", shape)
	}
	return nil
}

func defaultTag(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func wrapDocument(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode || node.Kind == yaml.StreamNode {
		return node
	}
	return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{node}}
}

func relinkAliases(nodes []*yaml.Node) error {
	for _, root := range nodes {
		anchors := map[string]*yaml.Node{}
		var walk func(*yaml.Node) error
		walk = func(node *yaml.Node) error {
			if node.Anchor != "" {
				anchors[node.Anchor] = node
			}
			if node.Kind == yaml.AliasNode {
				node.Alias = anchors[node.Value]
				if node.Alias == nil {
					return fmt.Errorf("unknown alias anchor %q", node.Value)
				}
			}
			for _, child := range node.Content {
				if err := walk(child); err != nil {
					return err
				}
			}
			return nil
		}
		if err := walk(root); err != nil {
			return err
		}
	}
	return nil
}

func processStructuredInput(input *structuredInput, target stageKind, profuse,
	compact, preserve bool, opts []yaml.Option,
) error {
	if targetRank(target) < targetRank(input.stage) {
		return fmt.Errorf("cannot convert %s input backward to %s output", input.stage, target)
	}

	switch input.stage {
	case stageToken:
		tokens, comments, err := tokenContractToLibyaml(input.tokens)
		if err != nil {
			return err
		}
		if target == stageToken {
			return writeTokenContract(input.tokens, profuse, compact)
		}
		events, err := parseTokenStream(tokens, comments)
		if err != nil {
			return err
		}
		return processEventStream(events, target, profuse, compact, preserve, opts)
	case stageEvent:
		events, err := eventContractToLibyaml(input.events)
		if err != nil {
			return err
		}
		if target == stageEvent {
			return writeEventContract(input.events, profuse, compact)
		}
		return processEventStream(events, target, profuse, compact, preserve, opts)
	case stageNode:
		if target == stageNode {
			return writeNodeContract(input.nodes, profuse)
		}
		return writeNodesAsYAML(input.nodes, preserve, opts)
	default:
		return errors.New("internal error: structured processor received YAML input")
	}
}

func targetRank(stage stageKind) int {
	switch stage {
	case stageToken:
		return 1
	case stageEvent:
		return 2
	case stageNode:
		return 3
	case stageYAML:
		return 4
	default:
		return -1
	}
}

func tokenContractToLibyaml(infos []*TokenInfo) ([]libyaml.Token, []libyaml.Comment, error) {
	marks := make([]libyaml.Mark, len(infos))
	for i, info := range infos {
		start, _, err := parsePosition(info.Pos, i)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid token position at item %d: %w", i+1, err)
		}
		marks[i] = start
	}

	var tokens []libyaml.Token
	var comments []libyaml.Comment
	for i, info := range infos {
		start, end, _ := parsePosition(info.Pos, i)
		if info.Token == "COMMENT" {
			tokenMark := end
			for j := i + 1; j < len(infos); j++ {
				if infos[j].Token != "COMMENT" {
					tokenMark = marks[j]
					break
				}
			}
			comments = append(comments, libyaml.Comment{
				ScanMark: start, TokenMark: tokenMark,
				StartMark: start, EndMark: end,
				Head: []byte(info.Head), Line: []byte(info.Line), Foot: []byte(info.Foot),
			})
			continue
		}
		token := libyaml.Token{
			Type: tokenTypes[info.Token], StartMark: start, EndMark: end,
			Value: []byte(info.Value), Style: parseScalarStyle(info.Style),
		}
		switch info.Token {
		case "STREAM-START":
			token.SetEncoding(parseEncoding(info.Encoding))
		case "VERSION-DIRECTIVE":
			major, minor, err := parseVersion(info.Version)
			if err != nil {
				return nil, nil, err
			}
			token.SetVersion(major, minor)
		case "TAG-DIRECTIVE":
			token.Value = []byte(info.Handle)
			token.SetPrefix(info.Prefix)
		case "TAG":
			token.Value = []byte(info.Handle)
			token.SetSuffix(info.Suffix)
		}
		tokens = append(tokens, token)
	}
	return tokens, comments, nil
}

func parseTokenStream(tokens []libyaml.Token, comments []libyaml.Comment) ([]libyaml.Event, error) {
	parser := libyaml.NewParser()
	defer parser.Delete()
	parser.SetInputTokens(tokens, comments)
	var events []libyaml.Event
	for {
		var event libyaml.Event
		if err := parser.Parse(&event); err != nil {
			return nil, fmt.Errorf("failed to parse token input: %w", err)
		}
		events = append(events, event)
		if event.Type == libyaml.STREAM_END_EVENT {
			break
		}
	}
	return events, nil
}

func eventContractToLibyaml(infos []*EventInfo) ([]libyaml.Event, error) {
	events := make([]libyaml.Event, 0, len(infos))
	for i, info := range infos {
		implicit := defaultImplicit(info)
		quotedImplicit := implicit
		if info.QuotedImplicit != nil {
			quotedImplicit = *info.QuotedImplicit
		}
		var event libyaml.Event
		switch EventType(info.Event) {
		case EventStreamStart:
			event = libyaml.NewStreamStartEvent(parseEncoding(info.Encoding))
		case EventStreamEnd:
			event = libyaml.NewStreamEndEvent()
		case EventDocumentStart:
			var version *libyaml.VersionDirective
			if info.Version != "" {
				major, minor, err := parseVersion(info.Version)
				if err != nil {
					return nil, err
				}
				version = libyaml.NewVersionDirective(major, minor)
			}
			var directives []libyaml.TagDirective
			for _, directive := range info.TagDirectives {
				directives = append(directives,
					libyaml.NewTagDirective(directive.Handle, directive.Prefix))
			}
			event = libyaml.NewDocumentStartEvent(version, directives, implicit)
		case EventDocumentEnd:
			event = libyaml.NewDocumentEndEvent(implicit)
		case EventScalar:
			event = libyaml.NewScalarEvent([]byte(info.Anchor), []byte(info.Tag),
				[]byte(info.Value), implicit, quotedImplicit, parseScalarStyle(info.Style))
		case EventSequenceStart:
			event = libyaml.NewSequenceStartEvent([]byte(info.Anchor), []byte(info.Tag),
				implicit, parseSequenceStyle(info.Style))
		case EventSequenceEnd:
			event = libyaml.NewSequenceEndEvent()
		case EventMappingStart:
			event = libyaml.NewMappingStartEvent([]byte(info.Anchor), []byte(info.Tag),
				implicit, parseMappingStyle(info.Style))
		case EventMappingEnd:
			event = libyaml.NewMappingEndEvent()
		case "ALIAS":
			event = libyaml.NewAliasEvent([]byte(info.Anchor))
		case EventTailComment:
			event = libyaml.Event{Type: libyaml.TAIL_COMMENT_EVENT}
		default:
			return nil, fmt.Errorf("unknown event %q at item %d", info.Event, i+1)
		}
		start, end, err := parsePosition(info.Pos, i)
		if err != nil {
			return nil, fmt.Errorf("invalid event position at item %d: %w", i+1, err)
		}
		event.StartMark, event.EndMark = start, end
		event.HeadComment = []byte(info.Head)
		event.LineComment = []byte(info.Line)
		event.FootComment = []byte(info.Foot)
		event.TailComment = []byte(info.Tail)
		events = append(events, event)
	}
	return events, nil
}

func defaultImplicit(info *EventInfo) bool {
	if info.Implicit != nil {
		return *info.Implicit
	}
	return info.Tag == "" && info.Version == "" && len(info.TagDirectives) == 0
}

func processEventStream(events []libyaml.Event, target stageKind, profuse,
	compact, preserve bool, opts []yaml.Option,
) error {
	switch target {
	case stageEvent:
		infos := make([]*EventInfo, 0, len(events))
		for i := range events {
			infos = append(infos, formatEventInfo(convertLibyamlEvent(&events[i], profuse), profuse))
		}
		return writeEventContract(infos, profuse, compact)
	case stageNode:
		nodes, err := composeEventStream(events, opts)
		if err != nil {
			return err
		}
		return writeNodeContract(nodes, profuse)
	case stageYAML:
		if preserve {
			return emitEventStream(events, opts)
		}
		nodes, err := composeEventStream(events, opts)
		if err != nil {
			return err
		}
		return writeNodesAsYAML(nodes, false, opts)
	default:
		return fmt.Errorf("unsupported event conversion to %s", target)
	}
}

func composeEventStream(events []libyaml.Event, opts []yaml.Option) ([]*yaml.Node, error) {
	options, err := libyaml.ApplyOptions(opts...)
	if err != nil {
		return nil, err
	}
	return libyaml.ComposeEvents(events, options)
}

func emitEventStream(events []libyaml.Event, opts []yaml.Option) error {
	options, err := libyaml.ApplyOptions(opts...)
	if err != nil {
		return err
	}
	serializer := libyaml.NewSerializer(os.Stdout, options)
	for i := range events {
		if err := serializer.Emitter.Emit(&events[i]); err != nil {
			return fmt.Errorf("failed to emit event input: %w", err)
		}
	}
	return nil
}

func writeTokenContract(infos []*TokenInfo, profuse, compact bool) error {
	for _, info := range infos {
		copy := *info
		if !profuse {
			copy.Pos = ""
		}
		if err := writeContractItem(&copy, compact); err != nil {
			return err
		}
	}
	return nil
}

func writeEventContract(infos []*EventInfo, profuse, compact bool) error {
	for _, info := range infos {
		copy := *info
		if !profuse {
			copy.Pos = ""
		}
		if err := writeContractItem(&copy, compact); err != nil {
			return err
		}
	}
	return nil
}

func writeContractItem(item any, compact bool) error {
	var node yaml.Node
	if err := node.Dump(item); err != nil {
		return err
	}
	if compact {
		node.Style = yaml.FlowStyle
	}
	var buf bytes.Buffer
	dumper, err := yaml.NewDumper(&buf)
	if err != nil {
		return err
	}
	if err := dumper.Dump([]*yaml.Node{&node}); err != nil {
		return err
	}
	if err := dumper.Close(); err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, &buf)
	return err
}

func writeNodeContract(nodes []*yaml.Node, profuse bool) error {
	values := make([]any, 0, len(nodes))
	for _, node := range nodes {
		values = append(values, mapNodeForOutput(node, profuse))
	}
	var output any = values
	if len(values) == 1 {
		output = values[0]
	}
	dumper, err := yaml.NewDumper(os.Stdout)
	if err != nil {
		return err
	}
	if err := dumper.Dump(output); err != nil {
		return err
	}
	return dumper.Close()
}

func mapNodeForOutput(node *yaml.Node, profuse bool) any {
	if profuse {
		return FormatNode(*node, true)
	}
	return FormatNodeCompact(*node)
}

func writeNodesAsYAML(nodes []*yaml.Node, preserve bool, opts []yaml.Option) error {
	dumper, err := yaml.NewDumper(os.Stdout, opts...)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		if node.Kind == yaml.StreamNode {
			continue
		}
		if preserve {
			if err := dumper.Dump(node); err != nil {
				return err
			}
			continue
		}
		var value any
		if err := node.Load(&value, opts...); err != nil {
			return err
		}
		if err := dumper.Dump(value); err != nil {
			return err
		}
	}
	return dumper.Close()
}

func parsePosition(value string, index int) (libyaml.Mark, libyaml.Mark, error) {
	start := libyaml.Mark{Index: index, Line: 1, Column: index + 1}
	end := start
	if value == "" {
		return start, end, nil
	}
	parts := strings.SplitN(value, "-", 2)
	parsedStart, err := parsePoint(parts[0])
	if err != nil {
		return start, end, err
	}
	parsedStart.Index = index
	start = parsedStart
	end = start
	if len(parts) == 2 {
		if strings.Contains(parts[1], ":") {
			end, err = parsePoint(parts[1])
		} else {
			end.Column, err = strconv.Atoi(parts[1])
		}
		if err != nil {
			return start, end, err
		}
		end.Index = index
	}
	return start, end, nil
}

func parsePoint(value string) (libyaml.Mark, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return libyaml.Mark{}, fmt.Errorf("expected line:column, got %q", value)
	}
	line, err := strconv.Atoi(parts[0])
	if err != nil {
		return libyaml.Mark{}, err
	}
	column, err := strconv.Atoi(parts[1])
	if err != nil {
		return libyaml.Mark{}, err
	}
	return libyaml.Mark{Line: line, Column: column}, nil
}

func parseVersion(value string) (int, int, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid YAML version %q", value)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return major, minor, nil
}

func parseEncoding(value string) libyaml.Encoding {
	switch strings.ToUpper(value) {
	case "UTF-16LE":
		return libyaml.UTF16LE_ENCODING
	case "UTF-16BE":
		return libyaml.UTF16BE_ENCODING
	case "", "ANY":
		return libyaml.ANY_ENCODING
	default:
		return libyaml.UTF8_ENCODING
	}
}

func parseNodeEncoding(value string) yaml.Encoding {
	return parseEncoding(value)
}

func parseScalarStyle(value string) libyaml.ScalarStyle {
	switch strings.ToLower(value) {
	case "double":
		return libyaml.DOUBLE_QUOTED_SCALAR_STYLE
	case "single":
		return libyaml.SINGLE_QUOTED_SCALAR_STYLE
	case "literal":
		return libyaml.LITERAL_SCALAR_STYLE
	case "folded":
		return libyaml.FOLDED_SCALAR_STYLE
	default:
		return libyaml.PLAIN_SCALAR_STYLE
	}
}

func parseSequenceStyle(value string) libyaml.SequenceStyle {
	if strings.EqualFold(value, "flow") {
		return libyaml.FLOW_SEQUENCE_STYLE
	}
	return libyaml.BLOCK_SEQUENCE_STYLE
}

func parseMappingStyle(value string) libyaml.MappingStyle {
	if strings.EqualFold(value, "flow") {
		return libyaml.FLOW_MAPPING_STYLE
	}
	return libyaml.BLOCK_MAPPING_STYLE
}
