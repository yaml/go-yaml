# Migrating from v3 to v4

This guide will help you migrate your code from `go.yaml.in/yaml/v3`
(or `gopkg.in/yaml.v3`) to `go.yaml.in/yaml/v4`.

## Quick Migration Checklist

- [ ] Update import path
- [ ] Optionally migrate to new API (Load/Dump, Loader/Dumper)
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

## Compatibility with go-yaml

When migrating from [go-yaml](https://github.com/go-yaml/yaml/) ensure that
YAML module imports are updated in all dependent projects transitively, because
unmarshaller interface types `gopkg.in/yaml.v{version}.Unmarshaler` and
`go.yaml.in/yaml/v{version}.Unmarshaler` are different types and thus are
incompatible. Custom marshallers would not be called if executed with the
parser from the different library.

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

All v3 APIs continue to work in v4. The classic API remains supported
for simple use cases:

- `Unmarshal()` - Classic API (or use `Load()` for more flexibility)
- `Marshal()` - Classic API (or use `Dump()` for more flexibility)
- `NewDecoder()` - Classic API (or use `NewLoader()` for more flexibility)
- `NewEncoder()` - Classic API (or use `NewDumper()` for more flexibility)

You can migrate incrementally:
1. Update import path to v4
2. Verify tests pass
3. Optionally migrate to new API for additional features
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
4. Test with v4 defaults or choose formatting explicitly

### Strategy 3: Feature Adoption (Maximum Value)

1. Follow Strategy 2
2. Explore new functional options
3. Leverage structured error information
4. Adopt version presets for different use cases

## Testing Your Migration

```bash
# Run your existing tests
go test ./...

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

### Issue: Want more flexibility from classic API

**Solution:** Migrate to the new API for options support:
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
