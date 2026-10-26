# Plugin Guides

go-yaml plugins customize specific parts of YAML loading and dumping while
keeping the main API stable.
Plugins are registered with `yaml.WithPlugin(...)`.

## Available Plugins

| Plugin | Import Path | Use When |
|---|---|---|
| [Merge-Key](merge-key.md) | `go.yaml.in/yaml/v4/plugin/mergekey` | You need merge controls or a replacement merge algorithm. |
| [Limit](limit.md) | `go.yaml.in/yaml/v4/plugin/loaderlimits` | You need to relax, tighten, or replace YAML depth and alias safety limits. |
| [Error-format v3](error-format-v3.md) | `go.yaml.in/yaml/v4/plugin/errorformat/v3` | You need v3-compatible error strings like `yaml: line 2: message`. |
| [Error-format v4](error-format-v4.md) | `go.yaml.in/yaml/v4/plugin/errorformat/v4` | You need the v4 error format, different position text, or custom error templates. |

## Registering Plugins

```go
loader, err := yaml.NewLoader(r, yaml.WithPlugin(pluginValue))
```

Multiple plugins may be registered at once:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(loaderlimits.New(), errorformatv4.Must()),
)
```

When multiple plugins implement the error-formatting or loader-limits
interface, the last registered one wins.

## YAML Configuration

Plugins can also be configured with `yaml.OptsYAML`:

```yaml
plugin:
  loader-limits:
    depth: 50
  error-format:
    v4:
      position: long
```

The resulting option can be combined with other options:

```go
opts, err := yaml.OptsYAML(config)
if err != nil {
    return err
}
err = yaml.Load(data, &out, opts)
```

## Third-Party Plugins

Third-party plugins implement one of the public plugin interfaces, such as
`yaml.LoaderLimitsPlugin`, `yaml.MergeKeyPlugin`, or `yaml.ErrorFormatPlugin`,
and are passed to `yaml.WithPlugin(...)`.
