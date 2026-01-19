go-yaml Internals
=================

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
- [Comment Flow Diagram](comment-flow.mmd)


## Load Stack

The load stack transforms YAML text into Go values through a series of stages.

**Control Flow:** The load stack uses a pull-based call hierarchy. Entry points
(such as `Unmarshal()`, `Decoder.Decode()`, or `Node.Decode()`) orchestrate the
process by creating and coordinating the stages:

- **Entry points** create a **Composer** (which owns a **Parser**)
- **Entry points** call **Composer**.Parse() to get Node trees
- **Composer** calls **Parser**.Parse() to get Events (Parser and Scanner share the same struct)
- **Parser** calls its own **Scanner** methods (Scan via peekToken) to get Tokens
- **Scanner** calls **Reader** functions (updateBuffer) to get more bytes
- **Entry points** create a **Constructor** and call Construct() to convert Nodes to Go values
- **Resolver** is called by Composer (for tag inference) and Constructor (for value conversion)


### Reader

Handles input buffering and character encoding detection/conversion.

Info:
- File: internal/libyaml/reader.go (170 lines)
- Main Function: `func (parser *Parser) updateBuffer(length int) error`
- Input: `[]byte` or `io.Reader`
- Output: UTF-8 normalized bytes in `parser.buffer`
- Called From:
  * Scanner ([`scanner.go`](../../internal/libyaml/scanner.go) / `fetchNextToken()`)
- Important Processes:
  * `reader.go / determineEncoding       - Detects UTF-8/UTF-16 from BOM`
  * `reader.go / updateRawBuffer         - Reads more bytes from input source`

Transforms:
* **Encoding detection from BOM** (UTF-8, UTF-16LE, UTF-16BE)
* **UTF-16 to UTF-8 conversion** with surrogate pair handling
* **YAML 1.2 character set validation** (Tab, LF, CR, printable ASCII, BMP, supplementary planes)
* **UTF-8 sequence validation**
* **Input buffering** for lookahead

Notes:
* Encoding is stored in `parser.encoding` field
* The reader is not a separate stage but integrated into the Parser struct
* Character validation rules: allowed characters are Tab (0x09), LF (0x0A), CR (0x0D), printable ASCII (0x20-0x7E), BMP (U+0080-U+FFFD excluding surrogates), and supplementary planes (U+10000-U+10FFFF)


### Scanner

Lexical analysis - converts raw bytes into tokens.

Info:
- File: internal/libyaml/scanner.go (2599 lines)
- Main Function: `func (parser *Parser) Scan(token *Token) error`
- Input: UTF-8 bytes from reader buffer
- Output: `Token` struct
- Called From:
  * Parser ([`parser.go`](../../internal/libyaml/parser.go) / `peekToken()`)
- Important Processes:
  * `scanner.go / fetchMoreTokens        - Ensures token queue has enough tokens`
  * `scanner.go / fetchNextToken         - Dispatches to token fetchers by character`
  * `scanner.go / scanComments           - Classifies comments as head/foot/line`
  * `scanner.go / scanLineComment        - Captures same-line comments`
  * `reader.go / updateBuffer            - Refills input buffer (calls Reader)`

Transforms:
* **Character classification and dispatch** ([`fetchNextToken()`](../../internal/libyaml/scanner.go))
* **Flow level tracking** (`flow_level` field)
* **Indentation stack management** (`indent`, `indents` fields)
* **Simple key candidate tracking** (`simple_keys` stack)
* **Synthetic token generation** (BLOCK_SEQUENCE_START/BLOCK_MAPPING_START/BLOCK_END)
* **Escape sequence processing** in quoted scalars
* **Line folding** in folded block scalars
* **Chomping behavior** for block scalars (`-` strip, `+` keep)
* **URI decoding** in tags
* **Comment tokenization** via `scanLineComment()` and `scanComments()`
* **Comment classification** (head/foot/line) based on indentation and context

Notes:
* Scanner and Parser share the same `Parser` struct
* Tag handle and suffix remain separate at this stage (`Value` + `suffix` fields)
* Scalar style is recorded (Plain, SingleQuoted, DoubleQuoted, Literal, Folded)
* 2-token lookahead requirement for comment association (see [Comment Handling](#comment-handling-in-the-load-stack))
* Comment processing details covered in [Comment Handling](#comment-handling-in-the-load-stack) section
* Depth limits enforced: `max_flow_level` and `max_indents` both set to 10000 (see [Security Limits](#security-limits-and-protections))
* 23 token types defined:
  - `NO_TOKEN`
  - `STREAM_START_TOKEN`
  - `STREAM_END_TOKEN`
  - `VERSION_DIRECTIVE_TOKEN`
  - `TAG_DIRECTIVE_TOKEN`
  - `DOCUMENT_START_TOKEN`
  - `DOCUMENT_END_TOKEN`
  - `BLOCK_SEQUENCE_START_TOKEN`
  - `BLOCK_MAPPING_START_TOKEN`
  - `BLOCK_END_TOKEN`
  - `FLOW_SEQUENCE_START_TOKEN`
  - `FLOW_SEQUENCE_END_TOKEN`
  - `FLOW_MAPPING_START_TOKEN`
  - `FLOW_MAPPING_END_TOKEN`
  - `BLOCK_ENTRY_TOKEN`
  - `FLOW_ENTRY_TOKEN`
  - `KEY_TOKEN`
  - `VALUE_TOKEN`
  - `ALIAS_TOKEN`
  - `ANCHOR_TOKEN`
  - `TAG_TOKEN`
  - `SCALAR_TOKEN`
  - `COMMENT_TOKEN`


### Parser

Syntactic analysis - converts token stream into event stream.

Info:
- File: internal/libyaml/parser.go (1174 lines)
- Main Function: `func (parser *Parser) Parse(event *Event) error`
- Input: `Token` stream from Scanner (via internal `peekToken()`/`skipToken()`)
- Output: `Event` struct
- Called From:
  * Composer ([`composer.go / Composer.peek()`](../../internal/libyaml/composer.go))
  * Composer ([`composer.go / Composer.expect()`](../../internal/libyaml/composer.go))
- Important Processes:
  * `parser.go / stateMachine            - Dispatches to parser state handlers`
  * `parser.go / peekToken               - Looks at next token (calls Scanner)`
  * `parser.go / skipToken               - Consumes current token`
  * `scanner.go / Scan                   - Gets next token from Scanner (same struct)`
  * `parser.go / parseStreamStart        - Handles stream start event`
  * `parser.go / parseDocumentStart      - Handles document boundaries`
  * `parser.go / parseNode               - Parses scalar/sequence/mapping nodes`
  * `parser.go / UnfoldComments          - Joins comment lines to tokens`
  * `parser.go / setEventComments        - Transfers comments to Event`

Transforms:
* **LL(1) grammar production matching** (22 parser states):
  - `PARSE_STREAM_START_STATE`
  - `PARSE_IMPLICIT_DOCUMENT_START_STATE`
  - `PARSE_DOCUMENT_START_STATE`
  - `PARSE_DOCUMENT_CONTENT_STATE`
  - `PARSE_DOCUMENT_END_STATE`
  - `PARSE_BLOCK_NODE_STATE`
  - `PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE`
  - `PARSE_BLOCK_SEQUENCE_ENTRY_STATE`
  - `PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE`
  - `PARSE_BLOCK_MAPPING_FIRST_KEY_STATE`
  - `PARSE_BLOCK_MAPPING_KEY_STATE`
  - `PARSE_BLOCK_MAPPING_VALUE_STATE`
  - `PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE`
  - `PARSE_FLOW_SEQUENCE_ENTRY_STATE`
  - `PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE`
  - `PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE`
  - `PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE`
  - `PARSE_FLOW_MAPPING_FIRST_KEY_STATE`
  - `PARSE_FLOW_MAPPING_KEY_STATE`
  - `PARSE_FLOW_MAPPING_VALUE_STATE`
  - `PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE`
  - `PARSE_END_STATE`
* **Tag handle → full URI resolution** (`!!str` → `tag:yaml.org,2002:str`)
* **Token grouping** (anchor + tag + scalar tokens → single SCALAR_EVENT)
* **Comment attachment to events** via `UnfoldComments()` and `setEventComments()`
* **`TAIL_COMMENT_EVENT` generation** for block-end foot comments
* **`splitStemComment()` processing** for comments preceding nested structures
* **Implicit/quoted_implicit flag calculation**
* **Block structure tokens → hierarchical event pairs**

Notes:
* Tag handle/suffix split is lost here - only full URI remains
* The `Implicit` flag indicates whether tag was omitted in source
* Comment processing details covered in [Comment Handling](#comment-handling-in-the-load-stack) section


### Composer

Builds the Node tree from the event stream.

Info:
- File: internal/libyaml/composer.go (320 lines)
- Main Function: `func (c *Composer) Parse() *Node`
- Input: `Event` stream from owned Parser (via internal `peek()`/`expect()`)
- Output: `*Node` tree
- Called From:
  * Entry point [`yaml.go / unmarshal()`](../../yaml.go)
  * Entry point [`yaml.go / Decoder.Decode()`](../../yaml.go)
  * Entry point [`constructor.go / Construct()`](../../internal/libyaml/constructor.go)
- Important Processes:
  * `composer.go / peek                  - Peeks at next event type (calls Parser)`
  * `composer.go / expect                - Consumes event of expected type (calls Parser)`
  * `parser.go / Parser.Parse            - Gets next event from owned Parser`
  * `composer.go / node                  - Creates Node with tag/style`
  * `composer.go / scalar                - Builds ScalarNode from event`
  * `composer.go / mapping               - Builds MappingNode recursively`
  * `composer.go / sequence              - Builds SequenceNode recursively`
  * `composer.go / document              - Builds DocumentNode wrapper`
  * `resolver.go / resolve               - Infers tag for untagged scalars`

Transforms:
* **Event sequence → tree structure**
* **Tag short-form normalization** (`tag:yaml.org,2002:str` → `!!str`)
* **DEFAULT TYPE INFERENCE for untagged scalars** via `resolve("", value)` ⚠️
* **Anchor registration** in map (`c.anchors`)
* **Alias name → pointer to target Node**
* **Style flag conversion** (libyaml styles → Node Style flags)
* **Comment transfer** from Event to Node
* **Comment reassignment logic** in mappings (foot → key, tail → key)

Notes:
* Byte-level Index is lost here (only Line/Column preserved)
* Implicit document distinction is lost
* In the current implementation, Composer calls `resolve()` for untagged scalars - this may be the wrong place (see Problems section)
* Comment reassignment rules detailed in [Comment Handling](#comment-handling-in-the-load-stack) section


### Resolver

Resolves tags to determine the type of each scalar value.

Info:
- File: internal/libyaml/resolver.go (170 lines)
- Main Function: `func resolve(tag string, in string) (rtag string, out any)`
- Input: Tag string + scalar value
- Output: Resolved tag + typed Go value
- Called From:
  * Composer ([`composer.go`](../../internal/libyaml/composer.go) / `scalar()`)
  * Constructor ([`constructor.go`](../../internal/libyaml/constructor.go) / `scalar()`)
  * Representer ([`representer.go`](../../internal/libyaml/representer.go) / `stringv()`)
  * Serializer ([`serializer.go`](../../internal/libyaml/serializer.go) / `node()`)
- Important Processes:
  * `resolver.go / parseTimestamp        - Parses ISO 8601 timestamps`
  * `strconv / ParseInt                  - Parses signed integers`
  * `strconv / ParseUint                 - Parses unsigned integers`
  * `strconv / ParseFloat                - Parses floating point numbers`

Transforms:
* **Implicit tag resolution** based on scalar content (`true` → `!!bool`, `42` → `!!int`)
* **YAML 1.1 compatibility handling** (sexagesimal, old bools)
* **Timestamp parsing**
* **Special value recognition** (`.nan`, `.inf`, `null`)

Notes:
* In go-yaml, this is not a separate stage but a function called from multiple places
* Called from Composer (to set Node.Tag) and Constructor (to get typed value) ⚠️
* The YAML spec treats Resolver as a distinct stage producing the "Representation Graph"


### Constructor

Converts Node tree into Go values.

Info:
- File: internal/libyaml/constructor.go (1183 lines)
- Main Function: `func (c *Constructor) Construct(n *Node, out reflect.Value) bool`
- Input: `*Node` tree (received from Composer via entry point)
- Output: `reflect.Value` (Go values modified in place)
- Called From:
  * Entry point [`yaml.go / unmarshal()`](../../yaml.go)
  * Entry point [`yaml.go / Decoder.Decode()`](../../yaml.go)
  * Entry point [`node.go / Node.Decode()`](../../internal/libyaml/node.go)
- Important Processes:
  * `constructor.go / prepare            - Checks for Unmarshaler interface`
  * `constructor.go / scalar             - Converts ScalarNode to Go value`
  * `constructor.go / mapping            - Converts MappingNode to map/struct`
  * `constructor.go / sequence           - Converts SequenceNode to slice`
  * `constructor.go / document           - Unwraps DocumentNode`
  * `constructor.go / alias              - Follows alias pointer, reconstructs`
  * `resolver.go / resolve               - Re-resolves tag to get typed value`

Transforms:
* **Custom Unmarshaler interface detection and dispatch**
* **Duplicate key detection** (when `UniqueKeys` option enabled) - checks all mapping keys
* **Alias expansion ratio protection** (billion laughs defense) - limits constructs from alias expansion
* **Self-referential alias detection** - prevents infinite loops from aliases containing themselves
* **Re-resolution of tag/value** via `resolve(n.Tag, n.Value)` ⚠️
* **`indicatedString()` check** for quoted scalars (quoted/literal scalars skip `resolve()`)
* **Merge key (`<<`) handling** - explicit keys take precedence, can merge single mapping or sequence of mappings
* **Inline struct/map handling** (`,inline` tag) - one inline map per struct, string keys required, unknown keys go there
* **Known fields enforcement** (`WithKnownFields` option) - rejects unknown struct fields when enabled
* **Unhashable key error handling** - maps/slices as mapping keys trigger errors
* **TextUnmarshaler support** - for scalar types only
* **YAML 1.1 boolean compatibility** - `yes/no/on/off` recognized for typed bool targets
* **Type coercion** (YAML types → Go types)
* **Alias expansion** (full reconstruction each time)
* **Binary base64 decoding** (`!!binary` tag)
* **Struct field mapping** via `getStructInfo()`

Notes:
* `resolve()` is called again here - duplicates work done in Composer
* Aliases are fully reconstructed each time (not shared)
* Duplicate key detection (in [`mapping()`](../../internal/libyaml/constructor.go) function) compares all keys in mappings when enabled
* Merge key rules: explicit keys take precedence over merged keys, can merge a single mapping or a sequence of mappings
* Inline rules: only one inline map allowed per struct, requires string keys, unknown keys are stored in the inline map field
* Indicated strings (quoted or literal style) skip tag resolution via `indicatedString()` check
* Node tree, comments, style, anchors, positions are all discarded


## Comment Handling in the Load Stack

Comments flow through Scanner → Parser → Composer with classification, attachment,
and reassignment at each stage. The goal is to preserve comments and associate them
with the appropriate YAML nodes so they can be round-tripped or presented in the
Node tree.

See also: [Comment Flow Diagram](comment-flow.mmd)

### Comment Types

Comments are classified into several types based on their position relative to nodes:

| Type | Purpose | Example |
|------|---------|---------|
| HeadComment | Lines preceding a node (no blank line separation) | `# This is a comment\nkey: value` |
| LineComment | Same line as a node, after its value | `key: value  # inline comment` |
| FootComment | After a node, before any blank lines | `key: value\n# trailing comment` |
| TailComment | Internal: foot comment at end of block mapping value | Used during parsing only |
| stem_comment | Internal: comment on entry before nested structure | Used during parsing only |

### Scanner: Comment Tokenization

The scanner identifies and classifies comments as it processes tokens:

- **`scanLineComment()`** - Captures same-line comments (when no newlines have occurred since the last token)
- **`scanComments()`** - Main classifier that determines head/foot/line based on:
  - Indentation relative to `next_indent`
  - Flow context (all remaining comments become foot comments in `[...]` or `{...}`)
  - Empty line boundaries (blank lines separate head from foot)
- **2-token lookahead requirement** ([`scanLineComment()`](../../internal/libyaml/scanner.go)) - Scanner needs to peek ahead to properly associate comments
- **Special case**: Sequence entry line comments are transformed to head comments

Location: [`scanner.go`](../../internal/libyaml/scanner.go) / `scanComments()` and `scanLineComment()`

### Parser: Comment Attachment

The parser accumulates comments from tokens and attaches them to events:

- **`UnfoldComments()`** - Joins accumulated comment lines to tokens based on position
- **Parser comment fields** - `HeadComment`, `LineComment`, `FootComment` accumulate during token processing
- **`setEventComments()`** - Transfers parser comment fields to Event struct
- **`TAIL_COMMENT_EVENT`** - Special event type for foot comments at block ends (e.g., end of mapping value)
- **`splitStemComment()`** - Handles comments on entries preceding nested structures (splits into head comment for the nested item)
- **Document header splitting** - HeadComment is split at empty lines; content after blank goes to FootComment

Location: [`parser.go`](../../internal/libyaml/parser.go) / `UnfoldComments()`, `setEventComments()`, and `splitStemComment()`

### Composer: Comment Transfer and Reassignment

The composer transfers comments from events to nodes and applies reassignment logic:

- **Basic transfer**: Event.{Head,Line,Foot}Comment → Node.{Head,Line,Foot}Comment
- **Mapping reassignment rules** (in `mapping()` function):
  1. **Key FootComment reassignment for dedented comments** - If a comment is dedented (less indented than the key), it moves from key's FootComment to the mapping's FootComment
  2. **Value FootComment transfers to Key** - When the value has a FootComment but the key doesn't, the value's FootComment becomes the key's FootComment
  3. **TAIL_COMMENT_EVENT FootComment goes to Key** - Tail comments (from block-end events) are assigned to the key's FootComment
  4. **Final mapping FootComment moves to last key** - At the end of a mapping, if the mapping has a FootComment, it's moved to the last key's FootComment

Location: [`composer.go`](../../internal/libyaml/composer.go) / `mapping()` function

### Edge Cases

Several edge cases require special handling:

- **Sequence entry line-to-head transformation** - Line comments on sequence entries (`- item  # comment`) become head comments of the item
- **Block end token skip for head comments** - When BLOCK_END tokens have head comments, they're handled specially
- **Flow context closure** - When closing `]` or `}` in flow style, all remaining comments become foot comments
- **Document header splitting at empty lines** - Head comments with embedded blank lines are split (before blank stays head, after blank becomes foot)


## Security Limits and Protections

go-yaml includes several security features to prevent denial-of-service attacks.

### Depth Limits (Scanner)

The scanner enforces maximum nesting depth to prevent stack overflow:
- `max_flow_level = 10000` - Maximum nesting in flow style `[[[...]]]` or `{{{...}}}`
- `max_indents = 10000` - Maximum nesting via indentation (block style)

Location: [`scanner.go`](../../internal/libyaml/scanner.go) / flow level and indent tracking

### Alias Expansion Ratio (Constructor)

Prevents "billion laughs" style attacks via nested alias expansion:
- Documents under 400,000 constructs: allows up to 99% from alias expansion
- Documents over 4,000,000 constructs: allows only 10% from alias expansion
- Scales smoothly between thresholds

Error: `"document contains excessive aliasing"`

Location: [`constructor.go`](../../internal/libyaml/constructor.go) / `allowedAliasRatio()` function and expansion check in `sequence()`

### Self-Referential Alias Detection (Constructor)

Detects and prevents infinite loops from aliases referencing themselves:
- Tracks nodes being expanded in `c.aliases` map
- Error: `"anchor '%s' value contains itself"`

Location: [`constructor.go`](../../internal/libyaml/constructor.go) / `alias()` function


## Dump Stack

The dump stack transforms Go values (or Node trees) into YAML text.

**Control Flow:** The dump stack uses a push-based call hierarchy. Entry points
(such as `Marshal()`, `Encoder.Encode()`, or `Node.Encode()`) orchestrate the
process by creating and coordinating the stages:

- **Entry points** create a **Representer** (which owns an **Emitter**)
- **Entry points** call **Representer**.MarshalDoc() or similar methods
- **Representer** walks Go values and calls emit() to push Events to owned **Emitter**
- If input is a `*Node`, **Representer** delegates to **Serializer** (part of Representer)
- **Serializer** walks the Node tree and pushes Events to **Emitter** via emit()
- **Emitter** accumulates Events, formats output, and calls **Writer** to flush bytes
- **Resolver** is called by Representer and Serializer (to check if quoting/tags needed)


### Representer

Converts Go values directly to events (bypasses Node tree).

Info:
- File: internal/libyaml/representer.go (564 lines)
- Main Function: `func (r *Representer) MarshalDoc(tag string, in reflect.Value)`
- Input: `reflect.Value` + `Options`
- Output: Events pushed to owned Emitter
- Called From:
  * Entry point [`yaml.go / Marshal()`](../../yaml.go)
  * Entry point [`yaml.go / Encoder.Encode()`](../../yaml.go)
  * Entry point [`node.go / Node.Encode()`](../../internal/libyaml/node.go)
- Important Processes:
  * `representer.go / marshal            - Dispatches by Go type`
  * `representer.go / emit               - Sends event to owned Emitter`
  * `serializer.go / node                - Delegates Node to Serializer (same file scope)`
  * `representer.go / mapv               - Marshals map with sorted keys`
  * `representer.go / structv            - Marshals struct fields`
  * `representer.go / slicev             - Marshals slice/array`
  * `representer.go / stringv            - Marshals string with style choice`
  * `emitter.go / Emitter.Emit           - Emits events (owned Emitter)`
  * `resolver.go / resolve               - Checks if quoting needed`

Transforms:
* **Go type dispatch** ([`marshal()`](../../internal/libyaml/representer.go) type switch)
* **Custom Marshaler interface detection and dispatch**
* **TextMarshaler support** - detects and calls TextMarshaler interface methods
* **Struct field ordering and filtering** (exported, by tag, omitempty)
* **Map key sorting** (natural sort with numeric awareness) - ensures deterministic output
* **Resolve-check for quoting decisions** via `resolve()` ⚠️
* **YAML 1.1 compatibility checks** (`isBase60Float()`, `isOldBool()`)
* **Style selection** (literal for multiline strings)
* **Binary data base64 encoding** - non-UTF-8 strings automatically tagged `!!binary` and base64 encoded
* **Flow style from struct tags**

Notes:
* Representer calls `resolve()` to determine if quoting is needed - this couples dump to load logic
* When input is `*Node`, delegates to Serializer instead
* Map key sorting ensures deterministic output by using natural sort with numeric awareness


### Serializer

Converts Node tree to events (used when marshaling from Node).

Info:
- File: internal/libyaml/serializer.go (192 lines)
- Main Function: `func (r *Representer) node(node *Node, tail string)`
- Input: `*Node` tree
- Output: Events pushed to Emitter (via Representer)
- Called From:
  * Representer ([`representer.go / Representer.nodev()`](../../internal/libyaml/representer.go))
  * Representer ([`representer.go / marshal()`](../../internal/libyaml/representer.go))
- Important Processes:
  * `representer.go / emit               - Sends event to Emitter (via Representer)`
  * `representer.go / nilv               - Emits null scalar`
  * `serializer.go / node (recursive)    - Walks child nodes`
  * `serializer.go / isSimpleCollection  - Checks if flow style appropriate`
  * `emitter.go / Emitter.Emit           - Emits events (via Representer)`
  * `resolver.go / resolve               - Checks if tag can be elided`

Transforms:
* **Node tree → event stream**
* **Tag elision check** via `resolve()` ⚠️
* **Force quoting** when tag would be misresolved
* **Flow style detection for simple collections** (`WithFlowSimpleCollections` option) - automatically uses flow style for eligible collections
* **Comment placement/shifting** (foot → tail)
* **Style flag interpretation**
* **Invalid UTF-8 → base64**

Notes:
* `resolve()` is called to check if tag can be elided - fourth place it's called
* Simple collection = all scalar children, fits within line width


### Emitter

Converts events to YAML text output.

Info:
- File: internal/libyaml/emitter.go (2075 lines)
- Main Function: `func (emitter *Emitter) Emit(event *Event) error`
- Input: `Event` stream (queued in `emitter.events`)
- Output: UTF-8 bytes to `emitter.buffer`
- Called From:
  * Representer ([`representer.go / Representer.must()`](../../internal/libyaml/representer.go))
  * Representer ([`representer.go / Representer.emit()`](../../internal/libyaml/representer.go))
- Important Processes:
  * `emitter.go / needMoreEvents         - Checks if more events needed for lookahead`
  * `emitter.go / analyzeEvent           - Analyzes scalar/tag for style decisions`
  * `emitter.go / stateMachine           - Dispatches to emitter state handlers`
  * `emitter.go / selectScalarStyle      - Final style selection (can override)`
  * `emitter.go / writeScalar            - Writes scalar with chosen style`
  * `writer.go / flush                   - Flushes buffer to output (calls Writer)`

Transforms:
* **Event accumulation** for lookahead decisions
* **Final style selection** (`selectScalarStyle()` can override earlier choices)
* **Simple key eligibility check** (length ≤128, no multiline)
* **Block vs flow style** (context-dependent)
* **Scalar analysis** for style validity (`analyzeScalar()`)
* **Line wrapping** at `best_width`
* **Indentation management**
* **Escape sequence encoding**
* **Tag directive shortening**

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
- Called From:
  * Emitter ([`emitter.go / put()`](../../internal/libyaml/emitter.go))
- Important Processes:
  * `write_handler callback              - Configured output destination`

Transforms:
* **Buffer flush** to output destination

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

Location: [`yaml.go`](../../internal/libyaml/yaml.go) / `Token` struct definition

Notes:
* Tag handle and suffix are still separate here
* Full byte-level position (Index) is available
* For the complete list of 23 token types, see [Scanner Notes](#scanner)


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

Location: [`yaml.go`](../../internal/libyaml/yaml.go) / `Event` struct definition

Notes:
* Tag is now full URI - handle/suffix split is lost
* Comments are attached
* 11 event types defined:
  - `STREAM_START_EVENT`
  - `STREAM_END_EVENT`
  - `DOCUMENT_START_EVENT`
  - `DOCUMENT_END_EVENT`
  - `ALIAS_EVENT`
  - `SCALAR_EVENT`
  - `SEQUENCE_START_EVENT`
  - `SEQUENCE_END_EVENT`
  - `MAPPING_START_EVENT`
  - `MAPPING_END_EVENT`
  - `TAIL_COMMENT_EVENT`


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

Location: [`node.go`](../../internal/libyaml/node.go) / `Node` struct definition

Notes:
* Tag is short form - full URI is lost
* Byte Index is lost - only Line/Column remain
* 6 node kinds:
  - `DocumentNode`
  - `SequenceNode`
  - `MappingNode`
  - `ScalarNode`
  - `AliasNode`
  - `StreamNode`
* 6 style flags:
  - `TaggedStyle`
  - `DoubleQuotedStyle`
  - `SingleQuotedStyle`
  - `LiteralStyle`
  - `FoldedStyle`
  - `FlowStyle`
* `indicatedString()` method determines if a scalar skips tag resolution (quoted or literal style)
* `shouldUseLiteralStyle()` heuristic for multi-line strings decides between literal block style and quoted style


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
| [`composer.go`](../../internal/libyaml/composer.go) / `scalar()` | Load | Set Node.Tag for untagged scalars |
| [`constructor.go`](../../internal/libyaml/constructor.go) / `scalar()` | Load | Get actual Go value |
| [`representer.go`](../../internal/libyaml/representer.go) / `stringv()` | Dump | Check if quoting needed |
| [`serializer.go`](../../internal/libyaml/serializer.go) / `node()` | Dump | Check if tag can be elided |

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


### 7. v4 vs Legacy Default Differences

Different entry points use different default settings, which can cause inconsistent behavior:

| Setting | v4 Defaults | Legacy Defaults |
|---------|-------------|-----------------|
| Indent | 2 | 4 |
| CompactSeqIndent | true | false |
| LineWidth | 80 | -1 (unlimited) |

`Load/Dump` use v4 defaults; `Marshal/Unmarshal` use Legacy defaults.

Location: [`options.go`](../../internal/libyaml/options.go) / `Options` struct v4 defaults and `LegacyOptions` variable

**Problem:** Users may see different formatting depending on which API they use, even with identical data.


## Anything Else I Missed?

Areas that may need deeper investigation:
- Anchor/alias reconstruction behavior
- Error propagation and position reporting
- The relationship between root package types and internal/libyaml types


## Glossary

### A

**Alias** - A YAML reference to a previously defined anchor, allowing reuse of content. Represented with `*name` syntax in YAML. In go-yaml, AliasNode points to the target Node.

**Anchor** - A named marker (`&name`) that identifies a YAML node for later reference via aliases. Stored as a string in Node/Event/Token structs.

### B

**Billion Laughs Attack** - A denial-of-service attack using nested aliases to cause exponential expansion. go-yaml protects against this with alias expansion ratio limits.

**Block Style** - YAML's indentation-based syntax for collections and scalars. Examples: indented mappings/sequences, literal blocks (`|`), folded blocks (`>`).

**BOM (Byte Order Mark)** - A special Unicode character at the start of a file indicating encoding. go-yaml detects UTF-8, UTF-16LE, and UTF-16BE from BOM.

### C

**Chomping** - Controls how trailing newlines are handled in block scalars. Strip (`-`) removes them, keep (`+`) preserves them, clip (default) keeps one newline.

**Composer** - Load stack stage that builds Node trees from Event streams. Handles tag normalization, anchor registration, and comment transfer.

**Constructor** - Load stack stage that converts Node trees to native Go values. Performs type coercion, handles custom Unmarshalers, and enforces constraints.

### D

**Document** - A single YAML data structure within a stream. Can be explicit (starts with `---`) or implicit. Represented as DocumentNode in go-yaml.

**Dump Stack** - The pipeline that transforms Go values into YAML text: Representer → Serializer → Emitter → Writer.

### E

**Emitter** - Dump stack stage that converts Events to UTF-8 YAML text. Handles final style selection, line wrapping, and indentation.

**Event** - Parser output representing syntactic structure. 11 types including SCALAR_EVENT, MAPPING_START_EVENT, SEQUENCE_END_EVENT. Contains tag, value, comments, and position.

### F

**Flow Style** - YAML's compact JSON-like syntax using brackets `[]` and braces `{}`. Example: `{key: value}` or `[a, b, c]`.

**Folded Block Scalar** - Multi-line string (`>`) where newlines are converted to spaces, except for blank lines and more-indented lines.

**FootComment** - A comment appearing after a node but before any blank lines. Can be reassigned to keys in mappings.

### H

**HeadComment** - Comments appearing before a node with no blank line separation.

### I

**Implicit Tag** - A tag inferred by the resolver based on scalar content rather than explicitly specified. Example: `42` → `!!int`.

**Indicated String** - A scalar that has quoted or literal style, indicating it should be treated as a string regardless of content. Skips tag resolution.

**Indentless Sequence** - A YAML sequence where entries are indicated by `-` but not further indented relative to the parent. Used in some mapping values.

**Inline Struct** - A struct field tagged with `,inline` that merges its fields into the parent struct during marshaling/unmarshaling.

### L

**Literal Block Scalar** - Multi-line string (`|`) where newlines are preserved exactly as written.

**Load Stack** - The pipeline that transforms YAML text into Go values: Reader → Scanner → Parser → Composer → Resolver → Constructor.

### M

**Mapping** - YAML's key-value structure (like a map or dictionary). Represented as MappingNode in go-yaml.

**Marshaling** - The process of converting Go values to YAML text (Dump stack).

**Merge Key** - Special YAML key (`<<`) that merges content from another mapping or sequence of mappings. Explicit keys take precedence.

### N

**Node** - Tree-based representation of YAML structure. 6 kinds: DocumentNode, SequenceNode, MappingNode, ScalarNode, AliasNode, StreamNode.

### P

**Parser** - Load stack stage that converts Tokens to Events using LL(1) grammar. Implements 22 parser states and handles comment attachment.

**Pull-Based** - Architecture where higher stages request data from lower stages. Used in Load stack (Constructor pulls from Composer, which pulls from Parser, etc.).

**Push-Based** - Architecture where lower stages push data to higher stages. Used in Dump stack (Representer pushes Events to Emitter).

### R

**Reader** - Load stack component that handles encoding detection, UTF-8 conversion, and input buffering. Not a separate stage but integrated into Parser struct.

**Representer** - Dump stack stage that converts Go values to Events. Handles type dispatch, field filtering, key sorting, and style selection.

**Representation Graph** - In YAML spec, the data structure after tag resolution. In go-yaml, conceptually the Node tree with resolved tags.

**Resolver** - Function (not a separate stage) that infers tags from scalar content. Called from multiple places in both Load and Dump stacks.

### S

**Scalar** - YAML's atomic value type (string, number, boolean, null). Represented as ScalarNode with a style (plain, quoted, literal, folded).

**Scanner** - Load stack stage performing lexical analysis. Converts UTF-8 bytes to Tokens, handling indentation tracking, flow level tracking, and comment tokenization.

**Self-Referential Alias** - An alias that references itself directly or indirectly, causing infinite loops. go-yaml detects and prevents this.

**Sequence** - YAML's ordered list structure (like an array). Represented as SequenceNode in go-yaml.

**Serializer** - Dump stack stage that converts Node trees to Events. Handles tag elision checks and flow style detection.

**Simple Collection** - A sequence or mapping containing only scalar children that fits within line width. Eligible for automatic flow style.

**Simple Key** - A mapping key that is short enough (≤128 characters) and single-line. Flow mappings require simple keys.

**Stream** - Top-level container for YAML documents. A stream can contain multiple documents separated by `---`. Represented as StreamNode in go-yaml.

**Surrogate Pair** - UTF-16 encoding mechanism for characters outside the Basic Multilingual Plane. go-yaml handles these during UTF-16 to UTF-8 conversion.

### T

**Tag** - Type indicator for YAML nodes. Short form (`!!str`, `!!int`) or full URI (`tag:yaml.org,2002:str`). Can be explicit or implicit.

**Tag Directive** - YAML directive (`%TAG`) that defines a short handle for tag URIs. Example: `%TAG ! tag:yaml.org,2002:`.

**Tag Elision** - Omitting an explicit tag when it can be inferred. Serializer checks if tags can be elided during dump.

**Tag Handle** - Short prefix for tags (like `!!` or `!custom!`). Stored separately from suffix in Token but merged in Event.

**TAIL_COMMENT_EVENT** - Special event type for foot comments at block structure ends. Used during parsing to properly assign comments to mapping keys.

**TextMarshaler/TextUnmarshaler** - Go interfaces for custom text encoding/decoding. Supported by Representer (marshaling) and Constructor (unmarshaling, scalars only).

**Token** - Scanner output representing lexical units. 23 types including SCALAR_TOKEN, BLOCK_MAPPING_START_TOKEN, TAG_TOKEN. Contains byte-level position (Index).

### U

**Unmarshaling** - The process of converting YAML text to Go values (Load stack).

**UniqueKeys** - Option that enables duplicate key detection in mappings. Enabled by default in v2, v3, and v4.

### V

**Version Directive** - YAML directive (`%YAML 1.2`) specifying the YAML version. Stored in StreamNode.

### W

**Writer** - Dump stack component that flushes output buffer to destination. Very simple - just calls configured write handler.
