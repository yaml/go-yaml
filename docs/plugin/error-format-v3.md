# Error-format v3 Plugin

The v3 error formatting plugin renders YAML load and dump errors in the
go-yaml v3 style:

```text
yaml: line 2: did not find expected node content
yaml: cannot represent type: chan int
```

Use it when migrating from go-yaml v3 or when application tests and logs depend
on the older error string shape.

## Defaults

`yaml.WithV3Defaults()` installs the v3 error formatter automatically.
`yaml.Unmarshal` and `yaml.NewDecoder` use v3 error formatting for loading.
`yaml.Marshal` uses v3 error formatting for dumping.

`yaml.NewEncoder` keeps v3-compatible serialization options and explicitly
selects the v4 error formatter.
Its YAML output settings and error formatting are independent.

Bare `yaml.Load`, `yaml.Dump`, `yaml.NewLoader`, and `yaml.NewDumper` use the
v4 formatter.
`yaml.WithV2Defaults()` and `yaml.WithV4Defaults()` also select v4 formatting.
The v2 preset keeps current error formatting while selecting v2-compatible
processing and output options.
See [default selection](../plugin.md#default-behavior) for the complete mapping
and the historical v2 error style.

## Go Usage

```go
import (
    "go.yaml.in/yaml/v4"
    errorformatv3 "go.yaml.in/yaml/v4/plugin/errorformat/v3"
)

var out any
err := yaml.Load(data, &out, yaml.WithPlugin(errorformatv3.New()))
```

The same plugin applies to dumper errors:

```go
_, err := yaml.Dump(make(chan int), yaml.WithPlugin(errorformatv3.New()))
```

With v3 defaults:

```go
var out any
err := yaml.Load(data, &out, yaml.WithV3Defaults())
```

To combine v2-compatible options with legacy-style error strings, select the
v3 formatter after the preset:

```go
var out any
err := yaml.Load(data, &out,
    yaml.WithV2Defaults(),
    yaml.WithPlugin(errorformatv3.New()),
)
```

## Output

Known line:

```text
yaml: line 1: block sequence entries are not allowed in this context
```

Unknown line:

```text
yaml: cannot construct !!str `error` as a !!float
```

The plugin changes only `error.Error()` output.
Structured error matching with `errors.As` and `errors.Is` still works with
`*yaml.LoadError` and `*yaml.LoadErrors`.

## YAML Configuration

Configure the v3 formatter with `yaml.OptsYAML`:

```yaml
plugin:
  error-format:
    v3:
```

Example:

```go
opts, err := yaml.OptsYAML(`
plugin:
  error-format:
    v3:
`)
if err != nil {
    return err
}
err = yaml.Load(data, &out, opts)
```

## Migration Example

V4 format:

```text
go-yaml load error in scanner at L1.C8: block sequence entries are not allowed in this context
```

V3 format:

```text
yaml: line 1: block sequence entries are not allowed in this context
yaml: cannot represent type: chan int
```

Register `errorformatv3.New()` for code paths that need the v3 string while
keeping the v4 loader behavior and structured error values.
