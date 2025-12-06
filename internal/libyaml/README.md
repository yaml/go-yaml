# internal/libyaml

This package provides low-level YAML processing functionality through a 3-stage
pipeline: Scanner → Parser → Emitter.
It implements the libyaml C library functionality in Go.

## Directory Overview

The `internal/libyaml` package implements the core YAML processing stages:

1. **Scanner** - Tokenizes YAML text into tokens
2. **Parser** - Converts tokens into events following YAML grammar rules
3. **Emitter** - Serializes events back into YAML text

## File Organization

### Main Source Files

- **scanner.go** - YAML scanner/tokenizer implementation
- **parser.go** - YAML parser (tokens → events)
- **emitter.go** - YAML emitter (events → YAML output)
- **api.go** - Public API for Parser and Emitter types
- **yaml.go** - Core types and constants (Event, Token, enums)
- **reader.go** - Input handling and encoding detection
- **writer.go** - Output handling
- **yamlprivate.go** - Internal types and helper functions

### Test Files

- **scanner_test.go** - Scanner tests
- **parser_test.go** - Parser tests
- **emitter_test.go** - Emitter tests
- **api_test.go** - API tests
- **yaml_test.go** - Utility function tests
- **reader_test.go** - Reader tests
- **writer_test.go** - Writer tests
- **yamlprivate_test.go** - Character classification tests
- **yaml_data_test.go** - YAML test data loading framework

### Test Data Files (in `test-data/`)

- **scanner_test.yaml** - Scanner test cases
- **parser_test.yaml** - Parser test cases
- **emitter_test.yaml** - Emitter test cases
- **api_test.yaml** - API test cases
- **yaml_test.yaml** - Utility function test cases
- **reader_test.yaml** - Reader test cases
- **writer_test.yaml** - Writer test cases
- **yamlprivate_test.yaml** - Character classification test cases

## Processing Pipeline

### 1. Scanner (scanner.go)

The scanner converts YAML text into tokens.

**Input**: Raw YAML text (string or []byte)
**Output**: Stream of tokens

**Token types include**:
- `SCALAR_TOKEN` - Plain, quoted, or block scalar values
- `KEY_TOKEN`, `VALUE_TOKEN` - Mapping key/value indicators
- `BLOCK_MAPPING_START_TOKEN`, `FLOW_MAPPING_START_TOKEN` - Mapping delimiters
- `BLOCK_SEQUENCE_START_TOKEN`, `FLOW_SEQUENCE_START_TOKEN` - Sequence delimiters
- `ANCHOR_TOKEN`, `ALIAS_TOKEN` - Anchor definitions and references
- `TAG_TOKEN` - Type tags
- `DOCUMENT_START_TOKEN`, `DOCUMENT_END_TOKEN` - Document boundaries

**Responsibilities**:
- Character encoding detection (UTF-8, UTF-16LE, UTF-16BE)
- Line break normalization
- Indentation tracking
- Quote and escape sequence handling

### 2. Parser (parser.go)

The parser converts tokens into events following YAML grammar rules.

**Input**: Stream of tokens from Scanner
**Output**: Stream of events

**Event types include**:
- `STREAM_START_EVENT`, `STREAM_END_EVENT` - Stream boundaries
- `DOCUMENT_START_EVENT`, `DOCUMENT_END_EVENT` - Document boundaries
- `SCALAR_EVENT` - Scalar values
- `MAPPING_START_EVENT`, `MAPPING_END_EVENT` - Mapping boundaries
- `SEQUENCE_START_EVENT`, `SEQUENCE_END_EVENT` - Sequence boundaries
- `ALIAS_EVENT` - Anchor references

**Responsibilities**:
- Implementing YAML grammar and validation
- Managing document directives (%YAML, %TAG)
- Resolving anchors and aliases
- Tracking implicit vs explicit markers
- Style preservation (plain, single-quoted, double-quoted, literal, folded)

### 3. Emitter (emitter.go)

The emitter converts events back into YAML text.

**Input**: Stream of events
**Output**: YAML text

**Responsibilities**:
- Style selection (plain/quoted scalars, block/flow collections)
- Formatting control (canonical mode, indentation, line width)
- Character encoding
- Anchor and tag serialization
- Document marker generation (---, ...)

**Configuration options**:
- `Canonical` - Emit in canonical YAML form
- `Indent` - Indentation width (2-9 spaces)
- `Width` - Line width (-1 for unlimited)
- `Unicode` - Enable Unicode character output
- `LineBreak` - Line break style (LN, CR, CRLN)

## Testing Framework

### Test Architecture

The testing framework uses a data-driven approach:

1. **Test data** is stored in YAML files in the `test-data/` directory
2. **Test logic** is implemented in Go files (`*_test.go`)
3. **One-to-one pairing**: Each `test-data/foo_test.yaml` has a corresponding `foo_test.go`

**Benefits**:
- Easy to add new test cases without writing Go code
- Test data is human-readable and self-documenting
- Test logic is reusable across many test cases
- Test data is separated from test code for clarity
- Tests can become a common suite for multiple YAML frameworks

### Test Data Files

Each YAML file contains test cases for a specific component:

- **scanner_test.yaml** - Scanner/tokenization tests
  - Token sequence verification
  - Token property validation (value, style)
  - Error detection

- **parser_test.yaml** - Parser/event generation tests
  - Event sequence verification
  - Event property validation (anchor, tag, value, directives)
  - Error detection

- **emitter_test.yaml** - Emitter/serialization tests
  - Event-to-YAML conversion
  - Configuration options testing
  - Roundtrip testing (parse → emit)
  - Writer integration

- **api_test.yaml** - API constructor and method tests
  - Constructor validation
  - Method behavior and state changes
  - Panic conditions
  - Cleanup verification

- **yaml_test.yaml** - Utility function tests
  - Enum String() methods
  - Style accessor methods

- **reader_test.yaml** - Reader/input handling tests
  - Encoding detection (UTF-8, UTF-16LE, UTF-16BE)
  - Buffer management
  - Error handling

- **writer_test.yaml** - Writer/output handling tests
  - Buffer flushing
  - Output handlers (string, io.Writer)
  - Error conditions

- **yamlprivate_test.yaml** - Character classification tests
  - Character type predicates (isAlpha, isDigit, isHex, etc.)
  - Character conversion functions (asDigit, asHex, width)
  - Unicode handling

### Test Framework Implementation

The test framework is implemented in `yaml_data_test.go`:

**Core functions**:
- `LoadTestCases(filename string) []TestCase` - Loads and parses test YAML files

**Core types**:
- `TestCase` struct - Umbrella structure containing fields for all test types
  - Uses `interface{}` for flexible field types
  - Post-processing converts generic fields to specific types

**Post-processing**:
After loading, the framework processes test data:
- Converts `Want` (interface{}) to `WantEvents`, `WantTokens`, or `WantSpecs` based on test type
- Converts `Find` (interface{}) to `WantContains` (handles both scalar and sequence)
- Converts `Checks` to field validation specifications

### Test Types

#### Scanner Tests

**scan-tokens** - Verify token sequence

```yaml
- name: Simple scalar
  type: scan-tokens
  yaml: |-
    hello
  want:
  - STREAM_START_TOKEN
  - SCALAR_TOKEN
  - STREAM_END_TOKEN
```

**scan-tokens-detailed** - Verify token properties

```yaml
- name: Single quoted scalar
  type: scan-tokens-detailed
  yaml: |-
    'hello world'
  detailed: true
  want:
  - type: STREAM_START_TOKEN
  - type: SCALAR_TOKEN
    style: SINGLE_QUOTED_SCALAR_STYLE
    value: hello world
  - type: STREAM_END_TOKEN
```

**scan-error** - Verify error detection

```yaml
- name: Invalid character
  type: scan-error
  yaml: "\x01"
```

#### Parser Tests

**parse-events** - Verify event sequence

```yaml
- name: Simple mapping
  type: parse-events
  yaml: |
    key: value
  want:
  - STREAM_START_EVENT
  - DOCUMENT_START_EVENT
  - MAPPING_START_EVENT
  - SCALAR_EVENT
  - SCALAR_EVENT
  - MAPPING_END_EVENT
  - DOCUMENT_END_EVENT
  - STREAM_END_EVENT
```

**parse-events-detailed** - Verify event properties

```yaml
- name: Anchor and alias
  type: parse-events-detailed
  yaml: |
    - &anchor value
    - *anchor
  detailed: true
  want:
  - type: STREAM_START_EVENT
  - type: DOCUMENT_START_EVENT
  - type: SEQUENCE_START_EVENT
  - type: SCALAR_EVENT
    anchor: anchor
    value: value
  - type: ALIAS_EVENT
    anchor: anchor
  - type: SEQUENCE_END_EVENT
  - type: DOCUMENT_END_EVENT
  - type: STREAM_END_EVENT
```

**parse-error** - Verify error detection

```yaml
- name: Error state
  type: parse-error
  yaml: |
    key: : invalid
```

#### Emitter Tests

**emit** - Emit events and verify output contains expected strings

```yaml
- name: Simple scalar
  type: emit
  events:
  - type: STREAM_START_EVENT
    encoding: UTF8_ENCODING
  - type: DOCUMENT_START_EVENT
    implicit: true
  - type: SCALAR_EVENT
    value: hello
    implicit: true
    style: PLAIN_SCALAR_STYLE
  - type: DOCUMENT_END_EVENT
    implicit: true
  - type: STREAM_END_EVENT
  want: hello
```

**emit-config** - Emit with configuration

```yaml
- name: Custom indent
  type: emit-config
  config:
    indent: 4
  events:
  - type: STREAM_START_EVENT
    encoding: UTF8_ENCODING
  - type: DOCUMENT_START_EVENT
    implicit: true
  - type: MAPPING_START_EVENT
    implicit: true
    style: BLOCK_MAPPING_STYLE
  # ... more events
  want: key
```

**roundtrip** - Parse → emit, verify output

```yaml
- name: Roundtrip
  type: roundtrip
  yaml: |
    key: value
    list:
      - item1
      - item2
  want:
  - key
  - value
  - item1
```

**emit-writer** - Emit to io.Writer

```yaml
- name: Writer
  type: emit-writer
  events:
  - type: STREAM_START_EVENT
    encoding: UTF8_ENCODING
  # ... more events
  want: test
```

#### API Tests

**api-new** - Test constructors

```yaml
- name: New parser
  type: api-new
  using: NewParser
  test:
    raw-buffer:
      nil: false
      cap: 512
    buffer:
      nil: false
      cap: 1536
```

**api-method** - Test methods and field state

```yaml
- name: Parser set input string
  type: api-method
  using: NewParser
  bytes: true
  call: [SetInputString, 'key: value']
  test:
    input: {==: 'key: value'}
    input-pos: {==: 0}
    read-handler: {nil: false}
```

**api-panic** - Test methods that should panic

```yaml
- name: Parser set input string twice
  type: api-panic
  using: NewParser
  bytes: true
  setup: [SetInputString, first]
  call: [SetInputString, second]
  want: must set the input source only once
```

**api-delete** - Test cleanup

```yaml
- name: Parser delete
  type: api-delete
  using: NewParser
  bytes: true
  setup: [SetInputString, test]
  test:
    input: {len: 0}
    buffer: {len: 0}
```

**api-new-event** - Test event constructors

```yaml
- name: New stream start event
  type: api-new-event
  call: [NewStreamStartEvent, UTF8_ENCODING]
  test:
    Type: {==: STREAM_START_EVENT}
    encoding: {==: UTF8_ENCODING}
```

#### Utility Tests

**enum-string** - Test String() methods of enums

```yaml
- name: Scalar style plain
  type: enum-string
  enum: ScalarStyle
  value: PLAIN_SCALAR_STYLE
  want: Plain
```

**style-accessor** - Test style accessor methods

```yaml
- name: Event scalar style
  type: style-accessor
  method: ScalarStyle
  style: DOUBLE_QUOTED_SCALAR_STYLE
```

### Common Keys in Test YAML Files

- **name** - Test case name (title case convention)
- **type** - Test type (determines which test handler to use)
- **yaml** - Input YAML string to test
- **want** - Expected result (format varies by test type)
  - For api-panic: string containing expected panic message substring
  - For scan-error/parse-error: boolean (defaults to true if omitted; set to false if no error expected)
  - For enum-string: string representing expected String() output
  - For other types: varies (may be sequence or scalar)
- **events** - For emitter tests: list of event specifications to emit
- **config** - For emitter config tests: emitter configuration options
- **using** - For API tests: constructor name (NewParser, NewEmitter)
- **call** - For API tests: method call [MethodName, arg1, arg2, ...]
- **setup** - For API panic tests: setup method call before main method
- **bytes** - For API tests: boolean flag to convert string args to []byte
- **test** - For API tests: field validation specifications
- **enum** - For enum tests: enum type to test
- **value** - For enum tests: enum value to test
- **method** - For style accessor tests: accessor method name
- **style** - For style accessor tests: style value to test

### Running Tests

```bash
# Run all tests in the package
go test ./internal/libyaml

# Run specific test file
go test ./internal/libyaml -run TestScanner
go test ./internal/libyaml -run TestParser
go test ./internal/libyaml -run TestEmitter
go test ./internal/libyaml -run TestAPI
go test ./internal/libyaml -run TestYAML

# Run specific test case (using subtest name)
go test ./internal/libyaml -run TestScanner/Block_sequence
go test ./internal/libyaml -run TestParser/Anchor_and_alias
go test ./internal/libyaml -run TestEmitter/Flow_mapping

# Run with verbose output
go test -v ./internal/libyaml

# Run with coverage
go test -cover ./internal/libyaml
```
