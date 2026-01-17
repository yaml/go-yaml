go-yaml Processing Pipeline Internals
======================================

This document analyzes the load and dump stacks in the go-yaml implementation,
documenting each stage's input/output forms, the transforms performed, and
identifying where processing happens in the "wrong places" that could be
refactored for cleaner API hooks.

```
  LOAD (YAML → Native)      DUMP (Native → YAML)
  ────────────────────      ────────────────────
     (Native Value)     ←→    (Native Value)
            ↑                        ↓
       Constructor              Representer
            ↑                        ↓
         (Repr)                      ↓
            ↑                        ↓
        Resolver                     ↓
            ↑                        ↓
         (Nodes)        ←→        (Nodes)
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

See also:
- [Pipeline Overview Diagram](pipeline-overview.mmd)
- [Call Hierarchy Diagram](call-hierarchy.mmd)


## Load Stack

The load stack transforms YAML text into Go values through a series of stages.

**Control Flow:** The load stack uses a pull-based call hierarchy. Each outer
layer drives the process by requesting data from the layer below it on demand:

- Entry point (`Decoder.Decode` or `Node.Decode`) calls **Constructor**
- **Constructor** calls **Composer**.Parse() to get the next Node
- **Composer** calls **Parser**.Parse() to get the next Event
- **Parser** calls **Scanner**.Scan() (via peekToken) to get the next Token
- **Scanner** calls **Reader** functions (updateBuffer) to get more bytes
- **Resolver** is called by Composer (for tag inference) and Constructor (for value conversion)


### Reader

Handles input buffering and character encoding detection/conversion.

Info:
- File: internal/libyaml/reader.go (170 lines)
- Main Function: `func (parser *Parser) updateBuffer(length int) error`
- Input: `[]byte` or `io.Reader`
- Output: UTF-8 normalized bytes in `parser.buffer`
- Called From: `scanner.go / fetchNextToken / 688`
- Calls These:
  * `reader.go / determineEncoding       - Detects UTF-8/UTF-16 from BOM`
  * `reader.go / updateRawBuffer         - Reads more bytes from input source`

Transforms:
* Detects encoding from BOM (UTF-8, UTF-16LE, UTF-16BE)
* Converts non-UTF-8 input to UTF-8
* Validates UTF-8 sequences
* Buffers input for lookahead

Notes:
* Encoding is stored in `parser.encoding` field
* The reader is not a separate stage but integrated into the Parser struct


### Scanner

Lexical analysis - converts raw bytes into tokens.

Info:
- File: internal/libyaml/scanner.go (2599 lines)
- Main Function: `func (parser *Parser) Scan(token *Token) error`
- Input: UTF-8 bytes from reader buffer
- Output: `Token` struct
- Called From: `parser.go / peekToken / 56`
- Calls These:
  * `scanner.go / fetchMoreTokens        - Ensures token queue has enough tokens`
  * `scanner.go / fetchNextToken         - Dispatches to token fetchers by character`
  * `reader.go / updateBuffer            - Refills input buffer when needed`

Transforms:
* Character classification and dispatch (`fetchNextToken()` line 722)
* Flow level tracking (`flow_level` field)
* Indentation stack management (`indent`, `indents` fields)
* Simple key candidate tracking (`simple_keys` stack)
* Synthetic BLOCK_SEQUENCE_START/BLOCK_MAPPING_START/BLOCK_END token generation
* Escape sequence processing in quoted scalars
* Line folding in folded block scalars
* Chomping behavior for block scalars (`-` strip, `+` keep)
* URI decoding in tags

Notes:
* Scanner and Parser share the same `Parser` struct
* Tag handle and suffix remain separate at this stage (`Value` + `suffix` fields)
* Scalar style is recorded (Plain, SingleQuoted, DoubleQuoted, Literal, Folded)


### Parser

Syntactic analysis - converts token stream into event stream.

Info:
- File: internal/libyaml/parser.go (1174 lines)
- Main Function: `func (parser *Parser) Parse(event *Event) error`
- Input: `Token` stream (via internal `peekToken()`/`skipToken()`)
- Output: `Event` struct
- Called From: `composer.go / Composer.peek / 84`
- Calls These:
  * `parser.go / stateMachine            - Dispatches to parser state handlers`
  * `parser.go / peekToken               - Looks at next token without consuming`
  * `parser.go / skipToken               - Consumes current token`
  * `parser.go / parseStreamStart        - Handles stream start event`
  * `parser.go / parseDocumentStart      - Handles document boundaries`
  * `parser.go / parseNode               - Parses scalar/sequence/mapping nodes`

Transforms:
* LL(1) grammar production matching (22 parser states)
* **Tag handle → full URI resolution** (`!!str` → `tag:yaml.org,2002:str`)
* Token grouping (anchor + tag + scalar tokens → single SCALAR_EVENT)
* Comment attachment to events
* Implicit/quoted_implicit flag calculation
* Block structure tokens → hierarchical event pairs

Notes:
* Tag handle/suffix split is lost here - only full URI remains
* The `Implicit` flag indicates whether tag was omitted in source
* Comments are attached via `UnfoldComments()` and `setEventComments()`


### Composer

Builds the Node tree from the event stream.

Info:
- File: internal/libyaml/composer.go (320 lines)
- Main Function: `func (c *Composer) Parse() *Node`
- Input: `Event` stream (via internal `peek()`/`expect()`)
- Output: `*Node` tree
- Called From: `constructor.go / Constructor.unmarshal / 348`
- Calls These:
  * `parser.go / Parser.Parse            - Gets next event from parser`
  * `composer.go / peek                  - Peeks at next event type`
  * `composer.go / expect                - Consumes event of expected type`
  * `composer.go / node                  - Creates Node with tag/style`
  * `composer.go / scalar                - Builds ScalarNode from event`
  * `composer.go / mapping               - Builds MappingNode recursively`
  * `composer.go / sequence              - Builds SequenceNode recursively`
  * `composer.go / document              - Builds DocumentNode wrapper`
  * `resolver.go / resolve               - Infers tag for untagged scalars`

Transforms:
* Event sequence → tree structure
* **Tag short-form normalization** (`tag:yaml.org,2002:str` → `!!str`)
* **DEFAULT TYPE INFERENCE for untagged scalars** via `resolve("", value)` ⚠️
* Anchor registration in map (`c.anchors`)
* Alias name → pointer to target Node
* Style flag conversion (libyaml styles → Node Style flags)
* Comment transfer and reassignment

Notes:
* Byte-level Index is lost here (only Line/Column preserved)
* Implicit document distinction is lost
* In the current implementation, Composer calls `resolve()` for untagged scalars - this may be the wrong place (see Problems section)


### Resolver

Resolves tags to determine the type of each scalar value.

Info:
- File: internal/libyaml/resolver.go (170 lines)
- Main Function: `func resolve(tag string, in string) (rtag string, out any)`
- Input: Tag string + scalar value
- Output: Resolved tag + typed Go value
- Called From: `composer.go:182`, `constructor.go:702`, `representer.go:477`, `serializer.go:34`
- Calls These:
  * `resolver.go / parseTimestamp        - Parses ISO 8601 timestamps`
  * `strconv / ParseInt                  - Parses signed integers`
  * `strconv / ParseUint                 - Parses unsigned integers`
  * `strconv / ParseFloat                - Parses floating point numbers`

Transforms:
* Implicit tag resolution based on scalar content (`true` → `!!bool`, `42` → `!!int`)
* YAML 1.1 compatibility handling (sexagesimal, old bools)
* Timestamp parsing
* Special value recognition (`.nan`, `.inf`, `null`)

Notes:
* In go-yaml, this is not a separate stage but a function called from multiple places
* Called from Composer (to set Node.Tag) and Constructor (to get typed value) ⚠️
* The YAML spec treats Resolver as a distinct stage producing the "Representation Graph"


### Constructor

Converts Node tree into Go values.

Info:
- File: internal/libyaml/constructor.go (1183 lines)
- Main Function: `func (c *Constructor) Construct(n *Node, out reflect.Value) bool`
- Input: `*Node` tree
- Output: `reflect.Value` (Go values modified in place)
- Called From: `node.go / Node.Decode / 285`
- Calls These:
  * `constructor.go / prepare            - Checks for Unmarshaler interface`
  * `constructor.go / scalar             - Converts ScalarNode to Go value`
  * `constructor.go / mapping            - Converts MappingNode to map/struct`
  * `constructor.go / sequence           - Converts SequenceNode to slice`
  * `constructor.go / document           - Unwraps DocumentNode`
  * `constructor.go / alias              - Follows alias pointer, reconstructs`
  * `resolver.go / resolve               - Re-resolves tag to get typed value`

Transforms:
* Custom Unmarshaler interface detection and dispatch
* **Re-resolution of tag/value** via `resolve(n.Tag, n.Value)` ⚠️
* `indicatedString()` check for quoted scalars
* Type coercion (YAML types → Go types)
* Alias expansion (full reconstruction each time)
* Binary base64 decoding (`!!binary` tag)
* Merge key handling (`<<`)
* Struct field mapping via `getStructInfo()`

Notes:
* `resolve()` is called again here - duplicates work done in Composer
* Aliases are fully reconstructed each time (not shared)
* Node tree, comments, style, anchors, positions are all discarded


## Dump Stack

The dump stack transforms Go values (or Node trees) into YAML text.

**Control Flow:** The dump stack uses a push-based call hierarchy. Each outer
layer drives the process by pushing data down through the layers below it:

- Entry point (`Encoder.Encode` or `Node.Encode`) calls **Representer**
- **Representer** walks the Go value and calls emit() to push Events to **Emitter**
- If input is a `*Node`, **Representer** delegates to **Serializer** instead
- **Serializer** walks the Node tree and pushes Events to **Emitter**
- **Emitter** accumulates Events, formats output, and calls **Writer** to flush bytes
- **Resolver** is called by Representer and Serializer (to check if quoting/tags needed)


### Representer

Converts Go values directly to events (bypasses Node tree).

Info:
- File: internal/libyaml/representer.go (564 lines)
- Main Function: `func (r *Representer) MarshalDoc(tag string, in reflect.Value)`
- Input: `reflect.Value` + `Options`
- Output: Events emitted directly to Emitter
- Called From: `node.go / Node.Encode / 333`
- Calls These:
  * `representer.go / marshal            - Dispatches by Go type`
  * `representer.go / emit               - Sends event to emitter`
  * `representer.go / nodev              - Delegates Node to serializer`
  * `representer.go / mapv               - Marshals map with sorted keys`
  * `representer.go / structv            - Marshals struct fields`
  * `representer.go / slicev             - Marshals slice/array`
  * `representer.go / stringv            - Marshals string with style choice`
  * `resolver.go / resolve               - Checks if quoting needed`

Transforms:
* Go type dispatch (type switch at line 247)
* Custom Marshaler interface detection and dispatch
* Struct field ordering and filtering (exported, by tag, omitempty)
* Map key sorting (numeric-aware)
* **Resolve-check for quoting decisions** via `resolve()` ⚠️
* YAML 1.1 compatibility checks (`isBase60Float()`, `isOldBool()`)
* Style selection (literal for multiline strings)
* Invalid UTF-8 → base64 binary with `!!binary` tag
* Flow style from struct tags

Notes:
* Representer calls `resolve()` to determine if quoting is needed - this couples dump to load logic
* When input is `*Node`, delegates to Serializer instead


### Serializer

Converts Node tree to events (used when marshaling from Node).

Info:
- File: internal/libyaml/serializer.go (192 lines)
- Main Function: `func (r *Representer) node(node *Node, tail string)`
- Input: `*Node` tree
- Output: Events emitted to Emitter
- Called From: `representer.go / Representer.nodev / 567`
- Calls These:
  * `representer.go / emit               - Sends event to emitter`
  * `representer.go / nilv               - Emits null scalar`
  * `serializer.go / node (recursive)    - Walks child nodes`
  * `serializer.go / isSimpleCollection  - Checks if flow style appropriate`
  * `resolver.go / resolve               - Checks if tag can be elided`

Transforms:
* Node tree → event stream
* **Tag elision check** via `resolve()` ⚠️
* Force quoting when tag would be misresolved
* Flow style for "simple collections" (all scalar children)
* Comment placement/shifting (foot → tail)
* Style flag interpretation
* Invalid UTF-8 → base64

Notes:
* `resolve()` is called to check if tag can be elided - fourth place it's called


### Emitter

Converts events to YAML text output.

Info:
- File: internal/libyaml/emitter.go (2075 lines)
- Main Function: `func (emitter *Emitter) Emit(event *Event) error`
- Input: `Event` stream (queued in `emitter.events`)
- Output: UTF-8 bytes to `emitter.buffer`
- Called From: `representer.go / Representer.must / 210`
- Calls These:
  * `emitter.go / needMoreEvents         - Checks if more events needed for lookahead`
  * `emitter.go / analyzeEvent           - Analyzes scalar/tag for style decisions`
  * `emitter.go / stateMachine           - Dispatches to emitter state handlers`
  * `emitter.go / selectScalarStyle      - Final style selection (can override)`
  * `emitter.go / writeScalar            - Writes scalar with chosen style`
  * `writer.go / flush                   - Flushes buffer to output`

Transforms:
* Event accumulation for lookahead decisions
* **Final style selection** (`selectScalarStyle()` can override earlier choices)
* Simple key eligibility check (length ≤128, no multiline)
* Block vs flow style (context-dependent)
* Scalar analysis for style validity (`analyzeScalar()`)
* Line wrapping at `best_width`
* Indentation management
* Escape sequence encoding
* Tag directive shortening

Notes:
* Emitter can override style decisions made by Representer/Serializer
* Style decisions are split across multiple stages


### Writer

Flushes output buffer to destination.

Info:
- File: internal/libyaml/writer.go (32 lines)
- Main Function: `func (emitter *Emitter) flush() error`
- Input: `emitter.buffer`
- Output: bytes to `write_handler` callback
- Called From: `emitter.go / put / 19`
- Calls These:
  * `write_handler callback              - Configured output destination`

Transforms:
* Buffer flush to output destination

Notes:
* Very simple - just calls the configured write handler


## Intermediate Forms

The data representations passed between stages.


### Bytes

Raw input/output bytes.

At input: Raw bytes from file or string, potentially any encoding (UTF-8, UTF-16LE, UTF-16BE).

At output: UTF-8 encoded YAML text.


### Token

Scanner output - lexical tokens.

```go
type Token struct {
    Type       TokenType   // SCALAR_TOKEN, ALIAS_TOKEN, TAG_TOKEN, etc.
    StartMark  Mark        // Position: Index, Line, Column
    EndMark    Mark
    encoding   Encoding    // For STREAM_START_TOKEN only
    Value      []byte      // Scalar value, anchor name, or tag handle
    suffix     []byte      // Tag suffix (for TAG_TOKEN)
    prefix     []byte      // Tag directive prefix
    Style      ScalarStyle // Plain, SingleQuoted, DoubleQuoted, Literal, Folded
    major, minor int8      // For VERSION_DIRECTIVE_TOKEN
}
```

Location: `internal/libyaml/yaml.go:249`

Notes:
* Tag handle and suffix are still separate here
* Full byte-level position (Index) is available
* 23 token types defined


### Event

Parser output - syntactic events.

```go
type Event struct {
    Type             EventType   // SCALAR_EVENT, MAPPING_START_EVENT, etc.
    StartMark        Mark
    EndMark          Mark
    encoding         Encoding
    versionDirective *VersionDirective
    tagDirectives    []TagDirective
    HeadComment      []byte
    LineComment      []byte
    FootComment      []byte
    TailComment      []byte
    Anchor           []byte
    Tag              []byte      // FULL resolved URI (not handle+suffix)
    Value            []byte
    Implicit         bool        // Was tag omitted?
    quoted_implicit  bool        // Was tag omitted for quoted style?
    Style            Style
}
```

Location: `internal/libyaml/yaml.go:321`

Notes:
* Tag is now full URI - handle/suffix split is lost
* Comments are attached
* 11 event types defined


### Node

Composer output - tree structure.

```go
type Node struct {
    Kind          Kind     // DocumentNode, SequenceNode, MappingNode, ScalarNode, AliasNode
    Style         Style    // TaggedStyle, DoubleQuotedStyle, LiteralStyle, FlowStyle, etc.
    Tag           string   // SHORT form tag (!!str, !!int, etc.)
    Value         string   // Scalar content (unescaped, unquoted)
    Anchor        string
    Alias         *Node    // Points to target Node for AliasNode
    Content       []*Node  // Children for Document/Sequence/Mapping
    HeadComment   string
    LineComment   string
    FootComment   string
    Line, Column  int      // NOTE: Byte Index is lost
    // StreamNode-specific:
    Encoding      Encoding
    Version       *StreamVersionDirective
    TagDirectives []StreamTagDirective
}
```

Location: `internal/libyaml/node.go:129`

Notes:
* Tag is short form - full URI is lost
* Byte Index is lost - only Line/Column remain
* 6 node kinds, 6 style flags


### Repr (Representation Graph)

The Node tree after tag resolution.

In the YAML specification, the Representation Graph is a distinct stage where all tags have been fully resolved. In go-yaml's implementation, this is still represented using Node structures with resolved tags.

The "Repr" is conceptual - there's no separate struct. The transformation from Nodes to Repr happens when `resolve()` is called to determine implicit tags.


### Native Value

Go language values: structs, maps, slices, strings, ints, etc.

This is the final output of the Load stack and the input to the Dump stack. Represented as `reflect.Value` internally.


## Potential Problems and Inconsistencies

See also:
- [Resolver Problem Diagram](resolver-problem.mmd)

### 1. resolve() Called in Four Places

| Location | Stage | Purpose |
|----------|-------|---------|
| `composer.go:182` | Load | Set Node.Tag for untagged scalars |
| `constructor.go:702` | Load | Get actual Go value |
| `representer.go:473` | Dump | Check if quoting needed |
| `serializer.go:34` | Dump | Check if tag can be elided |

**Problem:** Same expensive resolution logic runs multiple times. Tags are resolved but values aren't stored.

**Better:** Store `(tag, resolvedValue)` at first resolution, or defer ALL resolution to Constructor.


### 2. Style Decisions Split Across Stages

**Load:** Scanner → Parser → Composer → Constructor (each touches style)

**Dump:** Representer → Serializer → Emitter (each can override style)

**Problem:** Hard to control or predict final output style. Emitter can override everything.

**Better:** Make style decisions once and respect them, or clearly separate "hints" from "requirements".


### 3. Information Lost at Each Stage

| Information | Lost At | Impact |
|-------------|---------|--------|
| Tag handle/suffix split | Parser | Can't reconstruct `!custom!type` |
| Byte-level Index | Composer | Less precise error positions |
| Implicit document flag | Composer | Can't round-trip explicit `---` |
| Full tag URI | Composer | Can't round-trip verbatim tags |


### 4. YAML 1.1 Compatibility Scattered

These checks appear in multiple files:
- `isBase60Float()` - representer.go, resolver.go
- `isOldBool()` - representer.go, resolver.go
- YAML 1.1 bool values (y/Y/yes/Yes/YES) - resolver.go

**Problem:** No single configuration point for YAML version semantics.


### 5. Representer Has Parser Logic

The representer calls `resolve()` to check if strings would be misresolved - this is parser-era logic living in the dump stack.

**Problem:** Changes to resolution affect both stacks. Round-trip safety depends on this coupling.


### 6. Scanner and Parser Share Struct

Both stages use the `Parser` struct, making it hard to:
- Expose tokens separately from events
- Have clean API boundaries between stages


## Anything Else I Missed?

Areas that may need deeper investigation:
- Comment handling complexity (attachment and reassignment rules)
- Anchor/alias reconstruction behavior
- Error propagation and position reporting
- The relationship between root package types and internal/libyaml types
