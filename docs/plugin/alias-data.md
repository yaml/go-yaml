# Alias-Data

Alias-Data controls anchor storage and alias lookup while loading.
The default policy is active and behaves like ordinary go-yaml v4:
anchors belong to one document and unknown aliases are errors.
Dumping behavior is unchanged.

| Configuration under `plugin.alias-data` | Behavior |
|---|---|
| Omitted | Ordinary document-scoped aliases |
| `true`, `{}`, or `alias-data` | Retain anchors across documents |
| `{stream: false}` | Ordinary document-scoped aliases |
| `{stream: true}` | Retain anchors across documents |
| `{path: true}` | Dotted paths and stream retention |
| `{path: true, stream: false}` | Dotted paths within each document |
| `false`, `disable`, or `{disable: true}` | Reject alias references |

Disabling aliases permits unused `&anchor` declarations and quoted `'*text'`.
It rejects local, external, cross-document, and merge-value alias references.
Saved settings in a disabled mapping are ignored without reading their sources.
Other plugin APIs keep their existing disabling semantics.

## Configuration

```yaml
plugin:
  alias-data:
    data:
      defaults: {color: blue, size: 3}
    file: aliases.yaml
    env: APP_*
    stream: false
```

`data` maps anchor names to arbitrary values.
`file` names exactly one mapping document, parsed with the selected parser and
ordinary aliases, without external alias sources.
Relative paths use the working directory.

`env` accepts `true` for the full environment, `false` to select nothing, a
string selector, or a sequence of exact names.
Only `*` has wildcard meaning in a string selector.
Environment values stay strings, including strings such as `001` and `false`.
Missing exact names are errors; an unmatched wildcard selects nothing.

Names supplied by data, file, and environment sources must match
`[A-Za-z0-9_-]+`.
Ordinary YAML anchor names retain go-yaml's existing rules.

Lookup order is current-document anchors, inline data, file data, environment
data, then anchors retained from previous documents.
A local definition takes precedence only after its position in the input.
Null and false are valid found values.
Mapping values work with merge keys.

An explicit empty selection enables stream retention.
A nonempty configuration defaults to document scope unless `stream: true`
or `path: true` is present.
An explicit `stream: false` overrides the path default.
`disable: false` is a host control field, so it does not make an otherwise
empty configuration nonempty.

Sources are read once when a new stream begins loading.
Configuration parsing, CLI builds, and help do not read sources.
Reusing a plugin or options creates a fresh snapshot and independent anchors
for each loader.
Callers must not mutate supplied Go values during snapshot creation.

## Dotted anchor paths

```yaml
# Options file
plugin:
  alias-data:
    path: true
```

```yaml
# Input
settings: &x1 {foo: {bar: 42}}
items: &x2 [zero, one, two, {baz: found}]
answer: *x1.foo.bar
selected: *x2.3.baz
numbers: &foo {'42': {bar: found}}
numeric: *foo.'42'.bar
```

The results include `answer: 42` and `selected: found`.
Split dotted aliases on `.` and validate every component against these
patterns, using ASCII word characters:

```text
^[\w\-\+]+$
^[0-9]+$
^'[0-9]+'$
```

The first component is the verbatim anchor name.
For later components, bare digits are sequence indices, single-quoted digits
are string mapping keys, and other matches are ordinary string mapping keys.
Check digit-only components before the overlapping ordinary-name pattern.
Sequence indices must match `0` or `[1-9][0-9]*`, without leading zeros.
Reject `00`, `01`, and `042` instead of treating them as mapping keys.
Numeric string keys require quotes: `*foo.'42'.bar` selects key `"42"`.
`'042'` selects the distinct string key `"042"`, without numeric conversion.
Actual integer mapping keys remain unsupported.
Bare indices cannot select mapping keys, and quoted digits cannot index
sequences.
Quoted words, signs, fractions, exponents, and double-quoted components are
invalid; only non-negative decimal digits may appear inside single quotes.
Unquoted `-42` and `+42` are ordinary string mapping keys.
Intermediate aliases are followed, and merged mappings preserve normal key
precedence, including the first mapping's priority in a merge sequence.
Terminal null and false values count as found.

Empty or invalid segments, missing keys, scalar traversal, invalid indices,
and cycles are errors identifying the full alias and failing segment.
The base anchor is selected by the normal source precedence before traversal.
A missing path never falls through to a lower-priority source.
Dots always separate segments when enabled, including when an exact dotted
anchor name exists.
The default `path: false` preserves literal dotted anchor names.
Aliases without dots bypass path validation even when path mode is enabled.
Document and stream root paths are deferred.

`path: true` enables stream retention unless `stream: false` is explicit.
The built-in and reference parsers accept dotted aliases.

## Go API

The implementation is included in the core module and named registry.
No registration or optional module is required.

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/aliasdata"
)

plugin, err := aliasdata.New(
    aliasdata.WithData(map[string]any{
        "defaults": map[string]any{"color": "blue", "size": 3},
    }),
    aliasdata.WithStream(false),
)
if err != nil {
    return err
}
var value map[string]any
err = yaml.Load([]byte("<<: *defaults\nsize: 5\n"), &value,
    yaml.WithPlugin(plugin))
```

Other options are `WithFile`, `WithEnv`, `WithPath`, and `WithDisable`.
Use `aliasdata.WithPath(true)` for dotted paths.
`New()` explicitly selects stream mode.
`WithPath(true)` enables stream retention by default, including plain aliases.
`WithStream(false)` explicitly selects document scope, regardless of option
order.
With only source options, use `WithStream(true)` to enable stream retention.
`yaml.WithNamedPlugin("alias-data")` does the same.
`NewFromYAML` accepts the implementation settings directly.
Multiple explicit Alias-Data plugins are an error.

Values are converted to node graphs and decoded through normal go-yaml
construction, including structs, custom unmarshaling, and merge keys.
The policy does not inject Go objects directly or preserve their identity.
Supplied node graphs are copied for each stream.
Normal alias expansion limits and recursive-alias errors remain active.

## Custom policies

Implement `yaml.AliasDataPlugin` to create a fresh `yaml.AliasDataContext`
for each stream, and pass it to `yaml.WithPlugin`.
Named implementations use `yaml.RegisterPlugin` with API `alias-data`.
The context owns all anchor storage and lookup, including local definitions.
Custom policies receive the complete alias name, including dots.
Path traversal belongs to the built-in policy and does not change the callback
contract.

The host supplies `yaml.AliasDataServices` with `NewNode`, `LoadFile`, and
`Environment` functions for operation-local conversion and snapshots.
`NewNode` returns a resolved data node and copies supplied node graphs.

Each document calls `BeginDocument`, then `DefineAnchor` and `ResolveAlias`
in input order, followed by `EndDocument` after successful composition.
Collection anchors are defined before their children are composed.
Policies may retain the live node pointer but must not mutate it.
They return data nodes, a found boolean, and an optional error.
External targets must have resolved tags; locally defined targets are resolved
by the host after composition.
A null value is a null scalar node, not a nil pointer.

Contexts must not retain open resources; there is no teardown callback.
Factories must allow concurrent context creation.
Callback errors preserve their cause through go-yaml load errors.

## CLI

```bash
go-yaml --plugin=alias-data -j stream.yaml
go-yaml --plugin=alias-data=disable -j input.yaml
go-yaml --config=options.yaml -y input.yaml
make cli PLUGIN=alias-data
make cli CONFIG=options.yaml
```

Normal JSON, YAML, and composed-node loading support Alias-Data.
Explicit policies are rejected in token/event output, preserving output,
legacy loading modes, and precomposed node input.
Those paths either bypass composition or preserve alias spellings instead of
materializing external values.
Default invocations keep the ordinary behavior of each mode.

Alias-Data is an in-process policy API with no shared-library installation.
