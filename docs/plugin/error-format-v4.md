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
when you want an alternate position style.

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

`WithPositionStyle` controls how the formatter renders positions.

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

## Context Errors

Some parser and scanner errors include context:

```text
go-yaml load error in scanner (while scanning a simple key) at L3.C1-L4.C1: could not find expected ':'
```

When context is present, the formatter includes its message in parentheses
and renders the range from the context mark to the error mark.
Without context, it renders only the primary error position.
Positions and ranges follow the configured position style.

## YAML Configuration

Configure v4 formatting with `yaml.OptsYAML`:

```yaml
plugin:
  error-format:
    v4:
      position: long
```

`position` accepts `short`, `long`, or `line`.
Set `plugin.error-format` to `true` to use v4 defaults:

```yaml
plugin:
  error-format: true
```

## Third-Party Error Formatting Plugins

The built-in formatter has fixed messages and configurable position styles.
For other representations, implement `yaml.ErrorFormatPlugin` in your
application or a separate plugin package.
The interface requires `FormatLoadError` and `FormatDumpError` methods.
A plugin can accept format strings through its own constructor:

```go
import (
    "fmt"

    "go.yaml.in/yaml/v4"
)

type printfErrors struct {
    loadFormat string
    dumpFormat string
}

func newPrintfErrors(loadFormat, dumpFormat string) *printfErrors {
    return &printfErrors{loadFormat: loadFormat, dumpFormat: dumpFormat}
}

func (p *printfErrors) FormatLoadError(err *yaml.LoadError) string {
    return fmt.Sprintf(p.loadFormat, err.Stage, err.Mark.ShortString(), err.Message)
}

func (p *printfErrors) FormatDumpError(err *yaml.DumpError) string {
    return fmt.Sprintf(p.dumpFormat, err.Stage, err.Message)
}
```

This example plugin chooses its own arguments: stage, primary position, and
message for load errors; stage and message for dump errors.
Those choices belong to this plugin, independently of the built-in formatter.
A plugin author can expose the constructor from their package.

Use the plugin with `yaml.WithPlugin`:

```go
formatter := newPrintfErrors("%[2]s: %[3]s", "%[2]s")
var out any
err := yaml.Load(data, &out, yaml.WithPlugin(formatter))
_, dumpErr := yaml.Dump(make(chan int), yaml.WithPlugin(formatter))
```

The load format above includes position and message; the dump format includes
only the message.
The same plugin implementation can accept different format strings for other
applications.

A plugin can also use `text/template` or fixed formatting in these methods.
Applications choose that implementation explicitly through `yaml.WithPlugin`.
The built-in formatter does not import `text/template`.
