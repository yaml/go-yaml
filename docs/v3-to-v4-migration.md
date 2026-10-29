# Migrating from v3 to v4

This guide will help you migrate your code from `go.yaml.in/yaml/v3`
(or `gopkg.in/yaml.v3`) to `go.yaml.in/yaml/v4`.

## Quick Migration Checklist

- [ ] Update import path
- [ ] Optionally migrate to new API (Load/Dump, Loader/Dumper)
- [ ] Review formatting defaults and choose compatibility options as needed
- [ ] Check error handling and custom unmarshaler signatures
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

When migrating from [go-yaml/yaml](https://github.com/go-yaml/yaml/) ensure
that YAML module imports are updated in all dependent projects transitively,
because unmarshaller interface types `gopkg.in/yaml.v{version}.Unmarshaler` and
`go.yaml.in/yaml/v{version}.Unmarshaler` are different types and thus are
incompatible. Custom unmarshallers would not be called if executed with the
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
loader, err := yaml.NewLoader(reader)
if err != nil {
    return err
}
err = loader.Load(&config)
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
dumper, err := yaml.NewDumper(writer)
if err != nil {
    return err
}
if err := dumper.Dump(&config); err != nil {
    return err
}
return dumper.Close()
```

## New Features in v4

### Stream Nodes

v4 adds `StreamNode`, a new node kind that exposes stream-level
metadata not available in v3: encoding, `%YAML` version directives,
and `%TAG` directives.
This is accessed via `Node.Stream`, which is non-nil only on
`StreamNode` nodes.

Enable stream nodes with `WithStreamNodes()`:

```go
loader, err := yaml.NewLoader(reader, yaml.WithStreamNodes())
if err != nil {
    return err
}
for {
    var node yaml.Node
    err := loader.Load(&node)
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    if node.Kind == yaml.StreamNode && node.Stream != nil {
        // Inspect encoding, version, and tag directives.
        fmt.Println(node.Stream.Encoding)
        fmt.Println(node.Stream.Version)       // *yaml.VersionDirective
        fmt.Println(node.Stream.TagDirectives) // []yaml.TagDirective
    }
}
```

With stream nodes enabled, the loader emits nodes in the pattern
`[Stream, Doc, Stream, Doc, ..., Stream]`: one opening stream node
per document boundary, plus a final closing stream node.

### Functional Options

v4 introduces a functional options pattern for configuration:

```go
// Version presets
yaml.Dump(&data, yaml.WithV2Defaults())  // Use v2 defaults
yaml.Dump(&data, yaml.WithV3Defaults())  // Use v3 defaults
yaml.Dump(&data, yaml.WithV4Defaults())  // Use v4 defaults (2-space, compact)

// Custom options
yaml.Dump(&data,
    yaml.WithIndent(4),
    yaml.WithCompactSeqIndent(false),
    yaml.WithLineWidth(100),
)

// Combine presets with overrides
yaml.Dump(&data, yaml.WithV3Defaults(), yaml.WithIndent(2))

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
- `WithSingleDocument()` - Only load the first document, ignoring later ones

Package-level `Load` rejects additional documents by default.
Adding `WithSingleDocument()` skips that check and leaves later documents
unread and unvalidated.

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

Use `yaml.WithV3Defaults()` to select v3 defaults.
When replacing `Marshal`, also disable wrapping to match its line-width
setting:

```go
// Match the classic Marshal formatting settings.
data, err := yaml.Dump(&config,
    yaml.WithV3Defaults(),
    yaml.WithLineWidth(-1),
)
```

Or customize individual options:

```go
data, err := yaml.Dump(&config,
    yaml.WithIndent(4),
    yaml.WithCompactSeqIndent(false),
)
```

## Backward Compatibility

The classic entry points remain supported in v4:

- `Unmarshal()` - Classic API (or use `Load()` for more flexibility)
- `Marshal()` - Classic API (or use `Dump()` for more flexibility)
- `NewDecoder()` - Classic API (or use `NewLoader()` for more flexibility)
- `NewEncoder()` - Classic API (or use `NewDumper()` for more flexibility)

You can migrate incrementally:

1. Update import path to v4
2. Verify tests pass
3. Optionally migrate to new API for additional features
4. Review error handling and custom unmarshaler types

Presets select compatibility defaults, including error formatting.
They do not undo every v4 bug fix or reproduce every legacy code path.
For example, `Unmarshal` retains legacy trailing-content handling beyond the
settings selected by `WithV3Defaults()`.
`NewEncoder` uses v3 output settings with the v4 error formatter.
For v3 error presentation through the new API, select `WithV3Defaults()`.

### Conversion errors

Conversion failures now return `*yaml.LoadErrors`.
Use `errors.As` rather than a direct type assertion when looking for legacy
`*yaml.TypeError` values:

```go
var typeErr *yaml.TypeError
if errors.As(err, &typeErr) {
    for _, message := range typeErr.Errors {
        fmt.Println(message)
    }
}
```

`TypeError.Errors` remains `[]string`.
For structured details, use `errors.As` with `*yaml.LoadErrors` instead.

### Custom unmarshalers across versions

The `UnmarshalYAML(*yaml.Node) error` hook depends on the exact package's
`Node` type.
A v3 node-based hook does not satisfy the v4 `Unmarshaler` interface.
V4 detects hooks using `gopkg.in/yaml.v3.Node` or
`go.yaml.in/yaml/v3.Node` and returns a migration error instead of invoking
them or silently decoding the struct without its hook.
This applies to classic APIs and version presets as well as `Load`.
For v4-specific implementations, use `go.yaml.in/yaml/v4.Node`.

The legacy callback signature `UnmarshalYAML(func(any) error) error` is still
recognized and can be used across v2, v3, and v4.

## Migration Strategies

### Strategy 1: Minimal Change (Fastest)

1. Update import path
2. Keep the classic entry points initially
3. Review errors and custom unmarshaler types
4. Run your application's tests, including output comparisons

```go
// Keep these calls while changing the package import to v4.
err := yaml.Unmarshal(input, &config)
data, err := yaml.Marshal(&config)
```

### Strategy 2: Adopt New API (Recommended)

1. Update import path
2. Replace `Unmarshal` → `Load`, `Marshal` → `Dump`
3. Replace `NewDecoder` → `NewLoader`, `NewEncoder` → `NewDumper`
4. Handle constructor errors before using loaders and dumpers
5. Test with v4 defaults or choose formatting explicitly

### Strategy 3: Feature Adoption (Maximum Value)

1. Follow Strategy 2
2. Explore new functional options
3. Use structured error information
4. Adopt version presets for different use cases

## Testing Your Migration

```bash
# Run your existing tests
go test ./...

# Verify YAML output formatting
# Use the go-yaml CLI tool to compare
go install go.yaml.in/yaml/v4/cmd/go-yaml@latest
go-yaml -n testfile.yaml
```

## Common Issues

### Issue: Output formatting changed

**Solution:** Select v3 defaults and, when matching `Marshal`, disable wrapping:
```go
output, err := yaml.Dump(&data,
    yaml.WithV3Defaults(),
    yaml.WithLineWidth(-1),
)
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
- Browse [examples](../example/README.md)
- Open an [issue](https://github.com/yaml/go-yaml/issues)
- Ask in [Slack](https://cloud-native.slack.com/archives/C08PPAT8PS7)

## Next Steps

- Explore the new [functional options](#functional-options)
- Review the [examples](../example/README.md) directory
- Read the [full API documentation](
  https://pkg.go.dev/go.yaml.in/yaml/v4)
- Try the [go-yaml CLI tool](../cmd/go-yaml/README.md) for
  debugging
