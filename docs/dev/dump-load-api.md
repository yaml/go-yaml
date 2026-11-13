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

The Load pipeline composes nodes, resolves tags, and constructs Go values.
The Dump pipeline represents Go values as nodes, removes inferable tags, and
serializes them.
Classic APIs use these pipelines with compatibility settings.

### Classic API

`Marshal`/`Unmarshal` and `Encode`/`Decode` provide a simple, options-free
interface:

- They work **without options** and use **v3 semantics** (4-space
  indent, non-compact sequences)
- They're perfect for simple use cases and upgrading existing v3 code
- For new code, use `Dump`/`Load` instead - they can replicate the same behavior
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
select compatibility defaults while giving you control when you need it.
Presets do not reproduce every legacy behavior.
`Marshal` disables wrapping; use `WithLineWidth(-1)` after `WithV3Defaults()`
when matching it.
`Load` validates trailing documents, unlike `Unmarshal`.

## API Overview

go-yaml provides these entry points for dumping and loading YAML:

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
file, err := os.Create("output.yaml")
if err != nil {
    log.Fatal(err)
}
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
file, err := os.Open("config.yaml")
if err != nil {
    log.Fatal(err)
}
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
loader, err := yaml.NewLoader(reader)
if err != nil {
    log.Fatal(err)
}

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

**Using Dump with classic Marshal formatting settings:**

```go
config := Config{Name: "myapp", Port: 8080, Enabled: true}
data, err := yaml.Dump(&config,
    yaml.WithV3Defaults(), yaml.WithLineWidth(-1))
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

**Using Load with v3 defaults for a single document:**

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
file, err := os.Create("output.yaml")
if err != nil {
    log.Fatal(err)
}
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

**Using NewDumper with classic Encoder output settings:**

```go
file, err := os.Create("output.yaml")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

dumper, _ := yaml.NewDumper(file,
    yaml.WithV3Defaults(), yaml.WithPlugin(errorformatv4.Must()))
dumper.Dump(&doc1)
dumper.Dump(&doc2)
dumper.Dump(&doc3)
dumper.Close()
// Import errorformatv4 from go.yaml.in/yaml/v4/plugin/errorformat/v4.
// NewEncoder uses v3 output settings and v4 error formatting.
```

### Decode

Decode values from an `io.Reader`:

```go
file, err := os.Open("config.yaml")
if err != nil {
    log.Fatal(err)
}
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
file, err := os.Open("config.yaml")
if err != nil {
    log.Fatal(err)
}
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
    yaml.WithKnownFields(),          // Reject unknown fields (off by default)
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

See [Options Guide](../options.md) for complete option reference.

## Node-Level Load and Dump

When working with the Node API directly, you can also use options through the
`Load()` and `Dump()` methods.

### Node.Load() - Load with Options

The `Node.Load()` method is useful inside custom `UnmarshalYAML`
implementations that need to set options independent of how they are being
loaded.
Each call starts with fresh defaults, independently of the outer loader.
Pass construction options explicitly; parser options cannot affect a graph
that has already been composed.

```go
type Config struct {
    Name string `yaml:"name"`
    Port int    `yaml:"port"`
}

func (c *Config) UnmarshalYAML(node *yaml.Node) error {
    type plain Config
    // Enable strict checking explicitly for this construction
    return node.Load((*plain)(c), yaml.WithKnownFields())
}
```

`node.Decode()` cannot accept options. It applies the `KnownFields` setting
of the load that produced the node (`Decoder.KnownFields(true)` or
`yaml.WithKnownFields()`), so a custom unmarshaler can simply delegate to it and
keep strict field checking (issues #321 and go-yaml/yaml#460). All other
options have their default values.
`node.Load()` does not inherit anything and applies exactly the options passed
to it.

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
err := node.Dump(&config)
if err != nil {
    return err
}

// Apply output formatting when serializing the node to YAML.
data, err := yaml.Dump(&node, yaml.WithIndent(2), yaml.WithExplicitStart())
fmt.Println(string(data))
// Output:
// ---
// name: myapp
// port: 8080
```

A node retains scalar and collection styles, but it does not store output
indentation or document-marker settings.
Apply those settings again when serializing the node.

**When to use:**

- Custom unmarshalers that need to set options themselves (`node.Load()`)
- Building Node trees with specific formatting (`node.Dump()`)
- Programmatic YAML manipulation with options
- Applying options explicitly at each relevant stage

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
        yaml.WithKnownFields(),       // Reject unknown fields (off by default)
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
    loader, err := yaml.NewLoader(reader)
    if err != nil {
        return err
    }

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
        data, err := os.ReadFile(f)
        if err != nil {
            return nil, err
        }
        var docs []any
        if err := yaml.Load(data, &docs, yaml.WithAllDocuments()); err != nil {
            return nil, err
        }
        allDocs = append(allDocs, docs...)
    }

    return yaml.Dump(allDocs, yaml.WithAllDocuments(), yaml.WithIndent(2))
}
```

## See Also

- [Options Guide](../options.md) - Complete option reference
- [API Documentation](https://pkg.go.dev/go.yaml.in/yaml/v4) - Full API
  reference
