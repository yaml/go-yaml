# YAML Dumping and Loading API

This guide covers go-yaml's dumping and loading APIs,
from simple one-liners to advanced streaming with full configuration control.

## Recommended API for v4+

**For new code in v4+, prefer `Dump`/`Load` over `Marshal`/`Unmarshal` and
`Encode`/`Decode`.**

The classic API (`Marshal`, `Unmarshal`, `Encoder`, `Decoder`) remains
supported for compatibility, but `Dump`/`Load` and `Dumper`/`Loader` offer
more flexibility with options support.

### Why Dump and Load?

The `Dump` and `Load` functions provide the same simplicity as
`Marshal`/`Unmarshal` while offering full control over YAML formatting and
parsing behavior through options:

```go
// Old way (v3 semantics, no options):
data, _ := yaml.Marshal(&config)

// New way (v4 semantics by default + full options):
data, _ := yaml.Dump(&config)  // v4 defaults: 2-space indent, compact
data, _ := yaml.Dump(&config, yaml.WithIndent(4))  // v4 with custom indent
```

The names `Dump` and `Load` are from the YAML specification and also the most
common names used for these actions in actual YAML implementations.

**Note**: Marshal, Unmarshal, Encode and Decode are not part of the standard
YAML vocabulary, but the are still fitting because go-yaml v2 and v3 did not
provide a full Dump and Load stack.
Those involve the Represent (for Dump) and Construct (for Load) YAML stack
stages and v4 will offer those (optionally) for Dump and Load.

### Classic API

`Marshal`/`Unmarshal` and `Encode`/`Decode` provide a simple, options-free
interface:

- They work **without options** and use **v3 semantics** (4-space
  indent, non-compact sequences)
- They're perfect for simple use cases and upgrading existing v3 code
- For new code, use `Dump`/`Load` instead—they can replicate the same behavior
  with added flexibility

### Flexibility

With `Dump` and `Load`, you can choose your starting point:

- **v4 semantics**: `yaml.Dump(&data)` (default: 2-space indent, compact
  sequences)
- **v3 semantics**: `yaml.Dump(&data, yaml.WithV3Defaults())` (same
  indentation and sequence style as Marshal: 4-space indent, non-compact)
- **v2 semantics**: `yaml.Dump(&data, yaml.WithV2Defaults())` (2-space indent, non-compact)
- **Custom options**: `yaml.Dump(&data, yaml.WithIndent(4),
  yaml.WithExplicitStart())`

This means `Dump` and `Load` default to modern v4 formatting, but can easily
replicate what `Marshal`/`Unmarshal` do, while also giving you complete control
when you need it.

## API Overview

go-yaml provides five main API functions for dumping and loading YAML:

| Reader | Writer | Configurable | Use Case |
|--------|--------|--------------|----------|
| `Load` | `Dump` | Yes | Single or multi-doc with options |
| `NewLoader` | `NewDumper` | Yes | Large files, continuous streams |
| `Unmarshal` | `Marshal` | No | Quick conversions, preset behavior |
| `NewDecoder` | `NewEncoder` | No | Multi-doc streams, preset behavior |

## Configurable API: Dump and Load

Like Marshal/Unmarshal but with options support.
Perfect for single operations that need custom behavior.

### Dump

Dump a single value with v4 defaults (2-space indent, compact):

```go
data, err := yaml.Dump(&config)
```

With custom options:

```go
data, err := yaml.Dump(&config, yaml.WithIndent(4))
```

Dump multiple values as a multi-document stream using `WithAllDocuments()`:

```go
docs := []Config{config1, config2, config3}
data, err := yaml.Dump(docs, yaml.WithAllDocuments(), yaml.WithIndent(2))
// Output:
// name: first
// ---
// name: second
// ---
// name: third
```

### Load

Load a single document with options:

```go
var config Config
err := yaml.Load(yamlData, &config, yaml.WithKnownFields())
```

**Important:** `Load` requires exactly one document by default.
Zero documents or multiple documents will return an error.

Strict validation catches typos:

```go
yamlData := []byte(`
name: myapp
prto: 8080  # typo!
`)

var config Config
err := yaml.Load(yamlData, &config, yaml.WithKnownFields())
// Error: field prto not found in type Config
```

Load all documents from a multi-document stream using `WithAllDocuments()`:

```go
multiDoc := []byte(`
name: first
---
name: second
---
name: third
`)

var docs []map[string]any
err := yaml.Load(multiDoc, &docs, yaml.WithAllDocuments())
for i, doc := range docs {
    fmt.Printf("Doc %d: %v\n", i, doc)
}
```

With typed slices:

```go
var configs []Config
err := yaml.Load(multiDoc, &configs, yaml.WithAllDocuments())
// Each document decoded as Config
```

**When to use:** Need options but don't need streaming.
Config files, API responses, test data.

## Configurable Streaming API: NewDumper and NewLoader

Full control over dumping/loading with streaming support and configurable
options.
Use for large files, network streams, or when you need to process documents one
at a time with custom configuration.

### NewDumper

Create a dumper that writes to any `io.Writer`:

```go
file, _ := os.Create("output.yaml")
defer file.Close()

dumper, err := yaml.NewDumper(file, yaml.WithIndent(2))
if err != nil {
    log.Fatal(err)
}

// Dump multiple documents
dumper.Dump(&doc1)
dumper.Dump(&doc2)
dumper.Dump(&doc3)

// Always close to flush remaining output
dumper.Close()
```

Write to a buffer:

```go
var buf bytes.Buffer
dumper, _ := yaml.NewDumper(&buf, yaml.WithIndent(2))
dumper.Dump(&config)
dumper.Close()
fmt.Println(buf.String())
```

**Note: `NewDumper` opens a write stream that must be closed with a call to
`Close()`.
This call can be deferred.**

### NewLoader

Create a loader that reads from any `io.Reader`:

```go
file, _ := os.Open("config.yaml")
defer file.Close()

loader, err := yaml.NewLoader(file, yaml.WithKnownFields())
if err != nil {
    log.Fatal(err)
}

var config Config
if err := loader.Load(&config); err != nil {
    log.Fatal(err)
}
```

Process multi-document streams:

```go
loader, _ := yaml.NewLoader(reader)

for {
    var doc map[string]any
    err := loader.Load(&doc)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    processDocument(doc)
}
```

**When to use:** Large files, network streams, processing documents
incrementally, or when you need maximum control.

## Classic API: Marshal and Unmarshal

Simple conversion with no options or streaming.

**Note:** These functions use **v3 semantics** (4-space indent, non-compact
sequences) and **do not accept options**.
For new code in v4+, use `Dump` and `Load` instead, which offer the
same simplicity with full options control.

### Marshal

```go
import "go.yaml.in/yaml/v4"

type Config struct {
    Name    string `yaml:"name"`
    Port    int    `yaml:"port"`
    Enabled bool   `yaml:"enabled"`
}

config := Config{Name: "myapp", Port: 8080, Enabled: true}
data, err := yaml.Marshal(&config)
// Output:
// name: myapp
// port: 8080
// enabled: true
```

**Equivalent using Dump with v3 semantics to match Marshal:**

```go
config := Config{Name: "myapp", Port: 8080, Enabled: true}
data, err := yaml.Dump(&config, yaml.WithV3Defaults())
// Same output as Marshal (4-space indent, non-compact)
```

### Unmarshal

```go
yamlData := []byte(`
name: myapp
port: 8080
enabled: true
`)

var config Config
err := yaml.Unmarshal(yamlData, &config)
fmt.Println(config.Name)  // "myapp"
```

**Equivalent using Load with v3 semantics to match Unmarshal:**

```go
var config Config
err := yaml.Load(yamlData, &config, yaml.WithV3Defaults())
fmt.Println(config.Name)  // "myapp"
```

**When to use:** Quick scripts, tests, simple config files where default
formatting is fine.

## Classic Streaming API: Encoder and Decoder

For multi-document streams without needing options.
Simple functions for encoding/decoding streams.

### Encode

Encode values to an `io.Writer`:

```go
file, _ := os.Create("output.yaml")
defer file.Close()

encoder := yaml.NewEncoder(file)
encoder.Encode(&doc1)
encoder.Encode(&doc2)
encoder.Encode(&doc3)
encoder.Close()
```

Each call to `Encode` writes a separate YAML document:

```yaml
name: first
---
name: second
---
name: third
```

**Equivalent using NewDumper with v3 semantics to match Encoder:**

```go
file, _ := os.Create("output.yaml")
defer file.Close()

dumper, _ := yaml.NewDumper(file, yaml.WithV3Defaults())
dumper.Dump(&doc1)
dumper.Dump(&doc2)
dumper.Dump(&doc3)
dumper.Close()
// Same output as NewEncoder (4-space indent, non-compact)
```

### Decode

Decode values from an `io.Reader`:

```go
file, _ := os.Open("config.yaml")
defer file.Close()

decoder := yaml.NewDecoder(file)

for {
    var doc map[string]any
    err := decoder.Decode(&doc)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    processDocument(doc)
}
```

**Equivalent using NewLoader with v3 semantics to match Decoder:**

```go
file, _ := os.Open("config.yaml")
defer file.Close()

loader, _ := yaml.NewLoader(file, yaml.WithV3Defaults())

for {
    var doc map[string]any
    err := loader.Load(&doc)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    processDocument(doc)
}
```

**When to use:** Multi-document streams where default formatting is fine.
No need for custom indentation or validation options.

## Options System

All flexible and streaming APIs accept options.
Options are applied left-to-right, with later options overriding earlier ones.

### Individual Options

```go
yaml.NewDumper(w,
    yaml.WithIndent(2),              // 2-space indentation
    yaml.WithCompactSeqIndent(),     // Compact list style (defaults to true)
    yaml.WithLineWidth(120),         // Wider lines
    yaml.WithUnicode(false),         // Escape Unicode (override default)
)

yaml.NewLoader(r,
    yaml.WithKnownFields(),          // Strict field checking (defaults to true)
    yaml.WithSingleDocument(),       // Only load first doc
    yaml.WithUniqueKeys(),           // Error on duplicate keys (defaults to true)
)
```

### Version Presets

Use preset options that match different go-yaml versions:

```go
// v2: 2-space indent, non-compact sequences
yaml.Dump(&config, yaml.WithV2Defaults())

// v3: 4-space indent, non-compact sequences
yaml.Dump(&config, yaml.WithV3Defaults())

// v4: 2-space indent, compact sequences (default)
yaml.Dump(&config, yaml.WithV4Defaults())  // or just yaml.Dump(&config)
```

### Combining Options

Mix presets with individual overrides:

```go
// Start with v3, then override indent to 2
yaml.NewDumper(w,
    yaml.WithV3Defaults(),
    yaml.WithIndent(2),  // This wins
)
```

### Loading Options from YAML

Configure formatting via YAML files:

```go
configYAML := `
indent: 3
compact-seq-indent: true
known-fields: true
`

opts, err := yaml.OptsYAML(configYAML)
if err != nil {
    log.Fatal(err)
}

data, err := yaml.Dump(&config, opts)
```

See [Options Guide](options.md) for complete option reference.

## Node-Level Load and Dump

When working with the Node API directly, you can also use options through the
`Load()` and `Dump()` methods.

### Node.Load() - Load with Options

The `Node.Load()` method is particularly useful inside custom `UnmarshalYAML`
implementations where you need to preserve options like `WithKnownFields()`.

```go
type Config struct {
    Name string `yaml:"name"`
    Port int    `yaml:"port"`
}

func (c *Config) UnmarshalYAML(node *yaml.Node) error {
    type plain Config
    // Use Load to preserve KnownFields option
    return node.Load((*plain)(c), yaml.WithKnownFields())
}
```

**Solves Issue #460:** Before `node.Load()`, calling `node.Decode()` in custom
unmarshalers would create a new decoder without options, losing strict field
validation.

**Usage example:**

```go
yamlData := []byte(`
name: myapp
port: 8080
unknown: field  # This will be caught!
`)

var config Config
err := yaml.Unmarshal(yamlData, &config)
if err != nil {
    fmt.Println("Error:", err)  // Error: field unknown not found in type Config
}
```

### Node.Dump() - Dump with Options

The `Node.Dump()` method lets you apply dumping options when building Node
trees programmatically:

```go
config := Config{Name: "myapp", Port: 8080}

var node yaml.Node
err := node.Dump(&config,
    yaml.WithIndent(2),
    yaml.WithExplicitStart(),
)

// Now marshal the node to YAML
data, _ := yaml.Marshal(&node)
fmt.Println(string(data))
// Output:
// ---
// name: myapp
// port: 8080
```

**When to use:**
- Custom unmarshalers that need strict validation (`node.Load()`)
- Building Node trees with specific formatting (`node.Dump()`)
- Programmatic YAML manipulation with options
- Preserving options through the full load/dump pipeline

## Examples

### Example 1: Config File with Validation

```go
func LoadConfig(filename string) (*Config, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    var config Config
    err = yaml.Load(data, &config,
        yaml.WithKnownFields(),       // Catch typos (defaults to true)
    )
    return &config, err
}
```

### Example 2: Generate CI/CD Config

```go
func GeneratePipeline(stages []Stage) ([]byte, error) {
    return yaml.Dump(&Pipeline{Stages: stages},
        yaml.WithIndent(2),  // CI systems prefer 2-space
    )
}
```

### Example 3: Process Log Stream

```go
func ProcessYAMLLogs(reader io.Reader) error {
    loader, _ := yaml.NewLoader(reader)

    for {
        var entry LogEntry
        err := loader.Load(&entry)
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }
        handleLogEntry(entry)
    }
}
```

### Example 4: Merge Multiple Configs

```go
func MergeConfigs(files []string) ([]byte, error) {
    var allDocs []any

    for _, f := range files {
        data, _ := os.ReadFile(f)
        var docs []any
        _ = yaml.Load(data, &docs, yaml.WithAllDocuments())
        allDocs = append(allDocs, docs...)
    }

    return yaml.Dump(allDocs, yaml.WithAllDocuments(), yaml.WithIndent(2))
}
```

## See Also

- [Options Guide](options.md) - Complete option reference
- [API Documentation](https://pkg.go.dev/go.yaml.in/yaml/v4) - Full API
  reference
