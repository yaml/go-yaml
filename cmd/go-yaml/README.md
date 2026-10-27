# go-yaml

The `go-yaml` binary is a YAML node inspection tool that provides various
modes for analyzing and transforming YAML data.

Below is a summary of its capabilities:

## License

See [LICENSE](../../LICENSE) and [NOTICE](../../NOTICE) for licensing details,
including the MIT license retained by the listed libyaml-derived files.

## Features

### YAML Parsing and Encoding

- `-y` / `--yaml`: Outputs YAML through ordinary value encoding.
- `-Y` / `--YAML`: Outputs YAML using nodes, retaining supported styles and
  comments without promising exact formatting preservation.

### JSON Conversion

- `-j` / `--json`: Outputs JSON in a compact format.
- `-J` / `--JSON`: Outputs JSON in a pretty-printed format.

### Token Inspection

- `-t` / `--token`: Outputs tokens from the YAML input.
- `-T` / `--TOKEN`: Outputs tokens with line information.

### Event Inspection

- `-e` / `--event`: Outputs events from the YAML input.
- `-E` / `--EVENT`: Outputs events with line information.

### Node Representation

- `-n` / `--node`: Outputs a compact representation of the YAML node structure.
- `-N` / `--NODE`: Outputs nodes with tags and styles.

Collection styles are `Block` or `Flow`.
`Plain` is a scalar style, alongside `Double`, `Single`, `Literal`, and `Folded`.
Detailed node input rejects `Plain` for mappings and sequences.

### Chaining

Token, event, and node output is also accepted as input. The supported forward
pipeline is:

```text
YAML text -> tokens -> events -> nodes -> YAML text
```

Input stages are detected from their sequence/map schema. Use
`-f` / `--from` with `token`, `event`, `node`, or `yaml` (or `t`, `e`, `n`,
or `y`) when input is ambiguous. Backward conversions fail explicitly because
later stages do not retain enough information to reconstruct earlier ones.

```bash
# Exercise the complete pipeline.
go-yaml -t file.yaml | go-yaml -e | go-yaml -N | go-yaml -Y

# Edit an event stream with yq before continuing.
go-yaml -e file.yaml |
  yq '(.[] | select(.event == "SCALAR" and .value == "old")).value = "new"' |
  go-yaml -Y

# Edit detailed nodes and then emit YAML.
go-yaml -N file.yaml |
  yq '(.. | select(.node? == "Scalar" and .value == "old")).value = "new"' |
  go-yaml -Y

# Treat contract-shaped data as ordinary YAML.
go-yaml -f yaml -n contract-shaped-data.yaml
```

The lowercase and uppercase forms select the same stage. Uppercase output adds
metadata such as source positions, tags, and styles; both forms are valid
inputs. Token and event streams include `STREAM-START` and `STREAM-END` so a
pipeline can validate that it received a complete stream.

### Formatting Options

- `-l` / `--long`: Enables long (block) formatted output.

### Processing Modes

The default processing path uses `Load` and `Dump`.
These long flags select the classic APIs for comparison:

- `--unmarshal`: Uses `Unmarshal` for YAML input.
- `--decode`: Uses `Decoder.Decode` for YAML input.
- `--marshal`: Uses `Marshal` for YAML output.
- `--encode`: Uses `Encoder.Encode` for YAML output.

### Help and Build Information

- `-h` / `--help`: Displays help information.
- `-o help`: Lists available option names and presets.

Use Go's build information to identify the installed module revision:

```bash
go version -m "$(command -v go-yaml)"
```

## Usage

Select an output mode and pass a file, or read from standard input.
Put flags before the file name:

```bash
go-yaml -J config.yaml
go-yaml -n < config.yaml
```

It validates flag combinations and provides error messages for incompatible
options.

## Plugin builds

The ordinary command contains the built-in `go-yaml` parser, loader-limits,
tab-indent, alias-data, and error-format plugins.
Alias-Data preserves ordinary document-scoped aliases by default.
An explicit `--plugin=alias-data` enables stream anchors.
Use `plugin: {alias-data: {stream: false}}` for document scope or
`plugin: {alias-data: disable}` to reject alias references.
Set `plugin: {alias-data: {path: true}}` in an options file for dotted paths
such as `*settings.server.port` and `*items.3.name`.
This also enables stream anchors unless `stream: false` is explicit.
Optional plugin implementations must be selected while building the command.

`PLUGIN` contains only the plugin selector DSL:

```bash
make cli PLUGIN=yaml-parser=reference@v0.2.6,loader-limits
./go-yaml --plugin=yaml-parser=reference@0.2.6,loader-limits \
  -j data.yaml
```

The selector forms are `API`, `API@VERSION`, `API=IMPLEMENTATION`, and
`API=IMPLEMENTATION@VERSION`.
Comma-separated selectors can be passed in one flag.
A bare API selects its default implementation.
The `yaml-parser` default is `go-yaml`, and the `tab-indent` default is
`tab-indent`.
The `error-format` default implementation is also named `error-format`:

```bash
make cli PLUGIN=error-format
./go-yaml --plugin=error-format -j data.yaml
```

`PLUGIN` and `--plugin` never name files and never contain YAML.
For optional implementations, the build resolves the newest v-prefixed plugin
release when a selector omits the version.
Core implementations such as `error-format` are unversioned within the
registry and ship with go-yaml.
Runtime selectors can choose only implementations already linked into the
binary.

## Build with embedded options

`CONFIG` names a YAML file.
The build links its selected optional implementations and embeds the complete
configuration as the command's defaults.

```yaml
indent: 4
plugin:
  yaml-parser: reference@v0.2.6
  loader-limits:
    depth: 50
    alias: 100
  error-format:
    v4:
      position: long
  tab-indent:
    mode: auto
```

```bash
make cli CONFIG=options.yaml
./go-yaml -j data.yaml
```

The binary does not need the options file at runtime.
Rebuild it to pick up file changes.
Missing files, invalid options, and unknown enabled implementations fail before
the existing binary is replaced.

`-C FILE` and `--config=FILE` load a YAML configuration at runtime.
A runtime configuration replaces embedded defaults.
`-o` flags then override individual non-plugin options.
An empty configuration mapping selects ordinary defaults while leaving compiled
implementations available to `--plugin`.

A plugin value may be a mapping, short string, or boolean.
`true` uses the API default, and `false` disables that API.
Mappings can use `disable: true` to retain settings without selecting the
plugin.
Null plugin values are invalid.
Versions may be entered with or without `v`, but documentation and Git tags use
the prefix.

External YAML-parser plugins work in JSON, YAML, node, and event modes.
Token and legacy modes reject them because those paths do not consume
YAML-parser plugin events.
Tab indentation works in YAML, JSON, node, event, and token modes.
Legacy loading and dumping modes reject it because they do not use the
plugin-aware pipeline.

For local development, related checkouts must be under `repos/`.
Set `REFERENCE-PARSER-LOCAL=1` to use the local reference-parser checkout.
Build staging, workspaces, and downloaded build metadata live under `.cache/`.
Go stores downloaded released modules in its module cache.

Use `make cli` without `CONFIG`, `PLUGIN`, or `PLUGIN-PACKAGE` to restore the
ordinary command.

## External plugin packages

`PLUGIN-PACKAGE` accepts a comma-separated list of released Go packages, each
with an explicit tagged version.
Each package must export `Register() error` and register its implementations
using `yaml.RegisterPlugin`.

```bash
make cli \
  PLUGIN-PACKAGE=example.org/custom-parser/goyaml@v1.2.3 \
  PLUGIN=yaml-parser=custom
```

Package versions may include release suffixes such as `v1.2.3-rc.1`.
Local paths, unversioned packages, and `@latest` are not supported.
A packages-only build keeps native defaults and permits later selection with
`--plugin` or `--config`.
Packages register in list order; configuration is validated after registration.
Compilation, registration, or configuration failures preserve an existing binary.

See [the plugin documentation](../../docs/plugin.md#external-plugin-packages)
for multiple-package builds and the external registration contract.

Run the [plugin demos](demo/README.md) to try the implementations from a fresh
clone, with optional interactive command editing.
