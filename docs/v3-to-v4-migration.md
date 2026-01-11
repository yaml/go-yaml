# Migrating from v3 to v4

This guide will help you migrate your code from `go.yaml.in/yaml/v3`
(or `gopkg.in/yaml.v3`) to `go.yaml.in/yaml/v4`.

## Quick Migration Checklist

- [ ] Update import path
- [ ] Replace deprecated API calls
- [ ] Handle TypeError.Errors type change (if you use it directly)
- [ ] Adjust formatting expectations or use yaml.V3 preset
- [ ] Update tests

## Import Path Change

**v3:**
```go
import "gopkg.in/yaml.v3"
// or
import "go.yaml.in/yaml/v3"
```

**v4:**
```go
import "go.yaml.in/yaml/v4"
```

Update all import statements throughout your codebase.

## API Changes

### Recommended: Use New API

v4 introduces a cleaner API with better naming.

#### Loading YAML

**v3:**
```go
err := yaml.Unmarshal(data, &config)
```

**v4:**
```go
err := yaml.Load(data, &config)
```

#### Dumping YAML

**v3:**
```go
data, err := yaml.Marshal(&config)
```

**v4:**
```go
data, err := yaml.Dump(&config)
```

#### Streaming Decoding

**v3:**
```go
decoder := yaml.NewDecoder(reader)
err := decoder.Decode(&config)
```

**v4:**
```go
loader := yaml.NewLoader(reader)
err := loader.Load(&config)
```

#### Streaming Encoding

**v3:**
```go
encoder := yaml.NewEncoder(writer)
err := encoder.Encode(&config)
encoder.Close()
```

**v4:**
```go
dumper := yaml.NewDumper(writer)
err := dumper.Dump(&config)
dumper.Close()
```

## Breaking Changes

### TypeError.Errors Field Type

This is the **only breaking change** in v4.

**v3:**
```go
type TypeError struct {
    Errors []string  // Simple error strings
}
```

**v4:**
```go
type TypeError struct {
    Errors []*UnmarshalError  // Structured errors with location info
}

type UnmarshalError struct {
    Line    int
    Column  int
    Problem string
    // ... other fields
}
```

**Migration:**

If you directly access `TypeError.Errors` expecting `[]string`,
you'll need to update your code:

```go
// v3 code
if typeErr, ok := err.(*yaml.TypeError); ok {
    for _, errStr := range typeErr.Errors {
        fmt.Println(errStr)
    }
}

// v4 code
if typeErr, ok := err.(*yaml.TypeError); ok {
    for _, unmarshalErr := range typeErr.Errors {
        fmt.Printf("Line %d, Column %d: %s\n",
            unmarshalErr.Line, unmarshalErr.Column, unmarshalErr.Problem)
    }
}
```

## New Features in v4

### Functional Options

v4 introduces a functional options pattern for configuration:

```go
// Version presets
yaml.Dump(&data, yaml.V2)  // Use v2 defaults
yaml.Dump(&data, yaml.V3)  // Use v3 defaults
yaml.Dump(&data, yaml.V4)  // Use v4 defaults (2-space, compact)

// Custom options
yaml.Dump(&data,
    yaml.WithIndent(4),
    yaml.WithCompactSeqIndent(false),
    yaml.WithLineWidth(100),
)

// Combine presets with overrides
yaml.Dump(&data, yaml.V3, yaml.WithIndent(2))

// Loading options
yaml.Load(data, &config,
    yaml.WithKnownFields(),   // Strict field checking
    yaml.WithUniqueKeys(),    // Enforce unique keys
)
```

Available dump options:
- `WithIndent(n)` - Set indentation spaces
- `WithCompactSeqIndent(bool)` - Compact sequence indentation
- `WithLineWidth(n)` - Maximum line width
- `WithUnicode(bool)` - Use Unicode characters
- `WithCanonical(bool)` - Canonical output format
- `WithLineBreak(lb)` - Line break style (LN, CR, CRLN)
- `WithExplicitStart(bool)` - Add `---` document start
- `WithExplicitEnd(bool)` - Add `...` document end
- `WithFlowSimpleCollections(bool)` - Use flow style for simple
  collections

Available load options:
- `WithKnownFields()` - Reject unknown struct fields
- `WithUniqueKeys()` - Enforce unique mapping keys
- `WithSingleDocument()` - Expect only one document

### Options from YAML

You can configure options using YAML:

```go
optsYAML := `
indent: 4
compact-seq-indent: false
line-width: 100
`

opts, err := yaml.OptsYAML(optsYAML)
if err != nil {
    log.Fatal(err)
}

data, err := yaml.Dump(&config, opts)
```

## Formatting Differences

v4 has different default formatting than v3:

| Aspect | v3 Default | v4 Default |
|--------|-----------|-----------|
| Indentation | 4 spaces | 2 spaces |
| Sequence style | Normal | Compact |

**Example:**

```yaml
# v3 default output
items:
    - name: foo
      value: 1
    - name: bar
      value: 2

# v4 default output
items:
- name: foo
  value: 1
- name: bar
  value: 2
```

### Preserving v3 Behavior

If you need v3's formatting, use the `yaml.V3` preset:

```go
// Get v3-style formatting in v4
data, err := yaml.Dump(&config, yaml.V3)
```

Or customize individual options:

```go
data, err := yaml.Dump(&config,
    yaml.WithIndent(4),
    yaml.WithCompactSeqIndent(false),
)
```

## Backward Compatibility

All v3 APIs continue to work in v4 but are **deprecated** for removal
in v5:

- `Unmarshal()` → Use `Load()`
- `Marshal()` → Use `Dump()`
- `NewDecoder()` → Use `NewLoader()`
- `NewEncoder()` → Use `NewDumper()`
- `Decoder.Decode()` → Use `Loader.Load()`
- `Encoder.Encode()` → Use `Dumper.Dump()`

You can migrate incrementally:
1. Update import path to v4
2. Verify tests pass
3. Gradually replace deprecated APIs
4. Update TypeError.Errors handling if applicable

## Migration Strategies

### Strategy 1: Minimal Change (Fastest)

1. Update import path
2. If using TypeError.Errors directly, update that code
3. Add `yaml.V3` preset to maintain v3 formatting
4. Done!

```go
// Only change needed for basic migration
data, err := yaml.Dump(&config, yaml.V3)
```

### Strategy 2: Adopt New API (Recommended)

1. Update import path
2. Replace `Unmarshal` → `Load`, `Marshal` → `Dump`
3. Replace `NewDecoder` → `NewLoader`, `NewEncoder` → `NewDumper`
4. Update TypeError.Errors handling
5. Test with v4 defaults or choose formatting explicitly

### Strategy 3: Feature Adoption (Maximum Value)

1. Follow Strategy 2
2. Explore new functional options
3. Leverage structured error information
4. Adopt version presets for different use cases

## Testing Your Migration

```bash
# Run your existing tests
go test ./...

# Check for deprecation warnings (Go 1.18+)
go test -v ./... 2>&1 | grep -i 'deprecat.*'

# Verify YAML output formatting
# Use the go-yaml CLI tool to compare
go install go.yaml.in/yaml/v4/cmd/go-yaml@latest
./go-yaml -n < testfile.yaml
```

## Common Issues

### Issue: Output formatting changed

**Solution:** Use `yaml.V3` preset to maintain v3 formatting:
```go
yaml.Dump(&data, yaml.V3)
```

### Issue: TypeError.Errors type mismatch

**Solution:** Update code to handle `[]*UnmarshalError`:
```go
for _, e := range typeErr.Errors {
    fmt.Println(e.Problem)  // Or use e.Line, e.Column
}
```

### Issue: Deprecation warnings

**Solution:** Replace deprecated APIs:
- `Unmarshal` → `Load`
- `Marshal` → `Dump`
- `NewDecoder` → `NewLoader`
- `NewEncoder` → `NewDumper`

## Getting Help

If you encounter issues during migration:

- Check the [API documentation](https://pkg.go.dev/go.yaml.in/yaml/v4)
- Browse [examples](example/)
- Open an [issue](https://github.com/yaml/go-yaml/issues)
- Ask in [Slack](https://cloud-native.slack.com/archives/C08PPAT8PS7)

## Next Steps

- Explore the new [functional options](#functional-options)
- Review the [examples](example/) directory
- Read the [full API documentation](
  https://pkg.go.dev/go.yaml.in/yaml/v4)
- Try the [go-yaml CLI tool](README.md#the-go-yaml-cli-tool) for
  debugging
