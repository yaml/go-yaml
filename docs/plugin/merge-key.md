# Merge-Key

The `merge-key` API selects the algorithm used to load YAML merge directives.
The built-in implementation uses the shallow merge rules from the
[YAML 1.1 merge-type draft](https://yaml.org/type/merge.html).
The mapping containing `<<` is the target; its merge values supply the sources.

## Configuration

```yaml
plugin:
  merge-key:
    multiple: combine
    conflicts: first
```

Merging is enabled by default for plain `<<` and explicit `!!merge` keys.
Quoted `<<` and explicitly string-tagged keys are ordinary keys.
Explicit merge tags can also use the full `tag:yaml.org,2002:merge` URI.

Disable merging with any of these configurations:

```yaml
plugin:
  merge-key: disable
```

```yaml
plugin:
  merge-key: false
```

```yaml
plugin:
  merge-key:
    disable: true
```

When disabled, plain `<<` and its value become ordinary data.
An explicit merge tag produces a construction error.
`disable: false`, `merge-key: true`, and an empty configuration enable merging.

`multiple` accepts `error` or `combine`.
`combine` collects sources from repeated merge directives in document order,
expanding sequence-valued directives in sequence order.
An explicit setting overrides `unique-keys` for recognized merge directives.
When omitted, the existing loader behavior applies: duplicate checking rejects
repeated directives, while `unique-keys: false` uses only the last directive.
Ordinary duplicate keys still follow `unique-keys`, including keys inside a
source mapping and plain `<<` keys when merging is disabled.

`conflicts` accepts `first`, `last`, or `error`.
Candidates are considered in this order: explicit target entries, then each
source mapping.

| Policy | Behavior |
|---|---|
| `first` (default) | Target entries win; earlier sources win over later sources |
| `last` | Later sources override earlier sources and target entries |
| `error` | Overlapping keys across mappings fail, even with equal values |

The position of `<<` relative to explicit target entries does not affect this
order.
Values are selected whole; the default algorithm does not recursively merge
the values of conflicting keys.
Nested merge directives are evaluated independently using the same plugin.
Repeated directives with `multiple: combine` are an extension to the draft.

## Direct Go use

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/mergekey"
)

opts := yaml.WithPlugin(mergekey.New(
    mergekey.WithMultiple("combine"),
    mergekey.WithConflicts("last"),
))
err := yaml.Load(input, &value, opts)
```

Use `mergekey.WithDisable(true)` to disable merging.
The same options work with streaming loaders and `Node.Load`.
Raw node loading preserves the original mapping structure.
Custom unmarshaling continues to receive the original nodes; use `Node.Load`
with the desired options when constructing their values.
Dumping does not apply merge algorithms.

## Replacement algorithms

Implement the public interface:

```go
type MergeKeyPlugin interface {
    MergeKeyConfig() MergeKeyConfig
    MergeMappings(MergeKeyInput) (*Node, error)
}
```

`MergeKeyConfig` has `Disable bool`, `Multiple string`, and `Conflicts string`.
Empty `Multiple` inherits the loader's duplicate-key setting.
Empty `Conflicts` selects `first` in the default implementation.
Replacement algorithms receive the configuration through their own factory
and can implement different conflict behavior.

`MergeKeyInput.Target` contains explicit target entries without merge
directives.
`Sources` contains ordered mapping nodes after alias resolution and nested
merges through the selected algorithm.
`Key(node)` constructs a comparable key using the destination's key type.
Use it to compare keys consistently with typed maps and struct fields.
It returns construction errors for invalid keys.

Input node graphs are read-only.
The callback is valid only during the call; retain no operation-local state.
Return a mapping node with paired, non-nil key/value children.
New nodes are allowed, so algorithms can compute values as well as select
existing entries.
The loader performs typed value construction after the algorithm returns.
Implementations must support independent concurrent calls.
Alias limits and merge-cycle checks remain the loader's responsibility.

Pass an implementation directly with `yaml.WithPlugin`, or register a factory:

```go
err := yaml.RegisterPlugin(yaml.PluginRegistration{
    API: "merge-key", Name: "custom",
    Factory: func(config map[string]any) (any, error) {
        return newCustomMerge(config)
    },
})
```

Then select `plugin: {merge-key: {name: custom}}` or
`go-yaml --plugin=merge-key=custom`.
The CLI disable shorthand is `go-yaml --plugin=merge-key=disable`.
Configured CLI builds also support `PLUGIN=merge-key=disable` and linked
third-party implementations through `PLUGIN-PACKAGE`.
