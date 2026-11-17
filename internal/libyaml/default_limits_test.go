// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package libyaml

import (
	"fmt"
	"strings"
	"testing"
)

func TestDefaultOptionsDepthLimits(t *testing.T) {
	for _, kind := range []string{"flow", "block"} {
		for _, depth := range []int{20, 10001} {
			t.Run(fmt.Sprintf("%s/depth=%d", kind, depth), func(t *testing.T) {
				data := strings.Repeat("[", depth) + "x" + strings.Repeat("]", depth)
				if kind == "block" {
					data = strings.Repeat("- ", depth) + "x\n"
				}
				compose := func() (err error) {
					defer handleErr(&err)
					composer := NewComposer([]byte(data), DefaultOptions)
					defer composer.Destroy()
					composer.Compose()
					return nil
				}
				err := compose()
				if depth > 10000 {
					if err == nil || !strings.Contains(err.Error(), "exceeded max depth of 10000") {
						t.Fatalf("got %v, want depth limit error", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
