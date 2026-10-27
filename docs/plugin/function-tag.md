# Function-Tag

`function-tag` is a load-only constructor API.
Functions may return native Go values, including structs and pointers.

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/functiontag"
)

plugin, err := functiontag.New(functiontag.Functions{
    "add": func(x, y int) int { return x + y },
})
if err != nil {
    return err
}
var result int
err = yaml.Load([]byte("!add* [1, 2]"), &result, yaml.WithPlugin(plugin))
```

`!foo value` passes one constructed argument.
`!foo* [a, b]` passes separate arguments, and `!foo* []` passes none.
Typed parameters, variadic functions, and uniform callbacks are supported.
Return signatures may produce one value, `(value, error)`, no results, or only
`error`.
No-result constructors produce `yaml.Omit` after successful execution.

Use `functiontag.RegisterLibrary(name, exports)` to register a map of functions
or an object with exported methods.
`functiontag.New` takes explicit functions and an optional map of library aliases.
Libraries can be registered names, export maps, or objects.
Configuration resolves function references against registered libraries:

```yaml
plugin:
  function-tag:
    libraries:
      math: application-math
    functions:
      add: math/Add
```

Explicit bindings win over library lookup.
Unqualified names found in multiple libraries are errors.
Names, including dots, are opaque and case-sensitive.
Only linked, registered libraries are available to a CLI build.
The existing `PLUGIN-PACKAGE` build option can link a registration package.

Construction controls are distinct from a real null result:

- `yaml.Omit` suppresses an entry, item, or document.
- `yaml.Splice(items)` inserts items into a sequence.
- `yaml.Splice(mapping)` or `yaml.Splice([]yaml.Entry{...})` inserts complete
  mapping entries.
- `yaml.SpliceValue`, returned by a key, constructs its associated mapping and
  promotes its entries into the containing mapping.

Keys run before values.
An omitted key skips its value completely.
A key returning a complete entry splice also skips its original value.
An omitted value drops its entry; a mapping splice returned by a value replaces
that entry.
The normal duplicate-key and typed-destination checks apply after expansion.
Splices at document roots are errors.

An omitted root returns `yaml.NoDocument`.
Recognize it using `errors.Is(err, yaml.NoDocument)`.
It leaves the destination untouched and is distinct from `io.EOF`.
Streaming loaders remain usable afterward, while `WithAllDocuments` and CLIs
skip omitted documents automatically.
Anchored constructor results, including pointers and controls, are reused by
aliases without repeating side effects.
Effects already executed are not rolled back after a later error.

Third-party implementations can implement `yaml.FunctionTagPlugin` directly.
Its `yaml.FunctionTagInput` supplies the tag, role, structural path, base-typed
payload node, both original resolved mapping-entry nodes, and an
operation-local decoding function.
Reading `PairKey` or `PairValue` does not construct the associated value.
Treat its node and path as read-only.
The interface, controls, and helpers also work with `Node.Load`.

Generate package exports with:

```bash
go run ./cmd/functiontag-gen -import math -package main \
  -name MathFunctions -symbols Max,Sqrt -o math_functions.go
```

The generated ordinary function map works with go-yaml and YAMLStar.
The generator uses the Go toolchain and has no third-party dependencies.
It honors the current build environment and deterministically sorts exports.
Generic functions and unsupported return signatures are skipped with diagnostics;
explicitly requesting one fails.

Function-Tag does not affect dumping or extend the shared-library plugin ABI.

## Constructor context and alias imports

A typed constructor can opt into operation services with a leading
`yaml.FunctionTagContext` argument.
That argument does not consume YAML input, including in spread calls.
Ordinary function signatures keep their existing behavior.
The context supplies the same read-only node, pair, role, and path information
as `FunctionTagInput`, plus these services:

- `ConstructValue()` constructs the paired mapping value lazily and caches it.
- `DefineAlias(name, value)` defines data in the selected Alias-Data context.
- `LoadFile(path)` loads one isolated YAML document using the selected parser
  and rules, without function tags or external Alias-Data sources.

Services belong to the active operation and must not be retained after it.
The paired-value service is available for mapping entries.
Alias definition requires an explicit Alias-Data selection.
Direct `FunctionTagPlugin` implementations receive these services too.

Function-Tag has no built-in tags or knowledge of Alias-Data.
The `yaml` package registers the `alias-data` function library with
`load` bound to `aliasdata.Load`.
In Go, bind the function directly:

```go
aliases, err := aliasdata.New(aliasdata.WithPath())
functions, err := functiontag.New(functiontag.Functions{
    "load": aliasdata.Load,
})
```

The binding name determines the tag; use `"import"` to enable `!import`.
To combine explicit functions with a namespace, supply both arguments:

```go
functions, err := functiontag.New(
    functiontag.Functions{"load": aliasdata.Load},
    map[string]any{
        "math": functiontag.Functions{
            "add": func(a, b int) int { return a + b },
        },
    },
)
```

This enables both `!load vars: vars.yaml` and `!math/add* [1, 2]`.
For YAML configuration, the registered `alias-data` library exports `load`.
Select it explicitly to enable this input:

```yaml
!load vars: vars.yaml

foo: *vars.bar
map1:
  <<: *vars.hash2
  key: value
```

For example, `vars.yaml` can contain:

```yaml
bar: 42
hash2: {inherited: true, key: base}
```

Configure both plugins:

```yaml
plugin:
  alias-data:
    path: true
  function-tag:
    libraries:
      aliases: alias-data
    functions:
      load: aliases/load
```

For a complete CLI example with input, options, and imported data, see
[the runnable example](../../example/function-tag/README.md).

`load` requires a mapping key containing the alias name.
It constructs the paired filename once, loads its data, defines the alias,
and returns `Omit` so the directive contributes no output entry.
The filename must be a nonempty string.
For CLI file inputs, relative import paths resolve beside the input file.
Stdin input and library calls use the working directory.
Each directive reads its file when encountered.
The file must contain exactly one document; its root can be any YAML value,
including null, false, or a sequence.
Local anchors and merges in the imported file work normally.
Imports do not run function tags or nested import directives.

Definitions become available in source order and follow Alias-Data name,
precedence, and scope rules.
`path: true` defaults to stream scope; `stream: false` limits definitions to
one document.
Later definitions replace earlier definitions of the same name.
Failed imports do not define their alias.
Node inspection never executes constructors or imports, so it cannot resolve
an alias whose definition requires `!load` execution.
