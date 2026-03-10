# Plugin System

The go-yaml v4 plugin system extends YAML processing with custom logic while
maintaining performance, safety and backward compatibility.

## Overview

Plugins allow you to customize certain internal processing during loading and
dumping.
Plugin interfaces use public types and can be implemented by external packages.

## Available Plugins

### Limit Plugin

The limit plugin controls the maximum nesting depth and alias expansion
allowed during parsing.
By default, go-yaml enforces conservative limits to prevent DoS attacks.
Use the limit plugin to relax or tighten those limits.

```go
import "go.yaml.in/yaml/v4/plugin/limit"

// Default limits (same as library defaults)
loader := yaml.NewLoader(data, yaml.WithPlugin(limit.New()))

// Disable alias checking (e.g. for documents with many programmatic aliases)
loader := yaml.NewLoader(data, yaml.WithPlugin(limit.New(limit.AliasNone())))

// Custom depth limit
loader := yaml.NewLoader(data, yaml.WithPlugin(limit.New(limit.DepthValue(50))))
```

#### Limit Options

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
    "go.yaml.in/yaml/v4/plugin/limit"
)

loader := yaml.NewLoader(data, yaml.WithPlugin(limit.New(limit.AliasNone())))
var result any
loader.Load(&result)
```

## Default Behavior

Both bare `NewLoader(data)` and version presets (`WithV4Defaults()`, etc.)
include default limits equivalent to `limit.New()`.

## YAML Configuration

Plugins can be configured from YAML strings using `OptsYAML`:

```go
opts, err := yaml.OptsYAML(`
  plugin:
    limit:
      depth: 50
      alias: 1000
`)
```

Each plugin key maps to a configuration object. For the limit plugin:
- `depth` (int) — max nesting depth; `null` disables depth checking
- `alias` (int) — max alias count; `null` disables alias checking
- Omitted keys keep defaults
- Bare `limit:` (null value) uses all defaults

```yaml
# Disable depth checking, keep default alias limits
plugin:
  limit:
    depth: null
```

## Third-Party Plugins

To write a third-party plugin, implement the `yaml.LimitPlugin`
interface:

```go
type LimitPlugin interface {
    CheckDepth(depth int, ctx *DepthContext) error
    CheckAlias(aliasCount, constructCount int) error
}
```

Pass an instance to `yaml.WithPlugin()` — no import of
`plugin/limit` is needed.

Example:

```go
type StrictLimit struct{}

func (s *StrictLimit) CheckDepth(depth int, ctx *yaml.DepthContext) error {
    if depth > 100 {
        return fmt.Errorf("depth %d exceeds policy limit of 100", depth)
    }
    return nil
}

func (s *StrictLimit) CheckAlias(aliasCount, constructCount int) error {
    if aliasCount > 1000 {
        return fmt.Errorf("alias count %d exceeds policy limit", aliasCount)
    }
    return nil
}

yaml.NewLoader(data, yaml.WithPlugin(&StrictLimit{}))
```
