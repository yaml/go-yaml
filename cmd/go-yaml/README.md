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
  yq '(.. | select(.kind? == "Scalar" and .value == "old")).value = "new"' |
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
