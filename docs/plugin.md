# Plugin System

The go-yaml v4 plugin system extends YAML processing with focused, optional
behavior while keeping the main API stable.
Plugins are registered with `yaml.WithPlugin(...)`.

## Plugin-Based Defaults

Much of go-yaml's configurable processing behavior is implemented as plugins, and
more internal policy may move behind plugin interfaces over time.
The default behavior is not a separate special mode: it is the result of using
the shipped plugins with their default configurations.

For example, bare `yaml.Load` and `yaml.NewLoader` use the same safety limits
as `limit.New()` and the same error formatting as `errfmtv4.Must()`.
Version presets work the same way: they install a set of default options and
default plugins that match the requested compatibility profile.

This keeps the built-in behavior easy to override without changing the rest of
the load or dump pipeline.
Applications can replace one policy, such as error formatting or alias limits,
while keeping all other defaults intact.

## Available Plugins

| Plugin | Guide | Use When |
|---|---|---|
| Limit | [plugin/limit.md](plugin/limit.md) | You need to relax, tighten, or replace YAML depth and alias safety checks. |
| Errfmt v3 | [plugin/errfmt-v3.md](plugin/errfmt-v3.md) | You need v3-compatible error strings like `yaml: line 2: message`. |
| Errfmt v4 | [plugin/errfmt-v4.md](plugin/errfmt-v4.md) | You need v4 error formatting, long positions, line-only positions, or templates. |

See [plugin/](plugin/) for the plugin guide index.

## Basic Usage

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/limit"
)

loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(limit.New(limit.DepthValue(50))),
)
if err != nil {
    return err
}
```

Multiple plugins can be registered at once:

```go
import (
    "go.yaml.in/yaml/v4"
    errfmtv4 "go.yaml.in/yaml/v4/plugin/errfmt/v4"
    "go.yaml.in/yaml/v4/plugin/limit"
)

loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(
        limit.New(limit.DepthValue(50)),
        errfmtv4.Must(errfmtv4.WithPositionStyle(errfmtv4.PositionLong)),
    ),
)
```

When multiple plugins implement the same plugin interface, the last registered
one wins.

## Default Behavior

Bare `yaml.Load`, `yaml.Dump`, `yaml.NewLoader`, and `yaml.NewDumper` use
v4 defaults:

- default safety limits equivalent to `limit.New()`
- v4 error formatting equivalent to `errfmtv4.Must()`

Version presets select error formatting as follows:

| Preset | Error Formatter |
|---|---|
| `yaml.WithV2Defaults()` | v4 formatter |
| `yaml.WithV3Defaults()` | v3 formatter |
| `yaml.WithV4Defaults()` | v4 formatter |

## YAML Configuration

Plugins can be configured from YAML strings using `yaml.OptsYAML`:

```yaml
plugin:
  limit:
    depth: 50
    alias: 1000
  errfmt:
    v4:
      position: long
```

```go
opts, err := yaml.OptsYAML(configYAML)
if err != nil {
    return err
}
err = yaml.Load(data, &out, opts)
```

See the plugin-specific guides for supported configuration keys.

## Third-Party Plugins

Plugin interfaces use public types and can be implemented by external
packages.

Limit plugins implement `yaml.LimitPlugin`:

```go
type LimitPlugin interface {
    CheckDepth(depth int, ctx *yaml.DepthContext) error
    CheckAlias(aliasCount, constructCount int) error
}
```

Error formatting plugins implement `yaml.ErrorFmtPlugin`:

```go
type ErrorFmtPlugin interface {
    FormatLoadError(err *yaml.LoadError) string
    FormatDumpError(err *yaml.DumpError) string
}
```

Pass third-party plugin values to `yaml.WithPlugin(...)` the same way as
shipped plugins.
