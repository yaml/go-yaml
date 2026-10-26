# Error-format v4 Plugin

The v4 error formatting plugin renders YAML load and dump errors in the
current go-yaml v4 style:

```text
go-yaml load error in scanner at L2.C6: message
go-yaml dump error in representer: message
```

Bare `yaml.Load`, `yaml.Dump`, `yaml.NewLoader`, and `yaml.NewDumper` use this
formatter.
`yaml.WithV2Defaults()` and `yaml.WithV4Defaults()` also select v4 formatting.

`yaml.NewEncoder` keeps v3-compatible serialization options and explicitly
selects the v4 error formatter.
`yaml.Marshal` uses v3 error formatting.
See [default selection](../plugin.md#default-behavior) for the complete mapping
and the historical v2 error style.

Use this plugin explicitly when you want v4 formatting after another preset, or
when you need custom position text or custom templates.

## Basic Usage

```go
import (
    "go.yaml.in/yaml/v4"
    errorformatv4 "go.yaml.in/yaml/v4/plugin/errorformat/v4"
)

var out any
err := yaml.Load(data, &out, yaml.WithPlugin(errorformatv4.Must()))
```

The same plugin applies to dumper errors:

```go
_, err := yaml.Dump(make(chan int), yaml.WithPlugin(errorformatv4.Must()))
```

Use `New` when options may return an error:

```go
formatter, err := errorformatv4.New(
    errorformatv4.WithPositionStyle(errorformatv4.PositionLong),
)
if err != nil {
    return err
}
err = yaml.Load(data, &out, yaml.WithPlugin(formatter))
```

## Position Styles

`WithPositionStyle` controls how the built-in template renders positions.

Short positions are the default:

```go
errorformatv4.Must(errorformatv4.WithPositionStyle(errorformatv4.PositionShort))
```

```text
go-yaml load error in scanner at L1.C8: message
```

Long positions use words:

```go
errorformatv4.Must(errorformatv4.WithPositionStyle(errorformatv4.PositionLong))
```

```text
go-yaml load error in scanner at line 1, column 8: message
```

Line-only positions omit columns:

```go
errorformatv4.Must(errorformatv4.WithPositionStyle(errorformatv4.PositionLine))
```

```text
go-yaml load error in scanner at line 1: message
```

## Load Templates

Use `WithLoadTemplate` for full control over rendered load errors.
Templates use Go's `text/template` package.

```go
formatter, err := errorformatv4.New(
    errorformatv4.WithLoadTemplate("{{.Stage}} at {{pos .Mark}}: {{.Message}}"),
)
if err != nil {
    return err
}
err = yaml.Load(data, &out, yaml.WithPlugin(formatter))
```

Example output:

```text
scanner at L1.C8: block sequence entries are not allowed in this context
```

### Template Data

| Field | Description |
|---|---|
| `Stage` | Load stage, such as `scanner`, `parser`, or `constructor`. |
| `Message` | Error message without position or stage prefix. |
| `Mark` | Primary error position. |
| `ContextMark` | Optional context position. |
| `ContextMsg` | Optional context message. |
| `HasContext` | True when `ContextMsg` is set. |

### Template Functions

| Function | Description |
|---|---|
| `pos .Mark` | Render one mark with the configured position style. |
| `rangePos .ContextMark .Mark` | Render a range with the configured position style. |
| `line .Mark` | Render a line-only position. |
| `lineCol .Mark` | Render a long line/column position. |

Template parse errors are returned by `errorformatv4.New`.
`errorformatv4.Must` panics on invalid templates and is intended for known-good
static configuration.

## Dump Templates

Use `WithDumpTemplate` for full control over rendered dump errors:

```go
formatter, err := errorformatv4.New(
    errorformatv4.WithDumpTemplate("{{.Stage}}: {{.Message}}"),
)
if err != nil {
    return err
}
_, err = yaml.Dump(make(chan int), yaml.WithPlugin(formatter))
```

Dump templates receive:

| Field | Description |
|---|---|
| `Stage` | Dump stage, such as `representer`, `serializer`, `emitter`, or `writer`. |
| `Message` | Error message without stage prefix. |

## Context Errors

Some parser and scanner errors include context:

```text
go-yaml load error in scanner (while scanning a simple key) at L3.C1-L4.C1: could not find expected ':'
```

The default template includes context when `HasContext` is true.
A custom template can make the same choice:

```gotemplate
{{if .HasContext}}{{.Stage}} ({{.ContextMsg}}) at {{rangePos .ContextMark .Mark}}: {{.Message}}{{else}}{{.Stage}} at {{pos .Mark}}: {{.Message}}{{end}}
```

## YAML Configuration

Configure v4 formatting with `yaml.OptsYAML`:

```yaml
plugin:
  error-format:
    v4:
      position: long
```

Use load and dump templates:

```yaml
plugin:
  error-format:
    v4:
      load-template: '{{.Stage}} at {{pos .Mark}}: {{.Message}}'
      dump-template: '{{.Stage}}: {{.Message}}'
```

`position` accepts `short`, `long`, or `line`.
Set `plugin.error-format` to `true` to use v4 defaults:

```yaml
plugin:
  error-format: true
```

## Third-Party Error Formatting Plugins

You can implement `yaml.ErrorFormatPlugin` directly when template
configuration is not enough:

```go
type MessageOnlyErrors struct{}

func (m MessageOnlyErrors) FormatLoadError(err *yaml.LoadError) string {
    return err.Message
}

func (m MessageOnlyErrors) FormatDumpError(err *yaml.DumpError) string {
    return err.Message
}

err := yaml.Load(data, &out, yaml.WithPlugin(MessageOnlyErrors{}))
```
