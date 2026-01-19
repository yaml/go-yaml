# YAML Options Guide

This guide explains how to use the go-yaml options system to control YAML
formatting and behavior.

## Quick Start

The simplest way to use go-yaml is with the default settings:

```go
import "go.yaml.in/yaml/v4"

// Uses v4 defaults (2-space indent, compact sequences)
dumper, _ := yaml.NewDumper(writer)
dumper.Dump(&myData)
dumper.Close()
```

But what if you want 4-space indentation like v3? Or strict field checking?
That's where options come in.

## Individual Options

**Note:** All options work with both package-level functions (`Load`, `Dump`,
`NewLoader`, `NewDumper`) and node-level methods (`node.Load()`, `node.Dump()`).

You can tweak specific settings with individual option functions:

```go
dumper, _ := yaml.NewDumper(writer,
    yaml.WithIndent(4),                 // 4-space indentation (v3 style)
    yaml.WithCompactSeqIndent(false),   // Non-compact lists (v3 style)
)
```

### Available Options

#### Dumper (Encoding) Options

**Boolean Options:** All boolean options support variadic arguments.
Call without arguments to enable (defaults to `true`), or pass `false` to
explicitly disable:

```go
yaml.WithExplicitStart()       // Enable (same as WithExplicitStart(true))
yaml.WithExplicitStart(false)  // Explicitly disable
```

##### `yaml.WithIndent(spaces int)`

Controls how many spaces to use for indentation.

```go
// 2-space indent (compact)
yaml.NewDumper(w, yaml.WithIndent(2))

// 4-space indent (readable)
yaml.NewDumper(w, yaml.WithIndent(4))

// 8-space indent (very spacious)
yaml.NewDumper(w, yaml.WithIndent(8))
```

**Example output with different indents:**

```yaml
# WithIndent(2)
servers:
  web:
    host: localhost
    port: 8080

# WithIndent(4)
servers:
    web:
        host: localhost
        port: 8080

# WithIndent(8)
servers:
        web:
                host: localhost
                port: 8080
```

##### `yaml.WithCompactSeqIndent(...bool)`

Controls whether the list indicator `- ` counts as part of the indentation.

```go
// Enable compact sequences (short form)
yaml.NewDumper(w,
    yaml.WithIndent(4),
    yaml.WithCompactSeqIndent(),
)
```

**Example:**

```yaml
# compact=true (v4 default)
items:
  - name: first
    value: 1
  - name: second
    value: 2

# compact=false (v3 style)
items:
    - name: first
      value: 1
    - name: second
      value: 2
```

##### `yaml.WithLineWidth(width int)`

Sets the preferred line width for wrapping long strings.
Use -1 or 0 for unlimited width.

```go
// Wrap at 40 characters
yaml.NewDumper(w, yaml.WithLineWidth(40))

// Unlimited width (no wrapping)
yaml.NewDumper(w, yaml.WithLineWidth(-1))
```

**Example:**

```yaml
# LineWidth=40
description: |
  This is a long description
  that wraps to multiple lines.

# LineWidth=-1
description: This is a long description that stays on one line
```

**Default:** 80 characters

##### `yaml.WithUnicode(...bool)`

Controls whether non-ASCII characters appear as-is or are escaped.

```go
// Allow unicode (default)
yaml.NewDumper(w, yaml.WithUnicode(true))

// Escape non-ASCII characters
yaml.NewDumper(w, yaml.WithUnicode(false))
```

**Example:**

```yaml
# WithUnicode(true) - default
name: café

# WithUnicode(false)
name: "caf\u00e9"
```

**Use case:** Set to false for ASCII-only output required by legacy systems.

**Default:** true

##### `yaml.WithCanonical(...bool)`

Forces strictly canonical YAML output with explicit tags.
Primarily for debugging and spec testing.

```go
yaml.NewDumper(w, yaml.WithCanonical(true))
```

**Example:**

```yaml
# Normal output
name: John
age: 30

# Canonical output
!!map {
  ? !!str "name"
  : !!str "John",
  ? !!str "age"
  : !!int "30",
}
```

**Default:** false

##### `yaml.WithLineBreak(lineBreak yaml.LineBreak)`

Sets the line ending style.
Available options: `yaml.LineBreakLN` (Unix), `yaml.LineBreakCR` (old Mac),
`yaml.LineBreakCRLN` (Windows).

```go
// Unix line endings (default)
yaml.NewDumper(w, yaml.WithLineBreak(yaml.LineBreakLN))

// Windows line endings
yaml.NewDumper(w, yaml.WithLineBreak(yaml.LineBreakCRLN))
```

**Default:** `yaml.LineBreakLN` (Unix `\n`)

##### `yaml.WithExplicitStart(...bool)`

Controls whether document start markers (`---`) are always emitted.

```go
// Always emit --- at document start (short form)
yaml.NewDumper(w, yaml.WithExplicitStart())
```

**Example:**

```yaml
# WithExplicitStart(true)
---
name: test

# WithExplicitStart(false) - default
name: test
```

**Use case:** Multi-document streams, explicit document boundaries.

**Default:** false

##### `yaml.WithExplicitEnd(...bool)`

Controls whether document end markers (`...`) are always emitted.

```go
// Always emit ... at document end (short form)
yaml.NewDumper(w, yaml.WithExplicitEnd())
```

**Example:**

```yaml
# WithExplicitEnd(true)
name: test
...

# WithExplicitEnd(false) - default
name: test
```

**Use case:** Streaming scenarios where document end must be explicit.

**Default:** false

##### `yaml.WithFlowSimpleCollections(...bool)`

Controls whether simple collections use flow style.
Simple collections are sequences and mappings that:

- Contain only scalar values (no nested collections)
- Fit within the line width when rendered in flow style

```go
// Use flow style for simple collections (short form)
yaml.NewDumper(w, yaml.WithFlowSimpleCollections())
```

**Example:**

```yaml
# WithFlowSimpleCollections(true)
config:
  tags: [web, api, prod]
  metadata: {version: 1.0, author: admin}
  nested:  # Not simple - has nested collections
    items:
      - name: foo

# WithFlowSimpleCollections(false) - default
config:
  tags:
    - web
    - api
    - prod
  metadata:
    version: 1.0
    author: admin
```

**Use case:** Compact output for simple data structures, JSON-like formatting.

**Default:** false

##### `yaml.WithQuotePreference(style yaml.QuoteStyle)`

Controls which type of quotes to use when quoting is required by the YAML spec.

**Important:** This option only affects strings that *require* quoting. Plain strings that don't need quoting remain unquoted regardless of this setting.

Quoting is required for:
- Strings that look like other YAML types (true, false, null, 123, etc.)
- Strings with leading/trailing whitespace
- Strings containing special YAML syntax characters
- Empty strings in certain contexts

```go
// Use single quotes when quoting is required (v4 default)
yaml.NewDumper(w, yaml.WithQuotePreference(yaml.QuoteSingle))

// Use double quotes when quoting is required
yaml.NewDumper(w, yaml.WithQuotePreference(yaml.QuoteDouble))

// Use legacy v2/v3 behavior (mixed quoting)
yaml.NewDumper(w, yaml.WithQuotePreference(yaml.QuoteLegacy))
```

**Example:**

```yaml
# WithQuotePreference(yaml.QuoteSingle) - v4 default
plain: hello       # Plain string stays plain
bool: 'true'       # Looks like bool, gets single quotes
number: '123'      # Looks like number, gets single quotes
spaces: ' hello'   # Leading space, gets single quotes

# WithQuotePreference(yaml.QuoteDouble)
plain: hello       # Plain string stays plain
bool: "true"       # Looks like bool, gets double quotes
number: "123"      # Looks like number, gets double quotes
spaces: " hello"   # Leading space, gets double quotes

# WithQuotePreference(yaml.QuoteLegacy) - v2/v3 behavior
plain: hello       # Plain string stays plain
bool: "true"       # Looks like bool, gets double quotes
number: "123"      # Looks like number, gets double quotes
spaces: ' hello'   # Leading space, gets single quotes
```

**Use case:**
- `QuoteSingle`: Modern YAML style with single quotes (cleaner, less escaping)
- `QuoteDouble`: When you prefer double quotes or need consistency with JSON-style
- `QuoteLegacy`: Backward compatibility with go-yaml v2/v3 output

**Default:**
- v4: `QuoteSingle`
- v2/v3: `QuoteLegacy`

#### Loader (Decoding) Options

**Boolean Options:** All boolean options support variadic arguments.
Call without arguments to enable (defaults to `true`), or pass `false` to
explicitly disable:

```go
yaml.WithKnownFields()       // Enable (same as WithKnownFields(true))
yaml.WithKnownFields(false)  // Explicitly disable
```

##### `yaml.WithKnownFields(...bool)`

When enabled, loading will fail if the YAML contains fields that don't exist in
your struct.

```go
type Config struct {
    Name string `yaml:"name"`
    Port int    `yaml:"port"`
}

// Enable strict field checking (short form)
loader, _ := yaml.NewLoader(reader,
    yaml.WithKnownFields(),
)

// This will fail if YAML contains "unknown_field"
var config Config
err := loader.Load(&config)
```

**Great for:** Catching typos in config files, enforcing strict schemas.

##### `yaml.WithSingleDocument(...bool)`

Only load the first YAML document, then return EOF.

```go
// Enable single document mode (short form)
loader, _ := yaml.NewLoader(reader,
    yaml.WithSingleDocument(),
)

var doc1 Doc
loader.Load(&doc1)  // Loads first document

var doc2 Doc
loader.Load(&doc2)  // Returns io.EOF

// Dynamic based on environment/config
singleDoc := os.Getenv("SINGLE_DOC") == "true"
loader, _ := yaml.NewLoader(reader,
    yaml.WithSingleDocument(singleDoc),
)
```

**Use case:** When you expect exactly one document and want to catch extras.

**Default:** false (multi-document mode)

##### `yaml.WithUniqueKeys(...bool)`

Controls duplicate key detection in mappings.
When enabled, loading fails if duplicate keys are found.

```go
// Enforce unique keys (default - short form)
loader, _ := yaml.NewLoader(reader,
    yaml.WithUniqueKeys(),
)

// Allow duplicates (not recommended)
loader, _ := yaml.NewLoader(reader,
    yaml.WithUniqueKeys(false),
)
```

**Example YAML that would fail:**

```yaml
admin: false
admin: true  # Error: duplicate key
```

**Security note:** This prevents key override attacks where an attacker could
override security-critical keys.

**Default:** true (enabled)

## Version-Specific Option Presets

Instead of setting options one by one, you can use version presets that match
go-yaml v2, v3, or v4 behavior.

### What's the difference?

| Option                | v2      | v3      | v4     |
|-----------------------|---------|---------|--------|
| Indent                | 2       | 4       | 2      |
| Compact Sequences     | No      | No      | Yes    |
| Line Width            | 80      | 80      | 80     |
| Unicode               | Yes     | Yes     | Yes    |
| Unique Keys           | Yes     | Yes     | Yes    |
| Quote Preference      | Legacy  | Legacy  | Single |

**Note:** v4 uses compact sequences (modern YAML standard) while v2/v3 preserve
the historical non-compact behavior for backward compatibility.

### Using v2 Options

```go
dumper, _ := yaml.NewDumper(writer, yaml.V2)
```

**When to use:** Matching go-yaml v2 output format (2-space indent).

### Using v3 Options

```go
dumper, _ := yaml.NewDumper(writer, yaml.V3)
```

**When to use:** Explicitly requesting v3 behavior (4-space indent, non-compact
sequences).
Use this for compatibility with older go-yaml v3 output or when working with
code that expects v3 formatting.

### Using v4 Options (Default)

```go
dumper, _ := yaml.NewDumper(writer, yaml.V4)
```

**When to use:** Modern YAML output with 2-space indent and compact sequences.
This is the default, so you usually don't need to specify `yaml.V4` unless you
want to be explicit.

## Mixing Presets with Individual Options

You can start with a preset and override specific options.
**Options apply left-to-right**, so later options override earlier ones.

```go
// Start with v3 defaults (4-space), then override to 2-space
dumper, _ := yaml.NewDumper(writer,
    yaml.V3,
    yaml.WithIndent(2),  // This wins
)
```

**More examples:**

```go
// Default (v4) + override to 4-space indent
dumper, _ := yaml.NewDumper(writer,
    yaml.WithIndent(4),  // Overrides default 2-space
)

// Start with 2-space, then apply v3 (4-space wins)
dumper, _ := yaml.NewDumper(writer,
    yaml.WithIndent(2),
    yaml.V3,  // This overrides to 4
)
```

## Loading Options from YAML Config

You can load your YAML processing options from a YAML configuration file! This
is great for making formatting configurable.

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

dumper, _ := yaml.NewDumper(writer, opts)
```

**The YAML field names are:**
- `indent` - Number of spaces
- `compact-seq-indent` - Boolean
- `line-width` - Integer
- `unicode` - Boolean
- `canonical` - Boolean
- `line-break` - String (ln, cr, crln)
- `explicit-start` - Boolean
- `explicit-end` - Boolean
- `flow-simple-coll` - Boolean
- `quote-preference` - String (single, double, legacy)
- `known-fields` - Boolean
- `single-document` - Boolean
- `unique-keys` - Boolean

**Any field not in your config uses the version's defaults.**

## Real-World Examples

### Example 1: Strict Config File Loader

```go
type AppConfig struct {
    Database DatabaseConfig `yaml:"database"`
    Server   ServerConfig   `yaml:"server"`
}

func LoadConfig(filename string) (*AppConfig, error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    loader, err := yaml.NewLoader(f,
        yaml.V4,
        yaml.WithKnownFields(),          // Catch typos (defaults to true)
        yaml.WithSingleDocument(),       // Expect exactly one doc
    )
    if err != nil {
        return nil, err
    }

    var config AppConfig
    if err := loader.Load(&config); err != nil {
        return nil, err
    }

    return &config, nil
}
```

### Example 2: Matching External Tool Format

Your CI system expects 2-space indented YAML:

```go
func GenerateCI(config *CIConfig) ([]byte, error) {
    var buf bytes.Buffer

    dumper, err := yaml.NewDumper(&buf,
        yaml.V2,
    )
    if err != nil {
        return nil, err
    }

    if err := dumper.Dump(config); err != nil {
        return nil, err
    }
    dumper.Close()

    return buf.Bytes(), nil
}
```

### Example 3: User-Configurable Formatter

```go
func FormatYAML(input []byte, userPrefs string) ([]byte, error) {
    // Load user's formatting preferences
    opts, err := yaml.OptsYAML(userPrefs)
    if err != nil {
        return nil, err
    }

    // Parse input
    var node yaml.Node
    if err := yaml.Load(input, &node); err != nil {
        return nil, err
    }

    // Reformat with user preferences
    var buf bytes.Buffer
    dumper, _ := yaml.NewDumper(&buf, opts)
    dumper.Dump(&node)
    dumper.Close()

    return buf.Bytes(), nil
}
```

### Example 4: Multi-Document Stream with Options

```go
func ProcessMultiDocs(reader io.Reader) error {
    loader, err := yaml.NewLoader(reader,
        yaml.WithKnownFields(false),  // Allow unknown fields (override default)
    )
    if err != nil {
        return err
    }

    for {
        var doc Document
        err := loader.Load(&doc)
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        // Process document...
        processDocument(&doc)
    }

    return nil
}
```

## Common Patterns

### Pattern: Application Config Loader

```go
loader, _ := yaml.NewLoader(configFile,
    yaml.WithSingleDocument(),    // Only one config file
    yaml.WithKnownFields(),       // Strict validation (defaults to true)
)
```

### Pattern: Default YAML Output

```go
dumper, _ := yaml.NewDumper(output)  // Uses v4 defaults (2-space, compact)
```

### Pattern: Match Legacy Format

```go
dumper, _ := yaml.NewDumper(output, yaml.V3)  // 4-space like old go-yaml
```

## Tips & Tricks

**Default is v4:** `yaml.NewDumper(w)` uses v4 defaults (2-space indent,
compact sequences).

**Order matters:** Options are applied left-to-right, with later options
overriding earlier ones.
For example, `yaml.WithIndent(2), yaml.V3` gives you 4 spaces (V3 wins),
while `yaml.V3, yaml.WithIndent(2)` gives you 2 spaces.

**Load from files:** Use `yaml.OptsYAML()` to make formatting configurable by
users.

**Validation:** Use `WithKnownFields()` to catch config file typos early.

**One document:** Use `WithSingleDocument()` when you expect exactly one YAML
document.

**Test your options:** Run examples with different options to see the output
format.

## See Also

- [Dumping and Loading API Guide](dump-load-api.md) - Complete guide to Dump/Load APIs
- [API Documentation](https://pkg.go.dev/go.yaml.in/yaml/v4) - Full API
  reference
- [Examples](../example/README.md) - Runnable code examples
