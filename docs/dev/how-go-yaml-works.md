# How go-yaml Works

This document explains the internal architecture of YAML processing in
go-yaml.
Understanding this helps you debug issues, contribute to the project, and
appreciate why YAML loading isn't as simple as "parsing."

## Common Misconceptions

### "Loading" is not "Parsing"

Many people incorrectly refer to YAML loading as "parsing," and some
implementations even name their load function `parse()`.
This is technically wrong and obscures what's really happening.

**Parsing** is just one stage in a multi-stage pipeline.
A parser applies grammar rules to a token stream.
**Loading** encompasses the entire transformation from YAML bytes to
native language values, which involves many more steps than just parsing.

### YAML Processing is a Pipeline

YAML processing isn't a single monolithic operation.
Both loading and dumping are **pipelines of transforms**, where each stage:
- Has a single, well-defined responsibility
- Consumes input in one representation
- Produces output in a different representation
- Can be inspected and debugged independently

The two user-facing functions in go-yaml are `Load()` and `Dump()`, but
these are just the entry points to much deeper pipelines.

## The Big Picture: Paired Pipelines

Load and Dump are mirror-image stacks of transforms with mostly matching
stages.
Data flows through different representations at each stage:

```
  LOAD (YAML → Native)      DUMP (Native → YAML)
  ────────────────────      ────────────────────
     (Native Value)     ←→    (Native Value)
            ↑                        ↓
       Constructor              Representer
            ↑                        ↓
         (Nodes)             (Nodes - tagged)
            ↑                        ↓
        Resolver                 Desolver
            ↑                        ↓
   (Nodes - unresolved)   (Nodes - minimal tags)
            ↑                        ↓
        Composer                Serializer
            ↑                        ↓
        (Events)        ←→       (Events)
            ↑                        ↓
         Parser                      ↓
            ↑                        ↓
        (Tokens)                     ↓
            ↑                        ↓
         Scanner                  Emitter
            ↑                        ↓
      (Code Points)     ←→     (Code Points)
            ↑                        ↓
         Reader                    Writer
            ↑                        ↓
       (Raw Bytes)      ←→      (Raw Bytes)
```

**Stack Symmetry Improvements**

The v4 refactor has made the stacks more symmetric:

1. **3-Stage Pipelines**: Both Load and Dump now have clean 3-stage pipelines:
   - **Load:** Composer → Resolver → Constructor
   - **Dump:** Representer → Desolver → Serializer
2. **Symmetric Tag Handling**:
   - **Resolver** (Load) infers tags from content
   - **Desolver** (Dump) removes inferable tags - NEW in v4
3. **Node Trees on Both Sides**: Both stacks work with Node trees as intermediate representation
4. **Remaining Asymmetry**: Scanner+Parser on Load vs single Emitter on Dump (due to YAML's complex indentation-based syntax)

## Data Representations

Each stage consumes and produces data in specific representations:

### Raw Bytes
The actual file contents or byte stream.
No interpretation has been done yet.

### Code Points
Unicode characters after encoding detection and conversion.
The Reader handles UTF-8, UTF-16LE, and UTF-16BE encoding.

### Tokens
Lexical units produced by the Scanner.
Examples: `BLOCK_MAPPING_START_TOKEN`, `SCALAR_TOKEN`, `KEY_TOKEN`, `VALUE_TOKEN`, `ANCHOR_TOKEN`.

Tokens have no nested structure — they're a flat stream that describes
YAML syntax at the character level (indentation, indicators, scalars).

### Events
Structural units produced by the Parser.
Examples: `MAPPING_START_EVENT`, `MAPPING_END_EVENT`, `SCALAR_EVENT`, `ALIAS_EVENT`.

Events represent the grammar-level structure of YAML.
The Parser validates that tokens conform to YAML grammar rules and produces
a cleaner event stream.

### Nodes
The tree structure built by the Composer.
Each Node has a kind (Document, Mapping, Sequence, Scalar, Alias), value,
tag, style, and position.

At this stage, anchors are resolved to build the graph structure, but
tags are still in their raw form (`!!str`, `!!int`, or implicit).

### Repr (Representation Graph)
The node tree after the Resolver has processed tags.
Tags are resolved according to YAML tag resolution rules (implicit typing,
tag directives, etc.).

In go-yaml's implementation, this is still represented using Node
structures, but with all tags fully resolved.
The "Repr" is a conceptual stage from the YAML specification.

### Native Value
Go language values: structs, maps, slices, strings, ints, etc.
This is what application code works with.

## Loading Pipeline Stages

### 1. Reader (Raw Bytes → Code Points)

**File**: `internal/libyaml/reader.go`

The Reader handles input encoding:
- Detects encoding (UTF-8, UTF-16LE, UTF-16BE) via BOM or heuristics
- Converts bytes to Unicode code points
- Buffers input for efficient scanning

This stage ensures the Scanner works with a consistent Unicode stream
regardless of input encoding.

### 2. Scanner (Code Points → Tokens)

**File**: `internal/libyaml/scanner.go`

The Scanner performs lexical analysis:
- Tracks indentation levels
- Identifies block vs. flow context
- Detects simple keys (for compact mappings like `key: value`)
- Produces a stream of tokens

This is the most complex stage because YAML's indentation-based syntax
requires careful context tracking.

**Example tokens** for `foo: bar`:
- `STREAM_START_TOKEN`
- `BLOCK_MAPPING_START_TOKEN`
- `KEY_TOKEN`
- `SCALAR_TOKEN` (value: "foo")
- `VALUE_TOKEN`
- `SCALAR_TOKEN` (value: "bar")
- `BLOCK_END_TOKEN`
- `STREAM_END_TOKEN`

### 3. Parser (Tokens → Events)

**File**: `internal/libyaml/parser.go`

The Parser applies YAML grammar:
- Consumes the token stream
- Validates structure according to YAML grammar rules
- Produces a cleaner event stream

**This is what "parsing" actually means** — applying grammar rules to a
token stream.

**Example events** for `foo: bar`:
- `STREAM_START_EVENT`
- `DOCUMENT_START_EVENT`
- `MAPPING_START_EVENT`
- `SCALAR_EVENT` (value: "foo")
- `SCALAR_EVENT` (value: "bar")
- `MAPPING_END_EVENT`
- `DOCUMENT_END_EVENT`
- `STREAM_END_EVENT`

### 4. Composer (Events → Nodes)

**File**: `internal/libyaml/composer.go`

The Composer builds the node tree:
- Creates Document, Mapping, Sequence, and Scalar nodes
- Registers anchors and resolves aliases to build the graph structure
- Handles multi-document streams
- Attaches comments to nodes

The output is a tree (or graph, with aliases) of Node objects.

### 5. Resolver (Nodes → Repr)

**File**: `internal/libyaml/resolver.go`

The Resolver handles tag resolution:
- Determines implicit tags based on scalar content
  (e.g., `true` → `!!bool`, `42` → `!!int`, `foo` → `!!str`)
- Processes explicit tags (e.g., `!!str 42` forces string type)
- Applies tag directives from document headers
- Produces the Representation Graph

In go-yaml's implementation, the Repr is the node tree with fully
resolved tags.

### 6. Constructor (Repr → Native Values)

**File**: `internal/libyaml/constructor.go`

The Constructor converts YAML to Go:
- Maps YAML types to Go types (`!!str` → string, `!!seq` → slice)
- Handles struct field mapping via reflection
- Calls custom `UnmarshalYAML` methods when defined
- Supports `encoding.TextUnmarshaler` interface
- Tracks alias depth for security

This is where YAML becomes usable Go data structures.

## Dumping Pipeline Stages

### 1. Representer (Native Values → Tagged Nodes)

**File**: `internal/libyaml/representer.go`

The Representer converts Go values to a tagged Node tree:
- Handles basic types (maps, structs, slices, strings, numbers, bools)
- Calls custom `MarshalYAML` methods when defined
- Supports `encoding.TextMarshaler` interface
- Makes style decisions (literal vs. quoted scalars, flow vs. block
  collections)
- Processes struct tags (`yaml:"name,omitempty,flow"`)
- Assigns explicit tags to all nodes (`!!str`, `!!int`, `!!map`, etc.)
- **Returns a Node tree** (no longer emits events directly)

### 2. Desolver (Tagged Nodes → Minimal Nodes)

**File**: `internal/libyaml/desolver.go`

The Desolver removes inferable tags (NEW in v4, inverse of Resolver):
- Walks the tagged node tree from Representer
- Removes tags that can be inferred during parsing
- Preserves explicit tags when content would be misresolved
- Produces cleaner YAML output without unnecessary type annotations
- **Example:** `!!str "hello"` becomes `hello`, but `!!str "42"` keeps the tag

### 3. Serializer (Nodes → Events)

**File**: `internal/libyaml/serializer.go`

The Serializer linearizes the node tree:
- Walks the node tree depth-first
- Produces a stream of events
- Handles anchor assignment for sharing/circular references
- Determines whether collections should use flow style
- Pushes events to the Emitter

### 4. Emitter (Events → Code Points)

**File**: `internal/libyaml/emitter.go`

The Emitter generates formatted YAML:
- Converts events to YAML text
- Handles indentation and line wrapping
- Chooses between different scalar styles (plain, quoted, literal,
  folded)
- Produces Unicode code points
- Supports canonical output mode

This stage combines the work that Scanner+Parser do on the Load side.

### 5. Writer (Code Points → Raw Bytes)

**File**: `internal/libyaml/writer.go`

The Writer handles output:
- Converts Unicode code points to bytes
- Handles output encoding
- Buffers writes for efficiency
- Writes to the output stream

## Why This Architecture Matters

### Single Responsibility

Each stage has one job and does it well.
This makes the code easier to understand, test, and maintain.

### Debuggability

You can inspect intermediate representations at any stage.
The `go-yaml` CLI tool leverages this to show tokens, events, and nodes.

### Extensibility

A plugin system allows hooking into any stage to customize behavior.
Want custom tag resolution? Hook the Resolver.
Want custom output formatting? Hook the Emitter.

### Spec Compliance

The architecture follows the YAML specification's terminology and
processing model.
This makes go-yaml easier to understand for anyone familiar with the
YAML spec.

## Debugging with the go-yaml CLI Tool

The `go-yaml` command-line tool can show you each stage of processing:

```bash
# Show tokens
go-yaml -t <<< 'foo: bar'

# Show events
go-yaml -e <<< 'foo: bar'

# Show node tree
go-yaml -n <<< 'foo: bar'
```

This is invaluable for understanding what's happening at each stage and
debugging parsing issues.

See the [main README](../README.md#the-go-yaml-cli-tool) for more details
on using the CLI tool.

## Summary

YAML processing is a pipeline, not a single operation:

- **Loading** flows through: Reader → Scanner → Parser → Composer →
  Resolver → Constructor
- **Dumping** flows through: Representer → Desolver → Serializer → Emitter → Writer
- Each stage transforms data from one representation to another
- The v4 refactor has made the stacks more symmetric with 3-stage pipelines:
  - **Load:** Composer → Resolver → Constructor
  - **Dump:** Representer → Desolver → Serializer
- "Parsing" is just one stage (tokens → events), not the whole process

Understanding these stages helps you work with YAML more effectively,
debug issues faster, and appreciate the complexity hidden behind the
simple `yaml.Load()` and `yaml.Dump()` functions.
