# AI Assistant Guidelines for go-yaml

## Project Context

This is the **go-yaml** library (`go.yaml.in/yaml/v4`), a critical Go YAML
implementation used in **Kubernetes** and many other production systems.

**Critical Context:**
- Maintainer: Ingy döt Net (YAML spec lead maintainer)
- Code is scrutinized by experienced Go developers
- AI-generated code must meet professional Go standards
- Repeated mistakes will annoy the review team

**Package-specific guidelines:**
- See `internal/libyaml/.agents.md` for that package's patterns

## Go Coding Standards for This Project

### Code Formatting

```bash
# Always format code with gofumpt (configured in .golangci.yaml)
make fmt

# Run linters before committing
make lint

# Run tests
make test
```

**Formatting rules:**
- Use `gofumpt` formatter (stricter than `gofmt`)
- Try hard to keep lines to **80 characters maximum**
- Start doc comments and multi-line comments on new lines

### Linter Configuration

From `.golangci.yaml`:
- **Enabled**: `dupword`, `govet`, `misspell`, `nolintlint`, `staticcheck`
- **Disabled**: `errcheck`, `ineffassign`, `unused` (for now)
- **Note**: ST1003 (underscores in identifiers) is disabled because the
  codebase currently has many underscores

### Testing Patterns

#### Use testdata/ for Test Fixtures

```go
// CORRECT: Use testdata/ directory (Go build tool ignores this)
testFile := "testdata/scanner.yaml"

// WRONG: Don't scatter test data in arbitrary locations
testFile := "test_files/scanner.yaml"  // ❌
```

#### Use Assertion Helpers

```go
import "go.yaml.in/yaml/v4/internal/testutil/assert"

// CORRECT: Use assertion helpers for clear error messages
assert.Truef(t, ok, "scanTokens() failed")
assert.Equalf(t, expected, actual, "token[%d] mismatch", i)

// WRONG: Don't use raw if statements
if !ok {  // ❌
    t.Errorf("scanTokens() failed")
}
```

#### Use t.Helper() in Test Helpers

```go
// CORRECT: Mark test helper functions with t.Helper()
func runScanTokensTest(t *testing.T, tc TestCase) {
    t.Helper()  // This makes error messages point to the caller
    // ... test logic
}

// WRONG: Forgetting t.Helper() makes debugging harder
func runScanTokensTest(t *testing.T, tc TestCase) {  // ❌
    // ... test logic (errors will point here, not to caller)
}
```

### Type Assertions and Error Handling

```go
// CORRECT: Always check type assertions
wantSlice, ok := tc.Want.([]interface{})
assert.Truef(t, ok, "Want should be []interface{}, got %T", tc.Want)

// WRONG: Unchecked type assertion can panic
wantSlice := tc.Want.([]interface{})  // ❌ Can panic
```

### Documentation Standards

**Critical Rules from PR #194:**

1. **Never reference fields that don't exist**
   ```markdown
   ❌ WRONG: "The Find field contains expected results"
   ✅ RIGHT: "The Want field contains expected results"
   ```

2. **Verify examples match actual structs**
   ```yaml
   # ❌ WRONG: Including fields that don't exist
   - test-type:
       name: Example
       detailed: true  # This field doesn't exist!

   # ✅ RIGHT: Only use actual fields
   - test-type:
       name: Example
       yaml: "test"
   ```

3. **Check field names against code before writing docs**
   - Read the actual struct definition
   - Don't invent fields or guess names
   - If unsure, check the code first

### Naming Conventions

#### File Naming

```
scanner_test.go       ✅ Test files use _test.go suffix
yamldatatest_test.go  ✅ Underscores in multi-word names
testdata/             ✅ Go convention for test fixtures
```

#### Variable and Function Naming

```go
// Use camelCase for unexported names
func parseTokenType(t *testing.T, s string) TokenType

// Use PascalCase for exported names
func NewParser() Parser
func (p *Parser) SetInputString(input []byte)
```

### String Comparison Against External Data

**Critical Rule from PR #194:**

When comparing strings against values from YAML files or other external data,
**the Go code must use the EXACT format from the data file**.

```go
// Example: Test types in YAML files use hyphens
// testdata/emitter.yaml contains:
//   - emit-config:
//       name: Test

// ❌ WRONG: Using underscores when data uses hyphens
if tc.Type == "emit_config" {
    // This will never match!
}

// ✅ RIGHT: Match the exact format from YAML
if tc.Type == "emit-config" {
    // Matches the YAML data
}
```

**Common pitfall areas:**
- Test type names: `emit-config`, `scan-tokens-detailed` (use hyphens)
- Style constants: Check the actual constant name in code
- Error messages: Match exact text from implementation

### Error Handling

```go
// Return errors, don't panic in library code
func LoadYAML(data []byte) (interface{}, error) {
    if err := validate(data); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    // ...
}

// Tests can use assertions
func TestSomething(t *testing.T) {
    result, err := LoadYAML(data)
    assert.NoErrorf(t, err, "LoadYAML failed")
}

// Panic only for programmer errors (like bad constructor calls)
func createObject(t *testing.T, constructor string) interface{} {
    switch constructor {
    case "NewParser":
        return &Parser{}
    default:
        t.Fatalf("unknown constructor: %s", constructor)  // Test code can fail
    }
    return nil
}
```

## Commit Conventions

From `CONTRIBUTING.md`:

**Commit message format:**
- Start with capital letter
- Do NOT end with period
- Maximum 50 characters for subject
- No merge commits

```
✅ Add support for c1 control characters
✅ Fix block scalar hint handling
❌ add support for c1 control characters  (no capital)
❌ Fix block scalar hint handling.        (has period)
```

## Common AI Mistakes to Avoid

These issues were caught in PR #194 review:

### 1. Naming Mismatches

```yaml
# YAML data file uses hyphens
- emit-config:
    name: Test
```

```go
// ❌ WRONG: Go code uses underscores
if tc.Type == "emit_config" {
```

```go
// ✅ RIGHT: Go code matches YAML exactly
if tc.Type == "emit-config" {
```

### 2. Documentation Field Name Errors

```go
// Actual struct
type TestCase struct {
    Want interface{} `yaml:"want"`
}
```

```markdown
❌ WRONG in README:
The `Find` field contains the expected result.

✅ RIGHT in README:
The `Want` field contains the expected result.
```

### 3. Invalid Example Fields

```markdown
❌ WRONG in README:
- scan-tokens-detailed:
    name: Example
    detailed: true  # This field doesn't exist!

✅ RIGHT in README:
- scan-tokens-detailed:
    name: Example
    yaml: "test"
```

### 4. Inconsistent Naming Within Domain

```markdown
❌ WRONG: Mixing hyphens and underscores
scan-tokens-detailed  ✅
scan-tokens_detailed  ❌

✅ RIGHT: Consistent hyphens
scan-tokens
scan-tokens-detailed
parse-events
parse-events-detailed
```

## Before Submitting Code

**Checklist:**
1. ✅ Run `make fmt` (gofumpt formatting)
2. ✅ Run `make lint` (golangci-lint)
3. ✅ Run `make test` (all tests pass)
4. ✅ Verify string comparisons match data file formats exactly
5. ✅ Check documentation references actual field names
6. ✅ Confirm examples use only valid fields
7. ✅ Use `testdata/` for test fixtures
8. ✅ Use assertion helpers (`assert.Truef`, etc.)
9. ✅ Add `t.Helper()` to test helper functions

## Getting Help

- Read package-specific `.agents.md` files
- Check `CONTRIBUTING.md` for workflow
- Review existing code for patterns
- When uncertain, ask the user
