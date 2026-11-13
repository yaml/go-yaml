# go-yaml Examples

This directory contains examples demonstrating various features of the
go-yaml library.

## Running Examples

Each example can be run directly:

```bash
go run ./example/basic_loader
```

Or from within the example directory:

```bash
cd example/basic_loader
go run .
```

## Basic Examples

### Loading YAML

**`basic_loader/main.go`** - Simple YAML loading with Loader
- Creates a Loader from a reader
- Loads YAML into Go structs
- Basic unmarshaling example

**`load_into_node/main.go`** - Load YAML into Node representation
- Uses Node type for low-level YAML access
- Demonstrates working with the YAML AST

**`multi_document_loader/main.go`** - Load multiple YAML documents
- Processes multi-document streams
- Loops through documents until EOF

**`single_document_loader/main.go`** - Load only first document
- Uses WithSingleDocument option
- Subsequent Load() calls return EOF

### Dumping YAML

**`basic_dumper/main.go`** - Simple YAML dumping with Dumper
- Creates a Dumper to a writer
- Dumps Go structs to YAML

**`multi_document_dumper/main.go`** - Dump multiple YAML documents
- Creates a multi-document stream
- Multiple Dump() calls separated by `---`

**`dumper_with_indent/main.go`** - Custom indentation
- Uses WithIndent option
- Demonstrates formatting control

**`dumper_indent_comparison/main.go`** - Compare different indent levels
- Shows output with 2, 4, and 8-space indentation

## Options System Examples

**`with_v4_option.go`** - Using v4 option presets
- Demonstrates yaml.WithV4Defaults()
- Shows v4 defaults (2-space indent)
- Compares v3 presets with v4 defaults

**`with_v4_override.go`** - Overriding option presets
- Combines version presets with individual options
- Demonstrates left-to-right option application
- Shows how later options override earlier ones

**`multiple_options_loader/main.go`** - Combining multiple options
- Uses WithSingleDocument and WithKnownFields together
- Demonstrates strict field checking
- Shows error handling for unknown fields

## Node-Level API Examples

**`node_load_strict_unmarshaler/main.go`** - Strict field checking in custom
unmarshalers
- Applies strict checking explicitly inside custom UnmarshalYAML
- Uses node.Load() with WithKnownFields()
- Demonstrates proper error handling for unknown fields

**`node_dump_with_options/main.go`** - Encoding nodes with different options
- Builds nodes with stored quoting and flow styles
- Applies indentation when serializing the nodes
- Demonstrates explicit start markers at serialization time

**`node_load_decode_comparison/main.go`** - Compare Decode() vs Load()
- Side-by-side comparison of old and new approaches
- Shows that node.Decode() follows the KnownFields setting of the outer load
  (off here)
- Demonstrates passing WithKnownFields() to node.Load()

**`node_programmatic_build/main.go`** - Build YAML programmatically
- Build complex Node structures with options
- Shows round-trip (Dump then Load) with options
- Demonstrates node manipulation workflow

## Complete Demo

**`loader_dumper_demo/main.go`** - Comprehensive feature demonstration
- Covers Loader, Dumper, and options
- Multiple examples in one file
- Good overview of library capabilities

## Common Patterns

### Basic Load and Dump

```go
import "go.yaml.in/yaml/v4"

// Load
var config Config
yaml.Load(yamlData, &config)

// Dump
data, _ := yaml.Dump(&config)
```

### Streaming with Options

```go
// Load with options
loader, _ := yaml.NewLoader(reader,
    yaml.WithKnownFields(),
    yaml.WithSingleDocument(),
)
loader.Load(&config)

// Dump with options
dumper, _ := yaml.NewDumper(writer,
    yaml.WithIndent(2),
)
dumper.Dump(&config)
dumper.Close()
```

### Using Version Presets

```go
dumper, _ := yaml.NewDumper(writer,
    yaml.WithV4Defaults(),
)
```

## Learn More

- See the [main package documentation](https://pkg.go.dev/go.yaml.in/yaml/v4)
  for API reference
- Run `make doc-http` from the project root to view local documentation
- Check individual example source code for detailed comments
