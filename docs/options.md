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

**Note:** Options affect the stages that use them.
Loading options do not configure dumping, and dumping options do not configure
loading.
`node.Load()` constructs Go values from an already composed graph, so parser
settings do not apply and explicit Alias-Data policies are rejected.
`node.Dump()` builds a node; apply indentation and document-marker options again
when serializing that node to YAML.

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
Call without arguments to enable, or pass `false` to disable.
This argument shorthand does not indicate whether the option is enabled in the
library defaults:

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
# WithIndent(4), compact=true
items:
  - name: first
    value: 1
  - name: second
    value: 2

# WithIndent(4), compact=false (v3 preset)
items:
    - name: first
      value: 1
    - name: second
      value: 2
```

With the v4 defaults (`WithIndent(2)` and compact sequences), the `-` indicator
is not indented beneath `items:`.

##### `yaml.WithLineWidth(width int)`

Sets the preferred line width for wrapping long strings.
Use -1 or 0 for unlimited width.

```go
// Prefer wrapping at 40 characters
yaml.NewDumper(w, yaml.WithLineWidth(40))

// Unlimited width (no wrapping)
yaml.NewDumper(w, yaml.WithLineWidth(-1))
```

**Example:**

```yaml
# LineWidth=40
description: This is a long description that
  stays on one line

# LineWidth=-1
description: This is a long description that stays on one line
```

Wrapping preserves the scalar value; it does not insert content newlines.
Long words may exceed the preferred width.

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
name: "caf\xE9"
```

**Use case:** Set to false for ASCII-only output required by legacy systems.

**Default:** true

##### `yaml.WithCanonical(...bool)`

Selects the emitter's canonical layout, with explicit document starts, flow
collections, and quoted scalars.
Tags still depend on the node graph and desolver; this option alone does not
force an explicit tag on every value.
Primarily for debugging.
In this mode, an integer such as `30` can reload as a string because its tag
was removed before the emitter quoted it.
Use ordinary output for type-preserving round trips.

```go
yaml.NewDumper(w, yaml.WithCanonical(true))
```

**Example:**

```yaml
# Normal output for a struct with name and age fields
name: John
age: 30

# WithCanonical(true)
---
{
  ? "name"
  : "John",
  ? "age"
  : "30",
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
  metadata: {version: v1, author: admin}
  nested:  # Not simple - has nested collections
    items:
    - {name: foo}

# WithFlowSimpleCollections(false) - default
config:
  tags:
  - web
  - api
  - prod
  metadata:
    version: v1
    author: admin
  nested:
    items:
    - name: foo
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
Call without arguments to enable, or pass `false` to disable.
This argument shorthand does not indicate whether the option is enabled in the
library defaults:

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

**Default:** false.
Unknown struct fields are ignored unless strict checking is enabled.

##### `yaml.WithSingleDocument(...bool)`

Only load the first YAML document, then return `io.EOF` on subsequent
`Loader.Load` calls.
Additional documents are left unread and unvalidated.

```go
// Enable single document mode (short form)
loader, _ := yaml.NewLoader(reader,
    yaml.WithSingleDocument(),
)

var doc1 Doc
loader.Load(&doc1)  // Loads first document

var doc2 Doc
loader.Load(&doc2)  // Returns io.EOF even if more documents are present

// Dynamic based on environment/config
singleDoc := os.Getenv("SINGLE_DOC") == "true"
loader, _ := yaml.NewLoader(reader,
    yaml.WithSingleDocument(singleDoc),
)
```

**Use case:** Intentionally process only the first document in a stream.

**Default:** false (`Loader.Load` can continue reading subsequent documents).

Package-level `yaml.Load` rejects additional documents by default.
Passing `WithSingleDocument()` skips that validation, including syntax checks
on later documents.
Omit this option when loading a configuration that must contain exactly one
document.

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

##### `yaml.WithAllDocuments(...bool)`

Package-level `Load` can load every document into a slice:

```go
var docs []Config
err := yaml.Load(input, &docs, yaml.WithAllDocuments())
```

Package-level `Dump` can write a slice as a document stream:

```go
data, err := yaml.Dump(docs, yaml.WithAllDocuments())
```

The loading target must be a pointer to a slice; the dumping input must be a
slice.
An empty stream loads as an empty slice in this mode.
For incremental processing, use `NewLoader` or `NewDumper` and call their
methods once per document.

**Default:** false.

##### `yaml.WithStreamNodes(...bool)`

When loading into `yaml.Node`, include stream boundary nodes alongside document
nodes to inspect encoding and directives:

```go
loader, err := yaml.NewLoader(reader, yaml.WithStreamNodes())
if err != nil {
    return err
}
```

Read nodes until `io.EOF`.
Stream nodes carry metadata in `Node.Stream`; ordinary document nodes carry
content in `Node.Content`.
See the [migration guide](v3-to-v4-migration.md#stream-nodes) for a full loop.

**Default:** false.

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
| Known Fields          | No      | No      | No     |
| Quote Preference      | Legacy  | Legacy  | Single |

**Note:** v4 uses compact sequences while the v2/v3 presets select
the historical non-compact behavior for backward compatibility.

### Using v2 Options

```go
dumper, _ := yaml.NewDumper(writer, yaml.WithV2Defaults())
```

**When to use:** Matching go-yaml v2 output format (2-space indent).

### Using v3 Options

```go
dumper, _ := yaml.NewDumper(writer, yaml.WithV3Defaults())
```

**When to use:** Explicitly requesting v3 behavior (4-space indent, non-compact
sequences).
Use this for compatibility with older go-yaml v3 output or when working with
code that expects v3 formatting.

### Using v4 Options (Default)

```go
dumper, _ := yaml.NewDumper(writer, yaml.WithV4Defaults())
```

**When to use:** Modern YAML output with 2-space indent and compact sequences.
This is the default, so you usually don't need to specify `yaml.WithV4Defaults()` unless you
want to be explicit.

## Mixing Presets with Individual Options

You can start with a preset and override specific options.
**Options apply left-to-right**, so later options override earlier ones.

```go
// Start with v3 defaults (4-space), then override to 2-space
dumper, _ := yaml.NewDumper(writer,
    yaml.WithV3Defaults(),
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
    yaml.WithV3Defaults(),  // This overrides to 4
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
- `known-fields` - Boolean
- `single-document` - Boolean
- `unique-keys` - Boolean

Unspecified fields leave earlier options unchanged.
Without earlier overrides, they use the defaults.
To clear an earlier single-document setting, apply `WithSingleDocument(false)`
after the parsed configuration.
`quote-preference` is not accepted by `OptsYAML`; use `WithQuotePreference` in
Go.
The CLI can select quote behavior through the `v2`, `v3`, and `v4` presets.
Stream-node and all-document processing are selected through Go options or
the CLI's `-o` flags, rather than through `OptsYAML`.

## Real-World Examples

### Example 1: Strict Config File Loader

```go
type AppConfig struct {
    Database DatabaseConfig `yaml:"database"`
    Server   ServerConfig   `yaml:"server"`
}

func LoadConfig(filename string) (*AppConfig, error) {
    input, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    var config AppConfig
    if err := yaml.Load(input, &config,
        yaml.WithV4Defaults(),
        yaml.WithKnownFields(), // Reject unknown fields
    ); err != nil {
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
        yaml.WithV2Defaults(),
    )
    if err != nil {
        return nil, err
    }

    if err := dumper.Dump(config); err != nil {
        return nil, err
    }
    if err := dumper.Close(); err != nil {
        return nil, err
    }

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
    dumper, err := yaml.NewDumper(&buf, opts)
    if err != nil {
        return nil, err
    }
    if err := dumper.Dump(&node); err != nil {
        return nil, err
    }
    if err := dumper.Close(); err != nil {
        return nil, err
    }

    return buf.Bytes(), nil
}
```

### Example 4: Multi-Document Stream with Options

```go
func ProcessMultiDocs(reader io.Reader) error {
    loader, err := yaml.NewLoader(reader,
        yaml.WithKnownFields(false),  // Allow unknown fields (the default)
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

Use package-level `Load` to reject additional documents:

```go
err := yaml.Load(input, &config,
    yaml.WithKnownFields(),
    yaml.WithUniqueKeys(),
)
```

### Pattern: Default YAML Output

```go
dumper, _ := yaml.NewDumper(output)  // Uses v4 defaults (2-space, compact)
```

### Pattern: Match Legacy Format

```go
dumper, _ := yaml.NewDumper(output, yaml.WithV3Defaults())  // 4-space like old go-yaml
```

## Tips & Tricks

**Default is v4:** `yaml.NewDumper(w)` uses v4 defaults (2-space indent,
compact sequences).

**Order matters:** Options are applied left-to-right, with later options
overriding earlier ones.
For example, `yaml.WithIndent(2), yaml.WithV3Defaults()` gives you 4 spaces (V3 wins),
while `yaml.WithV3Defaults(), yaml.WithIndent(2)` gives you 2 spaces.

**Load from files:** Use `yaml.OptsYAML()` to make formatting configurable by
users.

**Validation:** Use `WithKnownFields()` to catch config file typos early.

**One document:** Use `yaml.Load` without `WithSingleDocument()` to reject
extra YAML documents.
Use `WithSingleDocument()` only to stop after the first document and ignore
the remaining input.

**Test your options:** Run examples with different options to see the output
format.

## See Also

- [Dumping and Loading API Guide](dev/dump-load-api.md) - Complete guide to Dump/Load APIs
- [API Documentation](https://pkg.go.dev/go.yaml.in/yaml/v4) - Full API
  reference
- [Examples](../example/README.md) - Runnable code examples
