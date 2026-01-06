# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](
https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Plugin system for extensibility (coming soon)
- GitHub Discussions for community Q&A
- Comprehensive migration guide (MIGRATION.md)
- This CHANGELOG file

### Changed
- Project maintenance transferred to official YAML organization
- Improved documentation with modern structure and badges

## [4.0.0] - TBD

### Added
- New `Load()` function as replacement for `Unmarshal()`
- New `Dump()` function as replacement for `Marshal()`
- New `Loader` type as replacement for `Decoder`
- New `Dumper` type as replacement for `Encoder`
- Functional options system for configuring YAML processing
- Version presets: `yaml.V2`, `yaml.V3`, `yaml.V4`
- `OptsYAML()` for configuring options via YAML
- Structured `UnmarshalError` type with line and column information
- `StreamNode` support for processing YAML streams as node trees
- Options API with `WithIndent()`, `WithKnownFields()`,
  `WithUniqueKeys()`, etc.
- 17 working examples in the `example/` directory
- `go-yaml` CLI tool for debugging and understanding YAML processing
- Support for both YAML 1.1 and 1.2 features

### Changed
- Default indentation changed from 4 spaces to 2 spaces
- Default sequence style changed to compact
  (list items at same indentation level)
- `TypeError.Errors` field now returns `[]*UnmarshalError` instead of
  `[]string` (BREAKING)
- Import path: `go.yaml.in/yaml/v4`
  (was `gopkg.in/yaml.v3` or `go.yaml.in/yaml/v3`)

### Deprecated
- `Unmarshal()` - use `Load()` instead (will be removed in v5)
- `Marshal()` - use `Dump()` instead (will be removed in v5)
- `NewDecoder()` - use `NewLoader()` instead (will be removed in v5)
- `NewEncoder()` - use `NewDumper()` instead (will be removed in v5)
- `Decoder.Decode()` - use `Loader.Load()` instead
  (will be removed in v5)
- `Encoder.Encode()` - use `Dumper.Dump()` instead
  (will be removed in v5)
- `Encoder.SetIndent()` - use `WithIndent()` option instead
- `Encoder.CompactSeqIndent()` - use `WithCompactSeqIndent(true)` option
  instead
- `Encoder.DefaultSeqIndent()` - use `WithCompactSeqIndent(false)`
  option instead
- `Decoder.KnownFields()` - use `WithKnownFields()` option instead

### Fixed
- Improved error messages with precise location information
- Better handling of edge cases in YAML 1.1/1.2 compatibility

### Security
- Regular security audits and dependency updates
- Frozen legacy versions (v1-v3) receive security fixes only

## [3.0.0] - 2020-03

This is the baseline version when the YAML organization took over
maintenance from the original go-yaml/yaml project.

### Highlights from v3
- Basic `Unmarshal`/`Marshal` API
- `NewDecoder`/`NewEncoder` streaming API
- Node-based YAML manipulation
- Struct tag support (`yaml:"field,omitempty,flow,inline"`)
- YAML 1.2 support with YAML 1.1 compatibility
- Based on pure Go port of libyaml

## Earlier Versions

For the complete history of versions prior to the YAML organization's
maintenance, please refer to the original project at
https://github.com/go-yaml/yaml.

---

## Version Support Policy

- **v4**: Active development - receives all new features and bug fixes
- **v3**: Frozen legacy - security fixes only
- **v2**: Frozen legacy - security fixes only
- **v1**: Frozen legacy - security fixes only

Users are encouraged to migrate to v4 for the best experience and
latest features.
