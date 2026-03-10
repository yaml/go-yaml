# Plugin System

The go-yaml v4 plugin system extends YAML processing with custom logic while
maintaining performance, safety and backward compatibility.

## Overview

Plugins allow you to customize certain internal processing during loading and
dumping.
Plugin interfaces use public types and can be implemented by external packages.

## Available Plugins

### Limits Plugin

The limits plugin controls the maximum nesting depth and alias expansion
allowed during parsing.
By default, go-yaml enforces conservative limits to prevent DoS attacks.
Use the limits plugin to relax or tighten those limits.

```go
import "go.yaml.in/yaml/v4/plugin/limits"

// Default limits (same as library defaults)
loader := yaml.NewLoader(data, yaml.WithPlugin(limits.New()))

// Disable alias checking (e.g. for documents with many programmatic aliases)
loader := yaml.NewLoader(data, yaml.WithPlugin(limits.New(limits.AliasNone())))

// Custom depth limit
loader := yaml.NewLoader(data, yaml.WithPlugin(limits.New(limits.DepthValue(50))))
```

#### Limits Options

| Option | Effect |
|---|---|
| `DepthValue(n)` | Max nesting depth (both flow and block) |
| `DepthNone()` | Disable depth checking |
| `DepthFunc(fn)` | Custom `func(depth int, ctx *yaml.DepthContext) error` |
| `AliasValue(n)` | Max alias expansion count (simple threshold) |
| `AliasNone()` | Disable alias ratio checking |
| `AliasFunc(fn)` | Custom `func(aliasCount, constructCount int) error` |

## Using Plugins

### Basic Usage

Register plugins with `WithPlugin()`:

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/limits"
)

loader := yaml.NewLoader(data, yaml.WithPlugin(limits.New(limits.AliasNone())))
var result interface{}
loader.Load(&result)
```

### Resetting to Defaults

Use `WithoutPlugin()` to reset a plugin kind to library defaults:

```go
// Reset limits to defaults (overrides any previous WithPlugin)
loader := yaml.NewLoader(data, yaml.WithoutPlugin("limits"))
```

## Default Behavior

Both bare `NewLoader(data)` and version presets (`WithV4Defaults()`, etc.)
include default limits equivalent to `limits.New()`.

## Third-Party Plugins

Implement [yaml.LimitsPlugin] directly for full control:

```go
type StrictLimits struct{}

func (s *StrictLimits) CheckDepth(depth int, ctx *yaml.DepthContext) error {
    if depth > 100 {
        return fmt.Errorf("depth %d exceeds policy limit of 100", depth)
    }
    return nil
}

func (s *StrictLimits) CheckAlias(aliasCount, constructCount int) error {
    if aliasCount > 1000 {
        return fmt.Errorf("alias count %d exceeds policy limit", aliasCount)
    }
    return nil
}

yaml.NewLoader(data, yaml.WithPlugin(&StrictLimits{}))
```
