// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestDetailedNodeContractKey(t *testing.T) {
	for _, source := range []string{
		"a: b\n",
		"a: &x [1, two]\nb: *x\n",
		"---\na: b\n---\n- c\n",
	} {
		cmd := exec.Command(testBinary, "-N")
		cmd.Stdin = strings.NewReader(source)
		output, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(output), "node: Document") ||
			strings.Contains(string(output), "kind:") {
			t.Fatalf("unexpected node contract: %s", output)
		}
		input, err := detectStructuredInput(output, "")
		if err != nil || input.stage != stageNode {
			t.Fatalf("node contract was not detected: %v", err)
		}
		if len(input.nodes) == 0 || input.nodes[0].Kind != yaml.DocumentNode {
			t.Fatal("expected a document node")
		}
	}
}

func TestRejectRetiredNodeKey(t *testing.T) {
	for _, source := range []string{
		"kind: Scalar\nvalue: old\n",
		"node: Scalar\nkind: Scalar\nvalue: old\n",
		"node: Document\ncontent:\n- kind: Scalar\n  value: old\n",
		"node: Document\ncontent:\n- node: Scalar\n  kind: Scalar\n  value: old\n",
	} {
		if _, err := detectStructuredInput([]byte(source), "n"); err == nil {
			t.Fatalf("accepted retired key in node input: %s", source)
		}
	}
	input, err := detectStructuredInput([]byte("kind: Scalar\nvalue: old\n"), "")
	if err != nil || input.stage != stageYAML {
		t.Fatalf("ordinary YAML should remain YAML: %v", err)
	}
}

func TestParseVersionBounds(t *testing.T) {
	for _, value := range []string{"0.0", "1.1", "1.2", "127.127"} {
		t.Run(value, func(t *testing.T) {
			major, minor, err := parseVersion(value)
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%d.%d", major, minor); got != value {
				t.Fatalf("version changed from %q to %q", value, got)
			}
		})
	}
}

func TestCLIRejectsOutOfRangeVersions(t *testing.T) {
	for _, contract := range []struct {
		stage string
		item  string
	}{
		{"token", "token: VERSION-DIRECTIVE"},
		{"event", "event: DOCUMENT-START"},
	} {
		for _, version := range []string{
			"128.1", "1.128", "257.1", "1.257", "257.258",
			"-1.1", "1.-1", "-255.-255",
		} {
			t.Run(contract.stage+"/"+version, func(t *testing.T) {
				cmd := exec.Command(testBinary, "-f", contract.stage, "-Y")
				cmd.Stdin = strings.NewReader(fmt.Sprintf(
					"- {%s: STREAM-START}\n- {%s, version: %s}\n"+
						"- {%s: STREAM-END}\n",
					contract.stage, contract.item, version, contract.stage))
				output, err := cmd.CombinedOutput()
				if err == nil {
					t.Fatal("expected invalid version to fail")
				}
				want := fmt.Sprintf("invalid YAML version %q", version)
				if !strings.Contains(string(output), want) {
					t.Fatalf("expected %q in output: %s", want, output)
				}
			})
		}
	}
}
