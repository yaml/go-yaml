// Copyright 2026 The go-yaml Project Contributors
// SPDX-License-Identifier: Apache-2.0

package aliasdata

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4/internal/libyaml"
)

// Load constructs a mapping-key directive that imports the paired filename,
// defines its data under name, and omits the entry from the resulting mapping.
// Bind it to any tag using functiontag.New. An Alias-Data plugin must be selected.
func Load(context libyaml.FunctionTagContext, name string) (any, error) {
	if context.Role != "key" || context.ConstructValue == nil {
		return nil, errors.New("alias-data/load requires a mapping key")
	}
	argument := context.Node
	if strings.HasSuffix(context.Tag, "*") && len(argument.Content) == 1 {
		argument = argument.Content[0]
	}
	if argument.Kind != libyaml.ScalarNode || argument.ShortTag() != "!!str" {
		return nil, errors.New("alias-data/load alias name must be a string")
	}
	value, err := context.ConstructValue()
	if err != nil {
		return nil, err
	}
	path, ok := value.(string)
	if !ok || strings.TrimSpace(path) == "" {
		return nil, errors.New("alias-data/load filename must be a non-empty string")
	}
	value, err = context.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load %q: %w", path, err)
	}
	if err := context.DefineAlias(name, value); err != nil {
		return nil, err
	}
	return libyaml.Omit, nil
}
