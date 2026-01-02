//
// Copyright (c) 2025 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0
//

package yaml

import (
	"errors"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Option allows configuring YAML loading and dumping operations.
// Re-exported from internal/libyaml.
type Option = libyaml.Option

// Re-export option functions from internal/libyaml
var (
	WithIndent                = libyaml.WithIndent
	WithCompactSeqIndent      = libyaml.WithCompactSeqIndent
	WithKnownFields           = libyaml.WithKnownFields
	WithSingleDocument        = libyaml.WithSingleDocument
	WithLineWidth             = libyaml.WithLineWidth
	WithUnicode               = libyaml.WithUnicode
	WithUniqueKeys            = libyaml.WithUniqueKeys
	WithCanonical             = libyaml.WithCanonical
	WithLineBreak             = libyaml.WithLineBreak
	WithExplicitStart         = libyaml.WithExplicitStart
	WithExplicitEnd           = libyaml.WithExplicitEnd
	WithFlowSimpleCollections = libyaml.WithFlowSimpleCollections
)

// Options combines multiple options into a single Option.
// This is useful for creating option presets or combining version defaults
// with custom options.
//
// Example:
//
//	opts := yaml.Options(yaml.V4, yaml.WithIndent(3))
//	yaml.Dump(&data, opts)
func Options(opts ...Option) Option {
	return libyaml.CombineOptions(opts...)
}

// OptsYAML parses a YAML string containing option settings and returns
// an Option that can be combined with other options using Options().
//
// The YAML string can specify any of these fields:
//   - indent (int)
//   - compact-seq-indent (bool)
//   - line-width (int)
//   - unicode (bool)
//   - canonical (bool)
//   - line-break (string: ln, cr, crln)
//   - explicit-start (bool)
//   - explicit-end (bool)
//   - flow-simple-coll (bool)
//   - known-fields (bool)
//   - single-document (bool)
//   - unique-keys (bool)
//
// Only fields specified in the YAML will override other options when
// combined. Unspecified fields won't affect other options.
//
// Example:
//
//	opts, err := yaml.OptsYAML(`
//	  indent: 3
//	  known-fields: true
//	`)
//	yaml.Dump(&data, yaml.Options(V4, opts))
func OptsYAML(yamlStr string) (Option, error) {
	var cfg struct {
		Indent                *int    `yaml:"indent"`
		CompactSeqIndent      *bool   `yaml:"compact-seq-indent"`
		LineWidth             *int    `yaml:"line-width"`
		Unicode               *bool   `yaml:"unicode"`
		Canonical             *bool   `yaml:"canonical"`
		LineBreak             *string `yaml:"line-break"`
		ExplicitStart         *bool   `yaml:"explicit-start"`
		ExplicitEnd           *bool   `yaml:"explicit-end"`
		FlowSimpleCollections *bool   `yaml:"flow-simple-coll"`
		KnownFields           *bool   `yaml:"known-fields"`
		SingleDocument        *bool   `yaml:"single-document"`
		UniqueKeys            *bool   `yaml:"unique-keys"`
	}
	if err := Load([]byte(yamlStr), &cfg, WithKnownFields()); err != nil {
		return nil, err
	}

	// Build options only for fields that were set
	var optList []Option
	if cfg.Indent != nil {
		optList = append(optList, WithIndent(*cfg.Indent))
	}
	if cfg.CompactSeqIndent != nil {
		optList = append(optList, WithCompactSeqIndent(*cfg.CompactSeqIndent))
	}
	if cfg.LineWidth != nil {
		optList = append(optList, WithLineWidth(*cfg.LineWidth))
	}
	if cfg.Unicode != nil {
		optList = append(optList, WithUnicode(*cfg.Unicode))
	}
	if cfg.ExplicitStart != nil {
		optList = append(optList, WithExplicitStart(*cfg.ExplicitStart))
	}
	if cfg.ExplicitEnd != nil {
		optList = append(optList, WithExplicitEnd(*cfg.ExplicitEnd))
	}
	if cfg.FlowSimpleCollections != nil {
		optList = append(optList, WithFlowSimpleCollections(*cfg.FlowSimpleCollections))
	}
	if cfg.KnownFields != nil {
		optList = append(optList, WithKnownFields(*cfg.KnownFields))
	}
	if cfg.SingleDocument != nil && *cfg.SingleDocument {
		optList = append(optList, WithSingleDocument())
	}
	if cfg.UniqueKeys != nil {
		optList = append(optList, WithUniqueKeys(*cfg.UniqueKeys))
	}
	if cfg.Canonical != nil {
		optList = append(optList, WithCanonical(*cfg.Canonical))
	}
	if cfg.LineBreak != nil {
		switch *cfg.LineBreak {
		case "ln":
			optList = append(optList, WithLineBreak(LineBreakLN))
		case "cr":
			optList = append(optList, WithLineBreak(LineBreakCR))
		case "crln":
			optList = append(optList, WithLineBreak(LineBreakCRLN))
		default:
			return nil, errors.New("yaml: invalid line-break value (use ln, cr, or crln)")
		}
	}

	return Options(optList...), nil
}

// V2 provides go-yaml v2 formatting defaults:
//   - 2-space indentation
//   - Non-compact sequence indentation
//   - 80-character line width
//   - Unicode enabled
//   - Unique keys enforced
//
// Usage:
//
//	yaml.Dump(&data, yaml.V2)
//	yaml.Dump(&data, yaml.V2, yaml.WithIndent(4))
var V2 = Options(
	WithIndent(2),
	WithCompactSeqIndent(false),
	WithLineWidth(80),
	WithUnicode(true),
	WithUniqueKeys(true),
)

// V3 provides go-yaml v3 formatting defaults:
//   - 4-space indentation (classic go-yaml v3 style)
//   - Non-compact sequence indentation
//   - 80-character line width
//   - Unicode enabled
//   - Unique keys enforced
//
// Usage:
//
//	yaml.Dump(&data, yaml.V3)
//	yaml.Dump(&data, yaml.V3, yaml.WithIndent(2))
var V3 = Options(
	WithIndent(4),
	WithCompactSeqIndent(false),
	WithLineWidth(80),
	WithUnicode(true),
	WithUniqueKeys(true),
)

// V4 provides go-yaml v4 formatting defaults:
//   - 2-space indentation (more compact than v3)
//   - Compact sequence indentation
//   - 80-character line width
//   - Unicode enabled
//   - Unique keys enforced
//
// Usage:
//
//	yaml.Dump(&data, yaml.V4)
var V4 = Options(
	WithIndent(2),
	WithCompactSeqIndent(true),
	WithLineWidth(80),
	WithUnicode(true),
	WithUniqueKeys(true),
)
