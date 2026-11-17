# Plugin Guides

go-yaml plugins customize specific parts of YAML loading and dumping while
keeping the main API stable.
Pass plugin values with `yaml.WithPlugin(...)`.
Named factories for configuration use `yaml.RegisterPlugin` instead.

## Available Plugins

| Plugin | Import Path | Use When |
|---|---|---|
| [Merge-Key](merge-key.md) | `github.com/pb33f/go-yaml/plugin/mergekey` | You need merge controls or a replacement merge algorithm. |
| [Alias-Data](alias-data.md) | `github.com/pb33f/go-yaml/plugin/aliasdata` | You need external anchor values, stream scope, or rejection of aliases. |
| [Loader-Limits](loader-limits.md) | `github.com/pb33f/go-yaml/plugin/loaderlimits` | You need to relax, tighten, or replace YAML depth and alias safety limits. |
| [Error-format v3](error-format-v3.md) | `github.com/pb33f/go-yaml/plugin/errorformat/v3` | You need v3-compatible error strings like `yaml: line 2: message`. |
| [Error-format v4](error-format-v4.md) | `github.com/pb33f/go-yaml/plugin/errorformat/v4` | You need the v4 error format, different position text, or a starting point for custom formatters. |
| [Tab-indent](../plugin.md#tab-indent-plugin) | `github.com/pb33f/go-yaml/plugin/tabindent` | You need tab-aware input or output indentation. |
| [Reference YAML-parser](../plugin.md#reference-yaml-parser-implementation) | `github.com/pb33f/go-yaml/plugin/yamlparser/reference` | You need the optional reference parser in place of the built-in parser. |

## Selecting Plugins

```go
loader, err := yaml.NewLoader(r, yaml.WithPlugin(pluginValue))
```

Multiple plugins may be passed at once:

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
