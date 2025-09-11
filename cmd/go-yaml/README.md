# go-yaml

The `go-yaml` binary is a YAML node inspection tool that provides various modes for analyzing and transforming YAML data.

Below is a summary of its capabilities:

## License

The `go-yaml` project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for more details.

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
- `-n` / `--node`: Outputs a detailed representation of the YAML node structure.

### Formatting Options
- `-l` / `--long`: Enables long (block) formatted output.

### Processing Modes
- `-u` / `--unmarshal`: Uses `Unmarshal` instead of `Decode` for YAML input.
- `-m` / `--marshal`: Uses `Marshal` instead of `Encode` for YAML output.

### Help and Version
- `-h` / `--help`: Displays help information.
- `--version`: Displays the version of the tool.

## Usage
The tool reads YAML data from `stdin` and processes it based on the specified flags. It validates flag combinations and provides error messages for incompatible options.