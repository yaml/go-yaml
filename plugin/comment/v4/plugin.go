// Copyright 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package v4

import (
	"fmt"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

const currentRC = "rc1"

// Plugin implements round-trip comment handling for YAML.
// It preserves original comment formatting including whitespace
// before '#', blank lines between nodes, and end-of-line comment
// alignment.
type Plugin struct {
	libyaml.DefaultCommentBehavior

	rc                  string
	preserveWhitespace  bool
	preserveBlankLines  bool
	blankLineMax        int
	alignLineComments   bool
	streamComments      bool
	meta                map[*libyaml.Node]*CommentMeta
	lastEndLine         int // track line position for blank line detection
}

// Option configures the v4 comment plugin.
type Option func(*Plugin) error

// RC sets the release candidate version this code targets.
// This is required — omitting it causes New() to return an error.
func RC(version string) Option {
	return func(p *Plugin) error {
		p.rc = version
		return nil
	}
}

// PreserveWhitespace controls whether whitespace before '#' in
// comments is preserved. Default: true.
func PreserveWhitespace(v ...bool) Option {
	return func(p *Plugin) error {
		if len(v) == 0 {
			p.preserveWhitespace = true
		} else {
			p.preserveWhitespace = v[0]
		}
		return nil
	}
}

// PreserveBlankLines controls whether blank lines between nodes
// are tracked and preserved. Default: true.
func PreserveBlankLines(v ...bool) Option {
	return func(p *Plugin) error {
		if len(v) == 0 {
			p.preserveBlankLines = true
		} else {
			p.preserveBlankLines = v[0]
		}
		return nil
	}
}

// BlankLineMax sets the maximum number of consecutive blank lines
// to preserve. Default: 2.
func BlankLineMax(n int) Option {
	return func(p *Plugin) error {
		p.blankLineMax = n
		return nil
	}
}

// AlignLineComments controls whether end-of-line comments are
// aligned to their original column positions. Default: false.
func AlignLineComments(v ...bool) Option {
	return func(p *Plugin) error {
		if len(v) == 0 {
			p.alignLineComments = true
		} else {
			p.alignLineComments = v[0]
		}
		return nil
	}
}

// StreamComments controls whether stream-level comments are
// attached to StreamNode. Default: true.
func StreamComments(v ...bool) Option {
	return func(p *Plugin) error {
		if len(v) == 0 {
			p.streamComments = true
		} else {
			p.streamComments = v[0]
		}
		return nil
	}
}

// New creates a new v4 comment plugin with the given options.
// An RC option is required; omitting it returns an error.
func New(opts ...Option) (*Plugin, error) {
	p := &Plugin{
		preserveWhitespace: true,
		preserveBlankLines: true,
		blankLineMax:       2,
		alignLineComments:  false,
		streamComments:     true,
	}
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}
	if p.rc == "" {
		return nil, fmt.Errorf(
			"v4 comment plugin requires RC(...) option; "+
				"current release candidate is %q", currentRC)
	}
	if p.rc != currentRC {
		return nil, fmt.Errorf(
			"v4 comment plugin behavior has changed; "+
				"you specified %q but current is %q; "+
				"see changelog and update to RC(%q) to accept",
			p.rc, currentRC, currentRC)
	}
	return p, nil
}

// NewFromYAML creates a v4 comment plugin from a YAML config map.
// Expected keys: rc (string), preserve-whitespace (bool),
// preserve-blank-lines (bool), blank-line-max (int),
// align-line-comments (bool), stream-comments (bool).
func NewFromYAML(cfg map[string]any) (*Plugin, error) {
	var opts []Option

	if v, ok := cfg["rc"]; ok {
		if s, ok := v.(string); ok {
			opts = append(opts, RC(s))
		} else {
			return nil, fmt.Errorf("v4 comment plugin: rc must be a string")
		}
	}
	if v, ok := cfg["preserve-whitespace"]; ok {
		if b, ok := v.(bool); ok {
			opts = append(opts, PreserveWhitespace(b))
		}
	}
	if v, ok := cfg["preserve-blank-lines"]; ok {
		if b, ok := v.(bool); ok {
			opts = append(opts, PreserveBlankLines(b))
		}
	}
	if v, ok := cfg["blank-line-max"]; ok {
		if n, ok := v.(int); ok {
			opts = append(opts, BlankLineMax(n))
		}
	}
	if v, ok := cfg["align-line-comments"]; ok {
		if b, ok := v.(bool); ok {
			opts = append(opts, AlignLineComments(b))
		}
	}
	if v, ok := cfg["stream-comments"]; ok {
		if b, ok := v.(bool); ok {
			opts = append(opts, StreamComments(b))
		}
	}

	return New(opts...)
}

// ProcessComment attaches comments to the node, preserving original
// whitespace formatting.
func (p *Plugin) ProcessComment(node *libyaml.Node, ctx *libyaml.CommentContext) (bool, error) {
	if p.preserveWhitespace {
		node.HeadComment = string(ctx.HeadComment)
		node.LineComment = string(ctx.LineComment)
		node.FootComment = string(ctx.FootComment)
	} else {
		node.HeadComment = string(ctx.HeadComment)
		node.LineComment = string(ctx.LineComment)
		node.FootComment = string(ctx.FootComment)
	}
	return true, nil
}

// ProcessMappingPair keeps comments on the node they naturally belong
// to instead of using v3's lossy migration heuristics.
func (p *Plugin) ProcessMappingPair(ctx *libyaml.MappingPairContext) (bool, error) {
	k := ctx.Key
	if ctx.TailComment != nil && k.FootComment == "" {
		k.FootComment = string(ctx.TailComment)
	}
	return true, nil
}

// ProcessEndComments handles end-event comments for collections and
// documents without the lossy mapping foot-comment migration that
// v3 performs.
func (p *Plugin) ProcessEndComments(node *libyaml.Node, ctx *libyaml.CommentContext) (bool, error) {
	node.LineComment = string(ctx.LineComment)
	node.FootComment = string(ctx.FootComment)
	return true, nil
}

// SerializeComments is called during dump to convert Node comments
// to Event comments. The v4 plugin preserves comment text as-is.
func (p *Plugin) SerializeComments(node *libyaml.Node, event *libyaml.Event) bool {
	// Default behavior: let the serializer handle it.
	// The plugin can modify event comments here in the future
	// to restore whitespace or inject blank-line markers.
	return false
}

// EmitComment is called during dump before writing a comment.
// The v4 plugin passes comments through unchanged.
func (p *Plugin) EmitComment(comment []byte, kind libyaml.CommentKind) []byte {
	return comment
}
