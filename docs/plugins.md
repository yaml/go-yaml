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

The `plugin` field must be a mapping whose keys identify plugin APIs.
Each API maps to a configuration object, `true` for defaults, or `false` to
leave that plugin disabled.
The host fields `name`, `version`, and `disable` are removed before the
remaining plugin-specific settings reach the plugin factory.
`name` selects an implementation and defaults to the API key.
`version` requires the linked implementation to match that release exactly.
Release versions may be written with or without a leading `v`.
Any plugin mapping can use `disable: true` to have the same effect as `false`,
even when other settings are present.
`disable: false` leaves the plugin enabled.
The `disable` value must be a boolean.
A null plugin value is invalid.
Disabling `limit` leaves the built-in default limits active.
The limit plugin accepts these settings:

- `depth` (int) - max nesting depth; `null` disables depth checking
- `alias` (int) - max alias count; `null` disables alias checking
- Omitted keys keep defaults
- `limit: true` uses all defaults

```yaml
# Disable depth checking, keep default alias limits
plugin:
  limit:
    depth: null
```

Keep a plugin's settings for later while leaving it unselected:

```yaml
plugin:
  limit:
    depth: 3
    alias: 100
    disable: true
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

Pass an instance to `yaml.WithPlugin()`; no import of
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

## Event Source Plugins

An event-source plugin supplies a complete event stream for a load operation.
The resulting events still pass through go-yaml's composer, resolver, and
constructor, including configured alias limits and value conversion.
Only one event-source plugin can be selected.

```go
type EventSourcePlugin interface {
    Parse(input []byte) ([]Event, error)
}
```

`Event` represents stream/document boundaries, mappings, sequences,
scalars, and aliases using ordinary Go fields.
See its Go documentation for event names and optional metadata.
The loader validates the complete event stream before composing nodes.
Depth limits apply to that event stream; they cannot limit work already
performed inside the plugin's parser.

Input is read once when loading begins.
The plugin returns the full event stream, and subsequent `Loader.Load` calls
consume its documents.
Plugin implementations must support independent concurrent calls.
Reader and parser errors retain their underlying cause.

### JSON Comments

The optional `go.yaml.in/yaml/v4/plugin/json-comments` module reuses YAMLStar's
Gloat-generated JSON-comments processor and reference parser.
It requires Go 1.24 or newer and Glojure, with no C compiler or shared library.
The core go-yaml module does not acquire these dependencies.

```go
import (
    "go.yaml.in/yaml/v4"
    jsoncomments "go.yaml.in/yaml/v4/plugin/json-comments"
)

var value any
err := yaml.Load([]byte("a: true // comment\n"), &value,
    yaml.WithPlugin(jsoncomments.New()))
```

The plugin accepts UTF-8 YAML containing `//` and non-nesting `/* */` comments.
It preserves comment markers inside quoted and block scalars and URLs.
JSON literals and numbers allow adjacent comments, such as `true// comment`.
Other plain scalars require separation, so `foo// text` remains scalar text.

Comments are discarded, and node positions are unknown (zero).
Styles, tags, anchors, aliases, and available version directives are preserved.
Original tag-directive declarations are not supplied by this parser.
Input is buffered, and syntax follows the reference parser.
Concurrent callers are supported, with parsing serialized by the plugin because
the generated parser shares mutable state.

### Named Configuration

Applications may call `jsoncomments.Register()` once at startup to enable
`yaml.WithNamedPlugin("json-comments")` and this `yaml.OptsYAML` configuration:

```yaml
plugin:
  json-comments:
    name: json-comments
    version: 0.1.8
```

The mapping key `json-comments` identifies the plugin API.
The `name` field selects its `json-comments` implementation, and `version`
requires that implementation release to be linked.
Both fields may be omitted because there is only one implementation here.
`true` and an empty mapping enable the default implementation.
`false` or `{disable: true}` leaves it disabled.
`{disable: false}` also enables it with defaults.
Null and unknown plugin-specific settings are rejected.
Registration alone does not select a plugin.
Direct use of `yaml.WithPlugin(jsoncomments.New())` needs no registration.

External packages can register a factory with
`yaml.RegisterPlugin(yaml.PluginRegistration{...})`.
The registration supplies the API, implementation name, optional release
version, and factory.
The factory receives only plugin-specific settings and returns a plugin or
error.
Registration is synchronized; duplicate API and name pairs are errors.

### Build Selection and Embedded Defaults

Use `make cli CONFIG=options.yaml` to compile the plugins named in a YAML
options file and embed that file as the default configuration:

```yaml
plugin:
  limit:
    depth: 50
    alias: 100
  json-comments: {}
```

The resulting `./go-yaml -j input.yaml` uses these settings automatically.
The configuration file is no longer needed at runtime; edits require a rebuild.
A runtime `-C` file replaces the entire embedded configuration, while `-o`
flags override the selected options.
`--plugin=NAME` selects the default implementation with default settings.
`--plugin=API=NAME` selects an implementation explicitly.

Build selection currently supports `limit` and `json-comments`.
Core-only configurations use the ordinary parser and do not link Glojure.
Invalid configuration fails before the existing binary is replaced.
`make cli` without `CONFIG` restores the ordinary build and defaults.

### Testing the Optional CLI

By default, the build resolves the newest v-prefixed JSON-comments release
directly from its Git repository each time.
Go downloads the selected module through the configured module proxy.
It requires no local checkout of the plugin repository.
For local development, put a checkout with generated parser sources under
`repos/yamlstar-plugin-json-comments` and set `JSON-COMMENTS-LOCAL=1`.
The Make targets create a temporary CLI module and workspace under `.cache/`.
Downloaded module sources use the Go module cache.

```bash
make cli CONFIG=example/json-comments/options.yaml
./go-yaml -j example/json-comments/data.yaml
make test-json-comments
make test-json-comments-race
```

To build with a specific release, use
`plugin: {json-comments: {version: 0.1.8}}` in the options file.
The `v0.1.8` form is also accepted.
`version` selects code at build time and remains in the embedded configuration.
The completed binary validates it against the linked release.
The same options file can therefore be supplied later through `-C`.
A different runtime version request fails with both versions in the error.
`JSON-COMMENTS-LOCAL=1` uses local sources and validates the requested version
against the local plugin manifest.

`make cli CONFIG=example/json-comments/options.yaml` uses the sample
configuration to link and enable JSON-comments in `./go-yaml`.
It performs no runtime downloads or dynamic library loading.
Runtime flags and `-C` can override the embedded configuration.
JSON, YAML, node, and event output modes support the plugin.
Token output and legacy loading modes reject event-source selection.

`make cli` replaces `./go-yaml` with the ordinary build.
The ordinary build reports an unregistered plugin if JSON-comments is selected.
