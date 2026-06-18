# Errfmt v4 Plugin

The v4 error formatting plugin renders YAML load and dump errors in the
current go-yaml v4 style:

```text
go-yaml load error in scanner at L2.C6: message
go-yaml dump error in representer: message
```

It is installed by default for bare `yaml.Load`, `yaml.NewLoader`,
`yaml.WithV2Defaults()`, and `yaml.WithV4Defaults()`.
Use this plugin explicitly when you want v4 formatting after another preset, or
when you need custom position text or custom templates.

## Basic Usage

```go
import (
    "go.yaml.in/yaml/v4"
    errfmtv4 "go.yaml.in/yaml/v4/plugin/errfmt/v4"
)

var out any
err := yaml.Load(data, &out, yaml.WithPlugin(errfmtv4.Must()))
```

The same plugin applies to dumper errors:

```go
_, err := yaml.Dump(make(chan int), yaml.WithPlugin(errfmtv4.Must()))
```

Use `New` when options may return an error:

```go
formatter, err := errfmtv4.New(
    errfmtv4.WithPositionStyle(errfmtv4.PositionLong),
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
errfmtv4.Must(errfmtv4.WithPositionStyle(errfmtv4.PositionShort))
```

```text
go-yaml load error in scanner at L1.C8: message
```

Long positions use words:

```go
errfmtv4.Must(errfmtv4.WithPositionStyle(errfmtv4.PositionLong))
```

```text
go-yaml load error in scanner at line 1, column 8: message
```

Line-only positions omit columns:

```go
errfmtv4.Must(errfmtv4.WithPositionStyle(errfmtv4.PositionLine))
```

```text
go-yaml load error in scanner at line 1: message
```

## Load Templates

Use `WithLoadTemplate` for full control over rendered load errors.
Templates use Go's `text/template` package.

```go
formatter, err := errfmtv4.New(
    errfmtv4.WithLoadTemplate("{{.Stage}} at {{pos .Mark}}: {{.Message}}"),
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

Template parse errors are returned by `errfmtv4.New`.
`errfmtv4.Must` panics on invalid templates and is intended for known-good
static configuration.

`WithTemplate` is kept as an alias for `WithLoadTemplate`.

## Dump Templates

Use `WithDumpTemplate` for full control over rendered dump errors:

```go
formatter, err := errfmtv4.New(
    errfmtv4.WithDumpTemplate("{{.Stage}}: {{.Message}}"),
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
  errfmt:
    v4:
      position: long
```

Use load and dump templates:

```yaml
plugin:
  errfmt:
    v4:
      load-template: '{{.Stage}} at {{pos .Mark}}: {{.Message}}'
      dump-template: '{{.Stage}}: {{.Message}}'
```

`position` accepts `short`, `long`, or `line`.
`template` is also accepted as an alias for `load-template`.
When `plugin.errfmt` is `null`, v4 defaults are used:

```yaml
plugin:
  errfmt:
```

## Third-Party Error Plugins

You can implement `yaml.ErrorPlugin` directly when template configuration is
not enough:

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
