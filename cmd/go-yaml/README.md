# go-yaml

The `go-yaml` binary is a YAML node inspection tool that provides various
modes for analyzing and transforming YAML data.

Below is a summary of its capabilities:

## License

The `go-yaml` project is licensed under the Apache License 2.0.
See the [LICENSE](LICENSE) file for more details.

## Features

### YAML Parsing and Encoding
- `-y` / `--yaml`: Outputs YAML in a compact format.
- `-Y` / `--YAML`: Outputs YAML while preserving styles and comments.

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
- `-u` / `--unmarshal`: Uses `Unmarshal` instead of `Decode` for YAML input.
- `-m` / `--marshal`: Uses `Marshal` instead of `Encode` for YAML output.

### Help and Version
- `-h` / `--help`: Displays help information.
- `--version`: Displays the version of the tool.

## Usage
The tool reads YAML data from `stdin` and processes it based on the specified
flags.
It validates flag combinations and provides error messages for incompatible
options.

## Plugin builds

The ordinary command contains the built-in `go-yaml` parser and loader-limits
plugin.
Optional plugin implementations must be selected while building the command.

`PLUGIN` contains only the plugin selector DSL:

```bash
make cli PLUGIN=yaml-parser=reference@v0.2.5,json-comments
./go-yaml --plugin=yaml-parser=reference@0.2.5,json-comments \
  -j example/json-comments/data.yaml
```

The selector forms are `API`, `API@VERSION`, `API=IMPLEMENTATION`, and
`API=IMPLEMENTATION@VERSION`.
Comma-separated selectors can be passed in one flag.
A bare API selects its default implementation.
The `json-comments` default is `sanitizer`, and the `yaml-parser` default is
`go-yaml`.

`PLUGIN` and `--plugin` never name files and never contain YAML.
The build resolves the newest v-prefixed plugin release when a selector omits
the version.
Runtime selectors can choose only implementations already linked into the
binary.

## Build with embedded options

`CONFIG` names a YAML file.
The build links its selected optional implementations and embeds the complete
configuration as the command's defaults.

```yaml
indent: 4
plugin:
  yaml-parser: reference@v0.2.5
  loader-limits:
    depth: 50
    alias: 100
  json-comments: sanitizer@v0.1.9
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

The JSON-comments sanitizer works in JSON, YAML, node, event, token, and legacy
loading modes when the built-in parser is selected.
External YAML-parser plugins work in JSON, YAML, node, and event modes.
Token and legacy modes reject them because those paths do not consume
YAML-parser plugin events.

For local development, related checkouts must be under `repos/`.
Set `JSON-COMMENTS-LOCAL=1` or `REFERENCE-PARSER-LOCAL=1` to use them.
Build staging, workspaces, and downloaded build metadata live under `.cache/`.
Go stores downloaded released modules in its module cache.

Use `make cli` without `CONFIG` or `PLUGIN` to restore the ordinary command.
