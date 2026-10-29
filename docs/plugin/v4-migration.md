# V4-Migration

The `v4-migration` API provides opt-in construction adapters for code moving to
v4.
Its first implementations support the two distinct go-yaml v3 node types.
It does not change processing defaults or migrate source code.

## Choose an adapter

The separate `go.yaml.in/yaml/v4/plugin/v4migration` module provides:

| Constructor | Accepted hook parameter | YAML implementation |
|---|---|---|
| `FromV3()` | `*go.yaml.in/yaml/v3.Node` | `from-v3` |
| `FromGopkgV3()` | `*gopkg.in/yaml.v3.Node` | `from-gopkg-v3` |

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/v4migration"
)

err := yaml.Load(data, &config,
    yaml.WithPlugin(v4migration.FromV3()))
```

Enable both adapters when dependencies use both package paths:

```go
opts := yaml.WithPlugin(
    v4migration.FromV3(),
    v4migration.FromGopkgV3(),
)
err := yaml.Load(data, &config, opts)
```

Native v4 node hooks and portable `UnmarshalYAML(func(any) error) error` hooks
run before migration adapters.
Adapters run in registration order until one handles the destination or
returns an error.
If none handles a known v3 hook, loading returns a migration error.
Importing the module, calling `Register`, and selecting version presets do
not enable an adapter automatically.

## Conversion behavior

Each invocation constructs a new tree using genuine nodes from the selected
v3 package.
It copies kind, style, tag, value, anchor, comments, positions, content, and
aliases while preserving shared references and cycles.
Legacy mutations do not affect the original v4 nodes.
V4-only metadata is omitted.
Stream nodes and other unsupported kinds produce an ordinary load error.

When a legacy hook calls `node.Decode`, it uses the v3 decoder's defaults.
It does not inherit v4 loader options such as `WithKnownFields` or custom
loader limits.
V4 options still apply to construction outside the legacy hook.
Errors retain their underlying causes for `errors.Is` and `errors.As`.

For libraries supporting v2, v3, and v4, prefer the portable callback hook.
It remains supported in core and needs no migration adapter.

## Classic-style calls with options

Core `yaml.Unmarshal` and `yaml.NewDecoder` retain their original signatures.
The optional module supplies option-aware wrappers:

```go
err := v4migration.Unmarshal(data, &config,
    yaml.WithPlugin(v4migration.FromV3()))

decoder := v4migration.NewDecoder(reader,
    yaml.WithPlugin(v4migration.FromV3()))
err = decoder.Decode(&config)
```

Without an explicit adapter, the wrappers reject v3 hooks too.
They apply classic defaults before caller options, preserving empty-input,
first-document, error-formatting, and streaming behavior by default.
Caller options may override those defaults.

The wrapper's `Decoder` is a separate type from `yaml.Decoder`.
It supports `Decode` and `KnownFields`.
Invalid constructor options are reported by each `Decode` call before any
input is read.
Calling `KnownFields` after such an error is safe and does not clear it.

## Named configuration

Register implementations once during startup:

```go
if err := v4migration.Register(); err != nil {
    return err
}
opts, err := yaml.OptsYAML(configYAML)
```

```yaml
plugin:
  v4-migration: from-v3
```

Use `from-gopkg-v3` for the older import path.
Neither implementation is a default, so `v4-migration: true` is an error.
`false` leaves migration disabled.
Adapters accept no implementation-specific settings.
Named configuration selects one adapter; use direct Go options to enable both.

## Implementing adapters

```go
type V4MigrationPlugin interface {
    TryUnmarshalYAML(target any, node *Node) (handled bool, err error)
}
```

`target` is an initialized destination pointer.
Returning `false, nil` declines without modifying it.
Any error stops adapter dispatch and is wrapped as a constructor error.
Implementations must support independent concurrent calls.
The interface has no legacy package dependencies.
Supporting other YAML frameworks is outside the first implementation's scope.
