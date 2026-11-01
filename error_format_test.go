// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package yaml_test

import (
	"fmt"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
	errorformatv4 "go.yaml.in/yaml/v4/plugin/errorformat/v4"
)

func TestDefaultLoadFormats(t *testing.T) {
	for _, tc := range []struct {
		name              string
		mark, contextMark yaml.Mark
		context           string
		positions         [3]string
	}{
		{
			name:      "point",
			mark:      yaml.Mark{Line: 2, Column: 6},
			positions: [3]string{"L2.C6", "line 2, column 6", "line 2"},
		},
		{
			name: "unknown point",
			positions: [3]string{
				"<unknown position>", "<unknown position>", "<unknown position>",
			},
		},
		{
			name:      "line only",
			mark:      yaml.Mark{Line: 2},
			positions: [3]string{"L2", "line 2", "line 2"},
		},
		{
			name:        "context across lines",
			mark:        yaml.Mark{Line: 2, Column: 6},
			contextMark: yaml.Mark{Line: 1, Column: 3},
			context:     "while parsing",
			positions: [3]string{
				"L1.C3-L2.C6", "line 1, column 3-line 2, column 6", "line 1-line 2",
			},
		},
		{
			name:        "context on same line",
			mark:        yaml.Mark{Line: 1, Column: 8},
			contextMark: yaml.Mark{Line: 1, Column: 3},
			context:     "while parsing",
			positions: [3]string{
				"L1.C3-C8", "line 1, columns 3-8", "line 1",
			},
		},
		{
			name:        "identical marks",
			mark:        yaml.Mark{Line: 2, Column: 6},
			contextMark: yaml.Mark{Line: 2, Column: 6},
			context:     "while parsing",
			positions:   [3]string{"L2.C6", "line 2, column 6", "line 2"},
		},
		{
			name:    "unknown context mark",
			mark:    yaml.Mark{Line: 2, Column: 6},
			context: "while parsing",
			positions: [3]string{
				"<unknown position>-L2.C6", "line 2, column 6", "line 2",
			},
		},
		{
			name:        "unknown error mark",
			contextMark: yaml.Mark{Line: 1, Column: 3},
			context:     "while parsing",
			positions: [3]string{
				"L1.C3-<unknown position>", "line 1, column 3", "line 1",
			},
		},
		{
			name:    "unknown context and error marks",
			context: "while parsing",
			positions: [3]string{
				"<unknown position>", "<unknown position>", "<unknown position>",
			},
		},
		{
			name:        "unknown context column",
			mark:        yaml.Mark{Line: 2, Column: 6},
			contextMark: yaml.Mark{Line: 2},
			context:     "while parsing",
			positions:   [3]string{"L2-L2.C6", "line 2", "line 2"},
		},
		{
			name:        "unknown error column",
			mark:        yaml.Mark{Line: 2},
			contextMark: yaml.Mark{Line: 2, Column: 3},
			context:     "while parsing",
			positions:   [3]string{"L2.C3-L2", "line 2", "line 2"},
		},
		{
			name:        "context mark without message",
			mark:        yaml.Mark{Line: 2, Column: 6},
			contextMark: yaml.Mark{Line: 1, Column: 3},
			positions:   [3]string{"L2.C6", "line 2, column 6", "line 2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for style, position := range tc.positions {
				p := errorformatv4.Must(errorformatv4.WithPositionStyle(
					errorformatv4.PositionStyle(style)))
				err := &yaml.LoadError{
					Stage: yaml.ParserStage, Message: "bad YAML",
					Mark: tc.mark, ContextMark: tc.contextMark, ContextMsg: tc.context,
				}
				want := "go-yaml load error in parser"
				if tc.context != "" {
					want += " (while parsing)"
				}
				want += " at " + position + ": bad YAML"
				if got := p.FormatLoadError(err); got != want {
					t.Errorf("style %d: got %q, want %q", style, got, want)
				}
			}
		})
	}
}

func TestDefaultDumpFormat(t *testing.T) {
	err := &yaml.DumpError{Stage: yaml.RepresenterStage, Message: "bad value"}
	want := "go-yaml dump error in representer: bad value"
	if got := errorformatv4.Must().FormatDumpError(err); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

type printfErrors struct {
	loadFormat string
	dumpFormat string
}

func newPrintfErrors(loadFormat, dumpFormat string) *printfErrors {
	return &printfErrors{loadFormat: loadFormat, dumpFormat: dumpFormat}
}

func (p *printfErrors) FormatLoadError(err *yaml.LoadError) string {
	return fmt.Sprintf(p.loadFormat, err.Stage, err.Mark.ShortString(), err.Message)
}

func (p *printfErrors) FormatDumpError(err *yaml.DumpError) string {
	return fmt.Sprintf(p.dumpFormat, err.Stage, err.Message)
}

var _ yaml.ErrorFormatPlugin = (*printfErrors)(nil)

func TestCustomErrorFormatPlugin(t *testing.T) {
	for _, tc := range []struct {
		name, loadFormat, dumpFormat, wantLoad, wantDump string
	}{
		{
			name:       "with details",
			loadFormat: "%[1]s at %[2]s: %[3]s",
			dumpFormat: "%[1]s: %[2]s",
			wantLoad:   "scanner at L1.C8: block sequence entries are not allowed in this context",
			wantDump:   "representer: cannot represent type: chan int",
		},
		{
			name:       "message only",
			loadFormat: "%[3]s",
			dumpFormat: "%[2]s",
			wantLoad:   "block sequence entries are not allowed in this context",
			wantDump:   "cannot represent type: chan int",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			formatter := newPrintfErrors(tc.loadFormat, tc.dumpFormat)
			var out any
			err := yaml.Load(errorFormatBadYAML, &out, yaml.WithPlugin(formatter))
			if err == nil || err.Error() != tc.wantLoad {
				t.Fatalf("load: got %v, want %q", err, tc.wantLoad)
			}
			_, err = yaml.Dump(make(chan int), yaml.WithPlugin(formatter))
			if err == nil || err.Error() != tc.wantDump {
				t.Fatalf("dump: got %v, want %q", err, tc.wantDump)
			}
		})
	}
}

func TestFormatConfig(t *testing.T) {
	p, err := errorformatv4.NewFromYAML(map[string]any{
		"position": "long",
	})
	if err != nil {
		t.Fatal(err)
	}
	loadErr := &yaml.LoadError{
		Stage: yaml.ParserStage, Mark: yaml.Mark{Line: 2, Column: 6},
		Message: "bad YAML",
	}
	if got := p.FormatLoadError(loadErr); got != "go-yaml load error in parser at line 2, column 6: bad YAML" {
		t.Errorf("load config: got %q", got)
	}
	dumpErr := &yaml.DumpError{Stage: yaml.RepresenterStage, Message: "bad value"}
	if got := p.FormatDumpError(dumpErr); got != "go-yaml dump error in representer: bad value" {
		t.Errorf("dump config: got %q", got)
	}
	for _, tc := range []struct {
		key  string
		val  any
		want string
	}{
		{"load-format", "%[3]s", `unknown key "load-format"`},
		{"dump-format", "%[2]s", `unknown key "dump-format"`},
		{"position", 1, "position must be a string"},
		{"position", "bad", "invalid position"},
		{"load-template", "%[3]s", `unknown key "load-template"`},
		{"dump-template", "%[2]s", `unknown key "dump-template"`},
	} {
		t.Run(tc.key+"/"+tc.want, func(t *testing.T) {
			_, err := errorformatv4.NewFromYAML(map[string]any{tc.key: tc.val})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
	if _, err := errorformatv4.New(errorformatv4.WithPositionStyle(-1)); err == nil {
		t.Fatal("expected invalid position style error")
	}
}
