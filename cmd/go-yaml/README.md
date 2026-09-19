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

## JSON-comments plugin build

From the repository root, run
`make cli CONFIG=example/json-comments/options.yaml` to build `./go-yaml`
with the plugins and defaults named in the options file.
The example configuration enables JSON-comments.
The build resolves the newest tagged JSON-comments Go release each time.

```bash
make cli CONFIG=example/json-comments/options.yaml
./go-yaml -j example/json-comments/data.yaml
```

The plugin also works in node, YAML, and event output modes and can be selected
through `-C` YAML configuration using `plugin: {json-comments: {}}`.
Token output and legacy loading modes do not support event-source plugins.
Input is buffered, comments are discarded, and source positions are unknown.
Use `make cli` to restore the ordinary binary.

## Build with embedded options

`make cli CONFIG=FILE` compiles the plugins named in the file and embeds its
contents as the CLI's default configuration.
For example:

```yaml
indent: 4
plugin:
  limit:
    depth: 50
    alias: 100
  json-comments: {}
```

```bash
make cli CONFIG=example/json-comments/options.yaml
printf 'a: true // comment\n' | ./go-yaml -j
```

The binary no longer needs the options file at runtime.
Rebuild to pick up edits to the file.
Missing files, invalid options, and unknown enabled plugin names fail the build
without replacing the existing binary.
Set a plugin to `true` for its defaults, `false` to leave it unselected,
or a mapping for its settings.
Any plugin mapping can include `disable: true` to leave it unselected while
retaining its other settings.
`disable: false` enables it and passes the other settings to the plugin.
`json-comments: false` keeps its optional event source out of the build.
Null plugin values are invalid.
To select a specific JSON-comments release, use
`json-comments: {version: 0.1.8}` instead of `true`.
`version: v0.1.8` is also accepted.
The version selects code at build time and remains in embedded defaults.
The binary validates it against the linked release when loading options.
The same file works with `-C`; a different version request fails.

A runtime `-C other.yaml` replaces the entire embedded configuration.
`-o` flags override the selected configuration's options.
`--plugin=NAME` selects the default implementation with default settings.
`--plugin=API=NAME` selects an implementation explicitly.
Either form overrides configuration for the same API.
Using `-C` with an empty mapping (`{}`) selects ordinary defaults; compiled
plugins remain available through `--plugin`.

A configured build containing only core options or `limit` does not link
Glojure.
For development against a local plugin checkout under
`repos/yamlstar-plugin-json-comments`, set `JSON-COMMENTS-LOCAL=1`.
`make cli` without `CONFIG` restores the ordinary CLI with no embedded options.
