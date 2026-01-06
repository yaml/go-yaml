# go.yaml.in/yaml

[![Go Reference](https://pkg.go.dev/badge/go.yaml.in/yaml/v4.svg)](
https://pkg.go.dev/go.yaml.in/yaml/v4)
[![Go Report Card](https://goreportcard.com/badge/go.yaml.in/yaml/v4)](
https://goreportcard.com/report/go.yaml.in/yaml/v4)
[![CI](https://github.com/yaml/go-yaml/actions/workflows/go.yaml/badge.svg)](
https://github.com/yaml/go-yaml/actions/workflows/go.yaml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](
LICENSE)

**YAML support for Go — fast, compliant, and actively maintained**

This is the official YAML library for Go, maintained by the
[YAML organization](https://github.com/yaml).
We are the actively maintained fork of the original go-yaml project.
[Learn more about the project history →](#project-history)


## What's New in v4

### New Streaming API

v4 introduces a cleaner, more intuitive API with `Load`/`Dump` and
`Loader`/`Dumper`:

```go
// Simple loading
var config Config
err := yaml.Load(data, &config)

// Streaming from io.Reader
loader := yaml.NewLoader(file)
err := loader.Load(&config)
```

The old `Unmarshal`/`Marshal` API still works but is deprecated.

### Functional Options System

Configure YAML processing with composable options:

```go
// Use version presets
yaml.Dump(&data, yaml.V4)

// Customize formatting
yaml.Dump(&data, yaml.WithIndent(4), yaml.WithCompactSeqIndent(false))

// Combine presets with custom options
yaml.Dump(&data, yaml.V3, yaml.WithIndent(2))
```

Available presets: `yaml.V2`, `yaml.V3`, `yaml.V4`

### Better Error Messages

Structured error types with precise location information:

```go
var config Config
if err := yaml.Load(data, &config); err != nil {
    if typeErr, ok := err.(*yaml.TypeError); ok {
        for _, e := range typeErr.Errors {
            fmt.Printf("Error at line %d, column %d: %s\n",
                e.Line, e.Column, e.Problem)
        }
    }
}
```

### Plugin System (Coming Soon)

v4 will support plugins that hook into the YAML processing pipeline,
allowing custom behavior at each stage (scanning, parsing, composition,
construction, etc.).
This is a major extensibility feature coming to v4.


## Installation

```bash
go get go.yaml.in/yaml/v4
```


## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "go.yaml.in/yaml/v4"
)

type Config struct {
    Name    string
    Version string
    Server  struct {
        Host string
        Port int
    }
}

func main() {
    data := []byte(`
name: MyApp
version: 1.0.0
server:
  host: localhost
  port: 8080
`)

    var config Config
    if err := yaml.Load(data, &config); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Name: %s\n", config.Name)
    fmt.Printf("Server: %s:%d\n", config.Server.Host, config.Server.Port)

    // Dump back to YAML
    output, err := yaml.Dump(&config, yaml.V4)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\n%s", output)
}
```


## Examples

The [`example/`](example/) directory contains 17 working examples
demonstrating different features:

| Example | Description |
|---------|-------------|
| [basic_loader](example/basic_loader) | Simple YAML loading into structs |
| [basic_dumper](example/basic_dumper) | Simple YAML dumping from structs |
| [multi_document_loader](example/multi_document_loader) | Loading multiple YAML documents |
| [multi_document_dumper](example/multi_document_dumper) | Dumping multiple YAML documents |
| [single_document_loader](example/single_document_loader) | Loading a single YAML document |
| [loader_dumper_demo](example/loader_dumper_demo) | Combining loading and dumping operations |
| [version_options](example/version_options) | Using YAML version presets |
| [with_v4_option](example/with_v4_option) | Using the V4 version preset |
| [with_v4_override](example/with_v4_override) | Overriding V4 preset options |
| [multiple_options_loader](example/multiple_options_loader) | Using multiple options together |
| [dumper_with_indent](example/dumper_with_indent) | Custom indentation settings |
| [dumper_indent_comparison](example/dumper_indent_comparison) | Comparing different indentation options |
| [load_into_node](example/load_into_node) | Loading YAML into Node structures |
| [node_dump_with_options](example/node_dump_with_options) | Dumping Nodes with custom options |
| [node_load_decode_comparison](example/node_load_decode_comparison) | Comparing loading vs decoding with Nodes |
| [node_load_strict_unmarshaler](example/node_load_strict_unmarshaler) | Strict unmarshaling with Nodes |
| [node_programmatic_build](example/node_programmatic_build) | Building YAML Nodes programmatically |


## Documentation

For detailed guides, see the [`docs/`](docs/) directory:

- [v3 to v4 Migration Guide](docs/v3-to-v4-migration.md) — Complete upgrade guide
- [Dump and Load API Guide](docs/dump-load-api.md) — Comprehensive API walkthrough
- [Options Reference](docs/options.md) — All configuration options explained

Full API documentation is available at
[pkg.go.dev/go.yaml.in/yaml/v4](
https://pkg.go.dev/go.yaml.in/yaml/v4).


## Migrating from v3

If you're upgrading from v3, the main changes are:

- **Import path**: `gopkg.in/yaml.v3` → `go.yaml.in/yaml/v4`
- **API naming**: `Unmarshal`/`Marshal` → `Load`/`Dump`
  (old API still works but deprecated)
- **Options**: New functional options system with version presets
- **Errors**: `TypeError.Errors` is now `[]*UnmarshalError` with
  line/column info (was `[]string`)
- **Formatting**: Default is 2-space indent with compact sequences
  (use `yaml.V3` preset for v3 behavior)

All v3 APIs remain functional but are deprecated for removal in v5.

See [MIGRATION.md](docs/v3-to-v4-migration.md) for a complete migration
guide.


## YAML Compatibility

This library supports YAML 1.2 with some YAML 1.1 compatibility for
common cases:

- **YAML 1.1 booleans** (`yes`/`no`, `on`/`off`) are supported when
  decoding into typed bool values, otherwise treated as strings
- **Octals** support both YAML 1.1 format (`0777`) and YAML 1.2 format
  (`0o777`)
- **Base-60 floats** are not supported (removed in YAML 1.2)


## Development

This project uses GNU Make for deterministic builds and testing:

```bash
make test          # Run all tests
make lint          # Run linter
make fmt           # Format code
make shell         # Open shell with project environment
```

The makefile auto-installs Go and dependencies to `.cache/` for
reproducibility.
You don't need Go installed locally.

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development
instructions, coding conventions, and how to contribute.


## The `go-yaml` CLI Tool

This repository includes a CLI tool for debugging and understanding YAML
processing:

```bash
# Build the tool
make go-yaml

# Use it to inspect YAML processing stages
./go-yaml --help
./go-yaml -t <<< 'foo: bar'    # Show tokens
./go-yaml -e <<< 'foo: bar'    # Show events
./go-yaml -n <<< 'foo: bar'    # Show node tree
```

Install globally:

```bash
go install go.yaml.in/yaml/v4/cmd/go-yaml@latest
```

The `go-yaml` tool is invaluable for debugging issues and understanding
how YAML is parsed.


## Project History

This project started as the widely-used
[go-yaml/yaml](https://github.com/go-yaml/yaml) library,
originally developed within [Canonical](https://www.canonical.com) as part
of the [juju](https://juju.ubuntu.com) project.

In April 2025, the original author
[@niemeyer](https://github.com/niemeyer)
[labeled the project as "Archived"](
https://github.com/go-yaml/yaml/blob/944c86a7d2/README.md).
The official [YAML organization](https://github.com/yaml/) took over
ongoing maintenance and development.

We have assembled a dedicated team of maintainers including
representatives from go-yaml's most important downstream projects.
Our goal is to provide a stable, actively maintained YAML library for the
Go ecosystem.

### Version Strategy

- **v4**: Active development — all new features and bug fixes go here
- **v1, v2, v3**: Frozen legacy versions receiving security fixes only

If you're starting a new project or can upgrade, please use
`go.yaml.in/yaml/v4`.

### Get Involved

We welcome contributions!
Join us:

- [GitHub Issues](https://github.com/yaml/go-yaml/issues) — Report bugs,
  request features
- [GitHub Discussions](https://github.com/yaml/go-yaml/discussions) —
  Ask questions, share ideas
- [Slack](https://cloud-native.slack.com/archives/C08PPAT8PS7) —
  Real-time chat with maintainers

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.


## License

The yaml package is licensed under the MIT and Apache License 2.0
licenses.
Please see the [LICENSE](LICENSE) file for details.


<!--
## Resources

Helpful resources for learning and using go-yaml:

- Blog posts
- Tutorials
- Conference talks
- Third-party tools

(Coming soon)
-->
