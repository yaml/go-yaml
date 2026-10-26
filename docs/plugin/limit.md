# Limit Plugin

The limit plugin controls YAML loading safety checks for nesting depth and
alias expansion.
The default go-yaml limits are conservative and are enabled automatically.
Use this plugin when you need stricter limits, looser limits, or custom policy.

## Basic Usage

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/loaderlimits"
)

loader, err := yaml.NewLoader(r, yaml.WithPlugin(loaderlimits.New()))
if err != nil {
    return err
}
```

With no options, `loaderlimits.New()` uses the same limits as go-yaml defaults.

## Depth Limits

Set a maximum nesting depth:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthValue(50))),
)
```

Disable depth checking:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthNone())),
)
```

Use a custom depth check:

```go
checkDepth := func(depth int, ctx *yaml.DepthContext) error {
    if ctx.Kind == yaml.DepthKindFlow && depth > 25 {
        return fmt.Errorf("flow depth %d exceeds limit", depth)
    }
    if depth > 100 {
        return fmt.Errorf("depth %d exceeds limit", depth)
    }
    return nil
}

loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthFunc(checkDepth))),
)
```

## Alias Limits

Set a simple maximum alias count:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasValue(1000))),
)
```

Disable alias checking:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasNone())),
)
```

Use a custom alias check:

```go
checkAlias := func(aliasCount, constructCount int) error {
    if aliasCount > 0 && aliasCount > constructCount/2 {
        return fmt.Errorf("too many aliases")
    }
    return nil
}

loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasFunc(checkAlias))),
)
```

## YAML Configuration

Configure limits with `yaml.OptsYAML`:

```yaml
plugin:
  loader-limits:
    depth: 50
    alias: 1000
```

Use `null` to disable a check:

```yaml
plugin:
  loader-limits:
    depth: null
```

Omitted keys keep their defaults.
`loader-limits: true` uses all defaults:

```yaml
plugin:
  loader-limits: true
```

## Third-Party Limit Plugins

You can implement `yaml.LoaderLimitsPlugin` directly:

```go
type StrictLimit struct{}

func (s StrictLimit) CheckDepth(depth int, ctx *yaml.DepthContext) error {
    if depth > 100 {
        return fmt.Errorf("depth %d exceeds limit", depth)
    }
    return nil
}

func (s StrictLimit) CheckAlias(aliasCount, constructCount int) error {
    if aliasCount > 1000 {
        return fmt.Errorf("alias count %d exceeds limit", aliasCount)
    }
    return nil
}

loader, err := yaml.NewLoader(r, yaml.WithPlugin(StrictLimit{}))
```
