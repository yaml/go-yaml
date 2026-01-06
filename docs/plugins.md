# YAML Plugins Guide

This guide explains how to use and create plugins to extend go-yaml's YAML
processing.

## What Are Plugins?

Plugins let you hook into the YAML processing pipeline to transform, validate,
or inspect YAML data.
They run at specific points:

- **LoadPlugin** - Runs after parsing, before converting to Go values
- **DumpPlugin** - Runs after converting from Go values, before encoding to YAML

Think of plugins as middleware for YAML.

## Quick Example: Comment Preservation

The simplest plugin use case is preserving comments:

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/comment/v3"
)

// Load YAML with comment preservation
loader, _ := yaml.NewLoader(reader,
    yaml.WithPlugin(v3.New()),
)
```

## Built-in Comment Plugin

go-yaml includes a comment handling plugin:

### `plugin/comment/v3` - v3 Comment Handling

Standard comment preservation:

```go
import commentv3 "go.yaml.in/yaml/v4/plugin/comment/v3"

loader, _ := yaml.NewLoader(reader,
    yaml.WithPlugin(commentv3.New()),
)
```

**When to use:**
- You need to preserve comments through load/dump cycles
- You're building tools that manipulate YAML while keeping comments
- Comment preservation is important for your use case

## Comment Fields in Nodes

YAML comments are stored in three fields on `yaml.Node`:

```go
type Node struct {
    HeadComment string  // Comments before the node
    LineComment string  // Comment on the same line
    FootComment string  // Comments after the node
    // ... other fields
}
```

**Example YAML:**

```yaml
# This is a head comment
name: myapp  # This is a line comment
# This is a foot comment
```

**Corresponding Node:**

```go
node := &yaml.Node{
    Kind:        yaml.ScalarNode,
    Value:       "myapp",
    HeadComment: "This is a head comment",
    LineComment: "This is a line comment",
    FootComment: "This is a foot comment",
}
```

## Using Multiple Plugins

You can add multiple plugins - they execute in the order specified:

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/comment/v3"
)

loader, _ := yaml.NewLoader(reader,
    yaml.WithPlugins(v3.New()),
    yaml.WithPlugins(myValidatorPlugin),
    yaml.WithPlugins(myLoggerPlugin),
)
```

**Each plugin receives the node from the previous plugin.**

## Creating Custom Plugins

Creating a plugin is straightforward.
Implement one or more of these interfaces:

```go
type Plugin interface {
    Name() string
}

type LoadPlugin interface {
    Plugin
    ProcessLoadNode(node *yaml.Node) (*yaml.Node, error)
}

type DumpPlugin interface {
    Plugin
    ProcessDumpNode(node *yaml.Node) (*yaml.Node, error)
}
```

### Example 1: Uppercase Plugin

Converts all scalar values to uppercase:

```go
package main

import (
    "strings"
    "go.yaml.in/yaml/v4"
)

type UppercasePlugin struct{}

func (p *UppercasePlugin) Name() string {
    return "uppercase"
}

func (p *UppercasePlugin) ProcessDumpNode(node *yaml.Node) (*yaml.Node, error) {
    // Process this node
    if node.Kind == yaml.ScalarNode {
        node.Value = strings.ToUpper(node.Value)
    }

    // Recursively process children
    for _, child := range node.Content {
        p.ProcessDumpNode(child)
    }

    return node, nil
}

// Usage
dumper, _ := yaml.NewDumper(writer,
    yaml.WithPlugins(&UppercasePlugin{}),
)
```

**Input:**

```yaml
name: hello
city: seattle
```

**Output:**

```yaml
name: HELLO
city: SEATTLE
```

### Example 2: Validation Plugin

Validates nodes during loading:

```go
type ValidationPlugin struct{}

func (p *ValidationPlugin) Name() string {
    return "validator"
}

func (p *ValidationPlugin) ProcessLoadNode(node *yaml.Node) (*yaml.Node, error) {
    // Check for required fields in mapping nodes
    if node.Kind == yaml.MappingNode {
        hasName := false
        for i := 0; i < len(node.Content); i += 2 {
            if node.Content[i].Value == "name" {
                hasName = true
                break
            }
        }

        if !hasName {
            return nil, fmt.Errorf("missing required field 'name' at line %d",
                node.Line)
        }
    }

    // Recursively validate children
    for _, child := range node.Content {
        if _, err := p.ProcessLoadNode(child); err != nil {
            return nil, err
        }
    }

    return node, nil
}

// Usage
loader, _ := yaml.NewLoader(reader,
    yaml.WithPlugins(&ValidationPlugin{}),
)

var data map[string]interface{}
err := loader.Load(&data)  // Will error if 'name' field is missing
```

### Example 3: Logging Plugin

Logs all YAML values during processing:

```go
type LoggingPlugin struct {
    logger *log.Logger
}

func NewLoggingPlugin(logger *log.Logger) *LoggingPlugin {
    return &LoggingPlugin{logger: logger}
}

func (p *LoggingPlugin) Name() string {
    return "logger"
}

func (p *LoggingPlugin) ProcessLoadNode(node *yaml.Node) (*yaml.Node, error) {
    if node.Kind == yaml.ScalarNode {
        p.logger.Printf("Loaded: %s = %s (line %d)",
            node.Tag, node.Value, node.Line)
    }

    for _, child := range node.Content {
        p.ProcessLoadNode(child)
    }

    return node, nil
}

// Usage
logger := log.New(os.Stdout, "[YAML] ", log.LstdFlags)
loader, _ := yaml.NewLoader(reader,
    yaml.WithPlugins(NewLoggingPlugin(logger)),
)
```

### Example 4: Secret Redaction Plugin

Redacts sensitive values in YAML output:

```go
type RedactionPlugin struct {
    patterns []string
}

func NewRedactionPlugin(patterns []string) *RedactionPlugin {
    return &RedactionPlugin{patterns: patterns}
}

func (p *RedactionPlugin) Name() string {
    return "redactor"
}

func (p *RedactionPlugin) ProcessDumpNode(node *yaml.Node) (*yaml.Node, error) {
    // Check if this is a key-value pair in a mapping
    if node.Kind == yaml.MappingNode {
        for i := 0; i < len(node.Content); i += 2 {
            key := node.Content[i].Value
            value := node.Content[i+1]

            // Redact if key matches a pattern
            for _, pattern := range p.patterns {
                if strings.Contains(strings.ToLower(key), pattern) {
                    if value.Kind == yaml.ScalarNode {
                        value.Value = "***REDACTED***"
                    }
                }
            }
        }
    }

    // Recursively process children
    for _, child := range node.Content {
        p.ProcessDumpNode(child)
    }

    return node, nil
}

// Usage
dumper, _ := yaml.NewDumper(writer,
    yaml.WithPlugins(NewRedactionPlugin([]string{"password", "secret", "token"})),
)
```

**Input:**

```yaml
username: admin
password: super_secret
api_token: abc123
port: 8080
```

**Output:**

```yaml
username: admin
password: ***REDACTED***
api_token: ***REDACTED***
port: 8080
```

### Example 5: Default Value Plugin

Adds default values to nodes:

```go
type DefaultsPlugin struct {
    defaults map[string]string
}

func NewDefaultsPlugin(defaults map[string]string) *DefaultsPlugin {
    return &DefaultsPlugin{defaults: defaults}
}

func (p *DefaultsPlugin) Name() string {
    return "defaults"
}

func (p *DefaultsPlugin) ProcessLoadNode(node *yaml.Node) (*yaml.Node, error) {
    if node.Kind != yaml.MappingNode {
        return node, nil
    }

    // Check which default keys are missing
    existingKeys := make(map[string]bool)
    for i := 0; i < len(node.Content); i += 2 {
        existingKeys[node.Content[i].Value] = true
    }

    // Add missing defaults
    for key, value := range p.defaults {
        if !existingKeys[key] {
            keyNode := &yaml.Node{
                Kind:  yaml.ScalarNode,
                Value: key,
            }
            valueNode := &yaml.Node{
                Kind:  yaml.ScalarNode,
                Value: value,
            }
            node.Content = append(node.Content, keyNode, valueNode)
        }
    }

    return node, nil
}

// Usage
defaults := map[string]string{
    "version": "1.0",
    "enabled": "true",
}

loader, _ := yaml.NewLoader(reader,
    yaml.WithPlugins(NewDefaultsPlugin(defaults)),
)
```

## Real-World Examples

### Example: Config File Processor with Validation

```go
type ConfigProcessor struct {
    validator *ValidationPlugin
    defaults  *DefaultsPlugin
    redactor  *RedactionPlugin
}

func NewConfigProcessor() *ConfigProcessor {
    return &ConfigProcessor{
        validator: &ValidationPlugin{},
        defaults: NewDefaultsPlugin(map[string]string{
            "port":    "8080",
            "timeout": "30s",
        }),
        redactor: NewRedactionPlugin([]string{"password", "token"}),
    }
}

func (cp *ConfigProcessor) LoadConfig(r io.Reader) (*Config, error) {
    loader, err := yaml.NewLoader(r,
        yaml.WithPlugins(cp.defaults),    // Add defaults first
        yaml.WithPlugins(cp.validator),   // Then validate
    )
    if err != nil {
        return nil, err
    }

    var config Config
    if err := loader.Load(&config); err != nil {
        return nil, err
    }

    return &config, nil
}

func (cp *ConfigProcessor) SaveConfig(w io.Writer, config *Config) error {
    dumper, err := yaml.NewDumper(w,
        yaml.WithPlugins(cp.redactor),  // Redact sensitive data
        yaml.WithIndent(2),
    )
    if err != nil {
        return err
    }

    if err := dumper.Dump(config); err != nil {
        return err
    }

    return dumper.Close()
}
```

### Example: YAML Pretty Printer

```go
type PrettyPrinter struct{}

func (p *PrettyPrinter) Name() string {
    return "pretty-printer"
}

func (p *PrettyPrinter) ProcessDumpNode(node *yaml.Node) (*yaml.Node, error) {
    // Add spacing between top-level items
    if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
        root := node.Content[0]
        if root.Kind == yaml.MappingNode {
            for i := 0; i < len(root.Content); i += 2 {
                if i > 0 {
                    root.Content[i].HeadComment = "\n"
                }
            }
        }
    }

    for _, child := range node.Content {
        p.ProcessDumpNode(child)
    }

    return node, nil
}

// Usage: Create nicely spaced YAML
dumper, _ := yaml.NewDumper(writer,
    yaml.WithPlugins(&PrettyPrinter{}),
    yaml.WithIndent(2),
)
```

## Plugin Best Practices

### 1. Always Process Children

Plugins should recursively process child nodes:

```go
func (p *MyPlugin) ProcessLoadNode(node *yaml.Node) (*yaml.Node, error) {
    // Process this node
    // ...

    // Process children
    for _, child := range node.Content {
        if _, err := p.ProcessLoadNode(child); err != nil {
            return nil, err
        }
    }

    return node, nil
}
```

### 2. Handle All Node Kinds

Different node kinds need different handling:

```go
switch node.Kind {
case yaml.DocumentNode:
    // Process document
case yaml.MappingNode:
    // Process key-value pairs (Content has keys and values alternating)
case yaml.SequenceNode:
    // Process list items
case yaml.ScalarNode:
    // Process scalar value
case yaml.AliasNode:
    // Handle YAML alias
}
```

### 3. Return Errors Properly

Return errors instead of panicking:

```go
func (p *MyPlugin) ProcessLoadNode(node *yaml.Node) (*yaml.Node, error) {
    if err := validateNode(node); err != nil {
        return nil, fmt.Errorf("validation failed at line %d: %w",
            node.Line, err)
    }
    return node, nil
}
```

### 4. Don't Modify Unless Necessary

Only modify nodes when you need to:

```go
// Good: Only modify what you need
if node.Kind == yaml.ScalarNode && needsChange(node.Value) {
    node.Value = transform(node.Value)
}

// Bad: Unnecessary modifications
node.Style = yaml.DoubleQuotedStyle  // Don't change style unless required
```

### 5. Document Your Plugin

Add clear documentation about what your plugin does:

```go
// CompressionPlugin removes unnecessary whitespace and empty nodes
// to create more compact YAML output.
//
// Example usage:
//     dumper, _ := yaml.NewDumper(w, yaml.WithPlugins(&CompressionPlugin{}))
type CompressionPlugin struct{}
```

## Tips & Tricks

**Order matters:** Plugins run in the order you add them.

**Load vs Dump:** LoadPlugin runs during parsing, DumpPlugin runs during
encoding.

**Both interfaces:** A plugin can implement both LoadPlugin and DumpPlugin.

**Node structure:** Mapping nodes have alternating key/value in Content (key at
i, value at i+1).

**Performance:** Comments are not loaded by default in V4; use the v3 plugin only if you need preservation.

**Debugging:** Add a logging plugin to see what's happening during processing.

**Validation:** Use LoadPlugin to validate structure before unmarshaling.

**Transformation:** Use DumpPlugin to transform output format.

## Common Use Cases

**✓ Preserve comments** - Use `plugin/comment/v3`
**✓ Validate structure** - Create a LoadPlugin that checks required fields
**✓ Redact secrets** - Create a DumpPlugin that masks sensitive values
**✓ Add defaults** - Create a LoadPlugin that inserts missing values
**✓ Transform values** - Create a DumpPlugin that modifies scalars
**✓ Logging/debugging** - Create a plugin that logs all nodes
**✓ Enforce schemas** - Create a LoadPlugin that validates against a schema

## See Also

- [Options Guide](options.md) - Configure YAML formatting with options
- [API Documentation](https://pkg.go.dev/go.yaml.in/yaml/v4) - Full API
  reference
- [Examples](../example/README.md) - Runnable code examples
