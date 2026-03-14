# Errfmt v3 Plugin

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
The legacy APIs `yaml.Unmarshal`, `yaml.Marshal`, `yaml.NewDecoder`, and
`yaml.NewEncoder` use v3 defaults.

Bare `yaml.Load`, `yaml.NewLoader`, `yaml.WithV2Defaults()`, and
`yaml.WithV4Defaults()` use the v4 formatter unless you explicitly register
the v3 plugin.

## Go Usage

```go
import (
    "go.yaml.in/yaml/v4"
    errfmtv3 "go.yaml.in/yaml/v4/plugin/errfmt/v3"
)

var out any
err := yaml.Load(data, &out, yaml.WithPlugin(errfmtv3.New()))
```

The same plugin applies to dumper errors:

```go
_, err := yaml.Dump(make(chan int), yaml.WithPlugin(errfmtv3.New()))
```

With v3 defaults:

```go
var out any
err := yaml.Load(data, &out, yaml.WithV3Defaults())
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
  errfmt:
    v3:
```

Example:

```go
opts, err := yaml.OptsYAML(`
plugin:
  errfmt:
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

Register `errfmtv3.New()` for code paths that need the v3 string while keeping
the v4 loader behavior and structured error values.
