# Plugin system

The go-yaml v4 plugin system lets applications replace selected processing
stages without adding optional dependencies to the core module.

A plugin API defines a role such as `yaml-parser`,
`loader-limits`, `merge-key`, `tab-indent`, or `error-format`.
Each API can have several named implementations.
The current implementations are:

| API | Implementation | Package | Default |
|---|---|---|---|
| `yaml-parser` | `go-yaml` | core | yes |
| `yaml-parser` | `reference` | `plugin/yamlparser/reference` | no |
| `error-format` | `error-format` | `plugin/errorformat/v3` and `plugin/errorformat/v4` | yes |
| `merge-key` | `merge-key` | `plugin/mergekey` | yes |
| `loader-limits` | `loader-limits` | `plugin/loaderlimits` | yes |
| `tab-indent` | `tab-indent` | `plugin/tabindent` | yes |

The reference YAML-parser package is a separate Go module.
Importing the core `go.yaml.in/yaml/v4` module does not acquire its generated
code or dependencies.

## Direct Go use

Pass an implementation to `yaml.WithPlugin` when code constructs the plugin
directly.

```go
import (
    "go.yaml.in/yaml/v4"
    "go.yaml.in/yaml/v4/plugin/loaderlimits"
    "go.yaml.in/yaml/v4/plugin/tabindent"
)

var value any
err := yaml.Load(input, &value,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthValue(50))),
    yaml.WithPlugin(tabindent.New()))
```

The public plugin interfaces are:

```go
type ErrorFormatPlugin interface {
    FormatLoadError(err *LoadError) string
    FormatDumpError(err *DumpError) string
}

type ParserPlugin interface {
    Parse(input []byte) ([]PluginEvent, error)
}

type LoaderLimitsPlugin interface {
    CheckDepth(depth int, ctx *DepthContext) error
    CheckAlias(aliasCount, constructCount int) error
}

type MergeKeyPlugin interface {
    MergeKeyConfig() MergeKeyConfig
    MergeMappings(MergeKeyInput) (*Node, error)
}

type TabIndentPlugin interface {
    TabIndentConfig() IndentConfig
}
```

The built-in `go-yaml` parser remains the default unless a YAML-parser plugin is
selected.

`PluginEvent` represents stream and document boundaries, mappings, sequences,
scalars, and aliases with ordinary Go fields.
The loader validates a YAML-parser plugin's complete event stream before
composing nodes.
Depth limits apply to that stream and cannot limit work already performed by
an external parser.

Input is buffered when a YAML-parser plugin is active.
Plugin implementations must support independent concurrent calls.

## Merge keys

The built-in merge algorithm remains enabled by default.
Use `merge-key` to disable merging, configure repeated directives and conflicts,
or select a replacement algorithm.
See [Merge-Key](plugin/merge-key.md) for configuration and implementation details.

## Error formatting

See [Error-format v3](plugin/error-format-v3.md) and
[Error-format v4](plugin/error-format-v4.md) for the formatter guides.

## Default Behavior

Bare `yaml.Load`, `yaml.Dump`, `yaml.NewLoader`, and `yaml.NewDumper` use
v4 defaults:

- default safety limits equivalent to `loaderlimits.New()`
- v4 error formatting equivalent to `errorformatv4.Must()`

Version presets select error formatting as follows:

| Preset | Error Formatter |
|---|---|
| `yaml.WithV2Defaults()` | v4 formatter |
| `yaml.WithV3Defaults()` | v3 formatter |
| `yaml.WithV4Defaults()` | v4 formatter |

Historical go-yaml v2 errors used the `yaml: ...` prefix, with `line N:` when
position information was available.
`WithV2Defaults()` selects v2-compatible processing and output options together
with the current v4 error formatter.
It does not reproduce historical v2 error strings.
For legacy-style error strings, apply the v3 formatter after the preset:

```go
opts := yaml.Options(
    yaml.WithV2Defaults(),
    yaml.WithPlugin(errorformatv3.New()),
)
```

The legacy APIs select their error formatters as follows:

| API | Error Formatter |
|---|---|
| `yaml.Unmarshal`, `yaml.NewDecoder` | v3 formatter |
| `yaml.Marshal` | v3 formatter |
| `yaml.NewEncoder` | v4 formatter |

`NewEncoder` keeps v3-compatible serialization options and explicitly selects
v4 error formatting.
Serialization options and error formatting are independent settings.

## YAML Configuration

Plugins can be configured from YAML strings using `yaml.OptsYAML`:

```yaml
plugin:
  loader-limits:
    depth: 50
    alias: 1000
  error-format:
    v4:
      position: long
```

```go
opts, err := yaml.OptsYAML(configYAML)
if err != nil {
    return err
}
err = yaml.Load(data, &out, opts)
```

See the plugin-specific guides for supported configuration keys.
An unversioned scalar first selects a registered implementation.
Otherwise, it can name an empty configuration entry accepted by the default
implementation.
For example, these configurations are equivalent:

```yaml
plugin:
  error-format: v3
```

```yaml
plugin:
  error-format:
    v3: {}
```

This shorthand works for any plugin whose default factory accepts that entry.
Registered implementation names take precedence.
Explicit `name` fields and versioned selectors always select implementations.
The same shorthand works with `--plugin=error-format=v3` at runtime and
`make cli PLUGIN=error-format=v3` at build time.

The default implementation is also named `error-format`:

```go
opts := yaml.WithNamedPlugin("error-format")
```

```yaml
plugin:
  error-format: error-format
```

## Loader-limits plugin

The loader-limits plugin controls maximum nesting depth and alias expansion.
The built-in defaults remain active unless another loader-limits implementation
is selected.

```go
loader := yaml.NewLoader(data, yaml.WithPlugin(loaderlimits.New()))
loader = yaml.NewLoader(data,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.AliasNone())))
loader = yaml.NewLoader(data,
    yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthValue(50))))
```

| Option | Effect |
|---|---|
| `DepthValue(n)` | Set the flow and block nesting limit |
| `DepthNone()` | Disable depth checking |
| `DepthFunc(fn)` | Supply a depth policy function |
| `AliasValue(n)` | Set the alias expansion limit |
| `AliasNone()` | Disable alias checking |
| `AliasFunc(fn)` | Supply an alias policy function |

## Tab-indent plugin

The tab-indent plugin enables tab-aware structural indentation while loading
and can emit structural indentation with tabs while dumping.
Its default `auto` mode detects tabs or spaces for each document and emits tabs.

```go
plugin := tabindent.New()
err := yaml.Load(input, &value, yaml.WithPlugin(plugin))
output, err := yaml.Dump(value, yaml.WithPlugin(plugin))
```

Configuration accepts `mode` (`auto` or `tabs`), `load` (`auto`, `spaces`, or
`tabs`), `dump` (`spaces` or `tabs`), and `auto` (`document` or `stream`).

## Named configuration

External packages register named implementations for `yaml.OptsYAML` and
`yaml.WithNamedPlugin`.
Call each optional package's `Register` function once during startup.

```go
if err := reference.Register(); err != nil {
    log.Fatal(err)
}
if err := tabindent.Register(); err != nil {
    log.Fatal(err)
}
```

Configuration can use mappings, strings, or booleans:

```yaml
plugin:
  yaml-parser: reference@v0.2.6
  loader-limits:
    depth: 50
    alias: 1000
  tab-indent:
    mode: auto
    auto: document
```

A string is the short form `IMPLEMENTATION` or `IMPLEMENTATION@VERSION`.
Versions with and without the leading `v` are accepted.
Documentation and Git tags use the `v` prefix.

`true` selects the API's default implementation with its defaults.
`false` leaves the plugin disabled.
A mapping can contain `name`, `version`, `disable`, and implementation
settings.
`disable: true` has the same effect as `false`, even when saved settings remain
in the mapping.
Null is invalid.

The host removes `name`, `version`, and `disable` before calling the
implementation factory.
A requested version must match the linked implementation exactly.

Disabling `loader-limits` leaves the core safety limits active.
The loader-limits implementation accepts integer `depth` and `alias` settings.
A null setting disables that one check.

### Release-candidate compatibility

Use `loader-limits` and `LoaderLimitsPlugin` in new code.
For migration from earlier v4 release candidates, the `limit` configuration
key and implementation name are accepted as deprecated aliases.
Configuring both `limit` and `loader-limits` is an error.
The deprecated `LimitPlugin` type alias also remains available.

The previously published `go.yaml.in/yaml/v4/plugin/limit` import path is
retained as a module deprecation marker.
Its original implementation remains available to users pinned to an earlier
release candidate.
Migrate imports to `go.yaml.in/yaml/v4/plugin/loaderlimits`.
The former `parser` name is not an alias for `yaml-parser`.

## Optional implementations

### Reference YAML-parser implementation

`go.yaml.in/yaml/v4/plugin/yamlparser/reference` adapts the generated Go parser
from `github.com/yamlstar/yamlstar-plugin-parser-reference`.
The canonical parser source remains in `yaml/yaml-reference-parser-clj`.
It requires Go 1.24 or newer and does not need Clojure, Gloat, CGO, or a shared
library at runtime.

```go
import reference "go.yaml.in/yaml/v4/plugin/yamlparser/reference"

var value any
err := yaml.Load(input, &value, yaml.WithPlugin(reference.New()))
```

## CLI build selection

The ordinary `go-yaml` binary contains only core implementations.
Optional implementations must be linked when the CLI is built.

`CONFIG` is a YAML options file.
It selects build dependencies and embeds the full configuration as the binary's
defaults.

```bash
make cli CONFIG=options.yaml
```

`PLUGIN` is only the plugin selector DSL.
It never names a file and never contains YAML.

```bash
make cli PLUGIN=yaml-parser=reference@v0.2.6,loader-limits
make cli PLUGIN=error-format
```

Selectors have these forms:

```text
API
API@VERSION
API=IMPLEMENTATION
API=IMPLEMENTATION@VERSION
```

A bare API selects its default implementation.
For optional implementations, the build resolves the newest v-prefixed
release when no version is supplied.
A version selects that exact release.
Go downloads released modules through its configured module proxy and stores
them in the Go module cache.
Build staging and generated workspaces live under `.cache/`.

For local development, place the related checkouts under `repos/` and set the
matching override:

```bash
make cli PLUGIN=yaml-parser=reference@v0.2.6,loader-limits \
  REFERENCE-PARSER-LOCAL=1
```

The compiled command uses the same selector DSL at runtime:

```bash
./go-yaml --plugin=yaml-parser=reference@0.2.6,loader-limits -j data.yaml
```

A runtime selector can choose only implementations already linked into the
binary.
It cannot download or add Go code.
`-C FILE` and `--config=FILE` continue to load YAML configuration files.
A runtime configuration replaces embedded defaults.

JSON, YAML, node, and event modes support YAML-parser plugins.
Token and legacy loading modes reject external parsers because those modes do
not consume YAML-parser plugin events.

Run the optional module checks with:

```bash
make test-limit
make test-reference-parser
```

## External plugin packages

`PLUGIN-PACKAGE` links released Go packages into a configured CLI without
adding adapters or dependencies to this repository.
Each package must export `Register() error` and register its implementations
through `yaml.RegisterPlugin`.
A parser implementation supplies the existing `yaml.ParserPlugin` interface.

```bash
make cli \
  PLUGIN-PACKAGE=example.org/custom-parser/goyaml@v1.2.3 \
  PLUGIN=yaml-parser=custom
./go-yaml --plugin=yaml-parser=custom -j data.yaml
```

Use a comma-separated list to link multiple packages:

```bash
make cli \
  PLUGIN-PACKAGE=example.org/parser/go@v1.2.3,example.org/limits@v0.2.0
```

Each entry requires an explicit version such as `v1.2.3` or `v1.2.3-rc.1`.
Local paths, unversioned entries, and `@latest` are not supported.
Go resolves dependencies using its configured module proxy and module cache.
The core module remains replaced by the current checkout in the staged CLI.

Packages are registered in list order before configuration is evaluated.
Registration failures, duplicate registrations, and invalid plugin options
fail the build before replacing an existing executable.
The completed executable validates the selected implementation and version.

`PLUGIN-PACKAGE` links code; `PLUGIN` and `CONFIG` select implementations and
embed defaults.
A build with only `PLUGIN-PACKAGE` keeps ordinary defaults and makes the
registered implementations available through runtime configuration or flags.
External packages may implement any existing plugin interface.
Only one parser implementation can be selected at a time.
Runtime flags cannot download or load additional packages.

## Plugin demos

Run `./cmd/go-yaml/demo/demo-plugins.sh` from a fresh clone to demonstrate the
plugins.
It builds the CLI once with the reference parser and external adapters fetched
through Go modules; no `repos/*` checkouts are needed.
The demo build selects the reference parser by default.
Run `make cli` afterward to restore the ordinary native CLI.

```bash
./cmd/go-yaml/demo/demo-plugins.sh --list
./cmd/go-yaml/demo/demo-plugins.sh 3 5-7 9
./cmd/go-yaml/demo/demo-plugins.sh -i 10-
```

See the [demo README](../cmd/go-yaml/demo/README.md) for interactive controls,
dependencies, and executable overrides.
