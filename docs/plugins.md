# Plugin System

The go-yaml v4 plugin system extends YAML processing with custom logic while
maintaining performance and backward compatibility.

## Overview

Plugins allow you to customize how YAML is processed during parsing and
dumping.
Plugin interfaces use public types and can be implemented by external packages.

## Available Plugins

### Comment Plugins

Comment plugins control how comments are attached to nodes during parsing.

#### V3 Legacy Comment Plugin

Replicates the comment handling behavior from go-yaml v3:

```go
import "go.yaml.in/yaml/v4/plugin/comment/v3legacy"

loader := yaml.NewLoader(data, yaml.WithPlugin(v3legacy.New()))
```

For simpler use cases, use `WithV3LegacyComments()` instead:

```go
loader := yaml.NewLoader(data, yaml.WithV3LegacyComments())
```

## Using Plugins

### Basic Usage

Register plugins with `WithPlugin()`:

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/comment/v3legacy"
)

loader := yaml.NewLoader(data, yaml.WithPlugin(v3legacy.New()))
var result interface{}
loader.Load(&result)
```

### Multiple Plugins

You can register multiple plugins at once:

```go
loader := yaml.NewLoader(data,
    yaml.WithPlugin(commentPlugin1, commentPlugin2))
```

### Disabling Plugins

Use `WithoutPlugin()` to disable plugin kinds:

```go
// Disable comment processing for performance
loader := yaml.NewLoader(data, yaml.WithoutPlugin("comment"))
```

## Default Behavior

**Important:** In v4, comments are skipped by default for better performance.

- Default (no options): Comments are skipped
- `WithV3LegacyComments()`: Comments are attached (v3 compatibility)
- `WithPlugin(v3legacy.New())`: Comments handled via plugin
- `WithoutPlugin("comment")`: Explicitly skip comments

## Performance Considerations

Comment processing has a performance cost:

1. The scanner must scan and store comment text
2. The parser must track comment positions
3. The composer must attach comments to nodes

When comments aren't needed, use the default behavior or explicitly disable
them with `WithoutPlugin("comment")` for best performance.

## Plugin Kinds

Currently supported plugin kinds:

- `comment`: Controls comment attachment

## Migration from V3

V3 behavior (comments always loaded):
```go
// v3
yaml.Unmarshal(data, &result)
```

V4 equivalent:
```go
// v4 with v3 compatibility
yaml.Unmarshal(data, &result)  // Still works, uses WithV3LegacyComments internally

// or explicitly
yaml.Load(data, &result, yaml.WithV3LegacyComments())
```

V4 performance mode (skip comments):
```go
// v4 default - skip comments for performance
yaml.Load(data, &result)
```
