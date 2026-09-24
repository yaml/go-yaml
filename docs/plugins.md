# Plugin system

The go-yaml v4 plugin system lets applications replace selected processing
stages without adding optional dependencies to the core module.

A plugin API defines a role such as `yaml-parser`, `json-comments`, or
`loader-limits`.
Each API can have several named implementations.
The current implementations are:

| API | Implementation | Package | Default |
|---|---|---|---|
| `yaml-parser` | `go-yaml` | core | yes |
| `yaml-parser` | `reference` | `plugin/yaml-parser/reference` | no |
| `json-comments` | `sanitizer` | `plugin/json-comments` | yes |
| `loader-limits` | `loader-limits` | `plugin/loader-limits` | yes |

The optional packages are separate Go modules.
Importing the core `go.yaml.in/yaml/v4` module does not acquire Glojure or the
YAMLStar generated code.

## Direct Go use

Pass an implementation to `yaml.WithPlugin` when code constructs the plugin
directly.

```go
import (
    "go.yaml.in/yaml/v4"
    jsoncomments "go.yaml.in/yaml/v4/plugin/json-comments"
    loaderlimits "go.yaml.in/yaml/v4/plugin/loader-limits"
)

var value any
err := yaml.Load(input, &value,
    yaml.WithPlugin(jsoncomments.New()),
    yaml.WithPlugin(loaderlimits.New(loaderlimits.DepthValue(50))))
```

The public plugin interfaces are:

```go
type YAMLParserPlugin interface {
    Parse(input []byte) ([]PluginEvent, error)
}

type JSONCommentsPlugin interface {
    Sanitize(input []byte) ([]byte, error)
}

type LoaderLimitsPlugin interface {
    CheckDepth(depth int, ctx *DepthContext) error
    CheckAlias(aliasCount, constructCount int) error
}
```

A JSON-comments plugin transforms the input first.
The selected parser then parses the transformed UTF-8 text.
The built-in `go-yaml` parser remains the default unless a YAML-parser plugin is
selected.

`PluginEvent` represents stream and document boundaries, mappings, sequences,
scalars, and aliases with ordinary Go fields.
The loader validates a YAML-parser plugin's complete event stream before
composing nodes.
Depth limits apply to that stream and cannot limit work already performed by
an external parser.

Input is buffered when a YAML-parser or JSON-comments plugin is active.
Plugin implementations must support independent concurrent calls.

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

## Named configuration

External packages register named implementations for `yaml.OptsYAML` and
`yaml.WithNamedPlugin`.
Call each optional package's `Register` function once during startup.

```go
if err := jsoncomments.Register(); err != nil {
    log.Fatal(err)
}
if err := reference.Register(); err != nil {
    log.Fatal(err)
}
```

Configuration can use mappings, strings, or booleans:

```yaml
plugin:
  yaml-parser: reference@v0.2.5
  json-comments: sanitizer@v0.1.9
  loader-limits:
    depth: 50
    alias: 1000
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

## Optional implementations

### Reference YAML-parser implementation

`go.yaml.in/yaml/v4/plugin/yaml-parser/reference` adapts the generated Go parser
from `github.com/yamlstar/yamlstar-plugin-parser-reference`.
The canonical parser source remains in `yaml/yaml-reference-parser-clj`.
It requires Go 1.24 or newer and does not need Clojure, Gloat, CGO, or a shared
library at runtime.

```go
import reference "go.yaml.in/yaml/v4/plugin/yaml-parser/reference"

var value any
err := yaml.Load(input, &value, yaml.WithPlugin(reference.New()))
```

### JSON comments

`go.yaml.in/yaml/v4/plugin/json-comments` adapts the sanitizer from
`github.com/yamlstar/yamlstar-plugin-json-comments`.
It accepts UTF-8 YAML containing `//` and non-nesting `/* */` comments.
Comment markers inside quoted scalars, block scalars, and URLs are preserved.
JSON literals and numbers permit adjacent comments such as `true// comment`.
Other plain scalars require separation, so `foo// text` remains scalar text.

The sanitizer replaces comment characters while preserving line endings.
The selected parser receives the sanitized text.
Comments do not appear in nodes, while YAML styles, tags, anchors, aliases,
and parser position information remain available.
See the upstream syntax document for the exact recognition rules.

## CLI build selection

The ordinary `go-yaml` binary contains only core implementations.
Optional implementations must be linked when the CLI is built.

`CONFIG` is a YAML options file.
It selects build dependencies and embeds the full configuration as the binary's
defaults.

```bash
make cli CONFIG=example/json-comments/options.yaml
```

`PLUGIN` is only the plugin selector DSL.
It never names a file and never contains YAML.

```bash
make cli PLUGIN=yaml-parser=reference@v0.2.5,json-comments
```

Selectors have these forms:

```text
API
API@VERSION
API=IMPLEMENTATION
API=IMPLEMENTATION@VERSION
```

A bare API selects its default implementation.
The build resolves the newest v-prefixed release when no version is supplied.
A version selects that exact release.
Go downloads released modules through its configured module proxy and stores
them in the Go module cache.
Build staging and generated workspaces live under `.cache/`.

For local development, place the related checkouts under `repos/` and set the
matching override:

```bash
make cli PLUGIN=yaml-parser=reference@v0.2.5,json-comments \
  REFERENCE-PARSER-LOCAL=1 JSON-COMMENTS-LOCAL=1
```

The compiled command uses the same selector DSL at runtime:

```bash
./go-yaml --plugin=yaml-parser=reference@0.2.5,json-comments -j data.yaml
```

A runtime selector can choose only implementations already linked into the
binary.
It cannot download or add Go code.
`-C FILE` and `--config=FILE` continue to load YAML configuration files.
A runtime configuration replaces embedded defaults.

JSON, YAML, node, and event modes support YAML-parser and JSON-comments plugins.
Token and legacy modes support the JSON-comments sanitizer with the built-in
`go-yaml` parser.
They reject an external parser because those modes do not consume YAML-parser
plugin events.

Run the optional module checks with:

```bash
make test-json-comments
make test-json-comments-race
make test-reference-parser
```

[upstream syntax document]: https://github.com/yamlstar/yamlstar-plugin-json-comments/blob/main/Syntax.md
