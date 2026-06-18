# Plugin Guides

go-yaml plugins customize specific parts of YAML loading and dumping while
keeping the main API stable.
Plugins are registered with `yaml.WithPlugin(...)`.

## Available Plugins

| Plugin | Import Path | Use When |
|---|---|---|
| [Limit](limit.md) | `go.yaml.in/yaml/v4/plugin/limit` | You need to relax, tighten, or replace YAML depth and alias safety limits. |
| [Errfmt v3](errfmt-v3.md) | `go.yaml.in/yaml/v4/plugin/errfmt/v3` | You need v3-compatible error strings like `yaml: line 2: message`. |
| [Errfmt v4](errfmt-v4.md) | `go.yaml.in/yaml/v4/plugin/errfmt/v4` | You need the v4 error format, different position text, or custom error templates. |

## Registering Plugins

```go
loader, err := yaml.NewLoader(r, yaml.WithPlugin(pluginValue))
```

Multiple plugins may be registered at once:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(limit.New(), errfmtv4.Must()),
)
```

When multiple plugins implement the same plugin interface, the last registered
one wins.

## YAML Configuration

Plugins can also be configured with `yaml.OptsYAML`:

```yaml
plugin:
  limit:
    depth: 50
  errfmt:
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
`yaml.LimitPlugin` or `yaml.ErrorPlugin`, and are passed to
`yaml.WithPlugin(...)`.
