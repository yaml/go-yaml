# Optional v4 migration module

See the [V4-Migration guide](../../docs/plugin/v4-migration.md) for adapter
selection, classic-style wrappers, and conversion behavior.

This module depends on both v3 packages.
The core module has no v3 dependency.

Run checks from the repository root:

```bash
make test-v4-migration lint-v4-migration
make test-v4-migration opts=-race
```

These targets run directly in this module with `GOWORK=off`.
The local `replace` directive selects the current core checkout, including
when running tests directly from this directory:

```bash
GOWORK=off go test ./...
```

No workspace or related checkout is required.

## Release dependency

The plugin must be released with a dependency on a core release exporting
`yaml.V4MigrationPlugin` and supporting it in `yaml.WithPlugin`.
The local replacement supplies that API from this checkout during development.
Before publishing this module, set its core requirement to a released version
containing the API rather than relying on the v4.0.0 development baseline.
The plugin can ship before or after v4.0.0; bridge removal is a separate change.
