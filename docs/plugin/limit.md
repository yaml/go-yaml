# Limit Plugin

The limit plugin controls YAML loading safety checks for nesting depth and
alias expansion.
The default go-yaml limits are conservative and are enabled automatically.
Use this plugin when you need stricter limits, looser limits, or custom policy.

## Basic Usage

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/limit"
)

loader, err := yaml.NewLoader(r, yaml.WithPlugin(limit.New()))
if err != nil {
    return err
}
```

With no options, `limit.New()` uses the same limits as go-yaml defaults.

## Depth Limits

Set a maximum nesting depth:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(limit.New(limit.DepthValue(50))),
)
```

Disable depth checking:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(limit.New(limit.DepthNone())),
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
    yaml.WithPlugin(limit.New(limit.DepthFunc(checkDepth))),
)
```

## Alias Limits

Set a simple maximum alias count:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(limit.New(limit.AliasValue(1000))),
)
```

Disable alias checking:

```go
loader, err := yaml.NewLoader(r,
    yaml.WithPlugin(limit.New(limit.AliasNone())),
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
    yaml.WithPlugin(limit.New(limit.AliasFunc(checkAlias))),
)
```

## YAML Configuration

Configure limits with `yaml.OptsYAML`:

```yaml
plugin:
  limit:
    depth: 50
    alias: 1000
```

Use `null` to disable a check:

```yaml
plugin:
  limit:
    depth: null
```

Omitted keys keep their defaults.
A bare `limit:` entry uses all defaults:

```yaml
plugin:
  limit:
```

## Third-Party Limit Plugins

You can implement `yaml.LimitPlugin` directly:

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
