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

These targets create a workspace under `.cache/` using the current core
checkout.
No related checkout or global workspace is required.

## Release dependency

The plugin must be released with a dependency on a core release exporting
`yaml.V4MigrationPlugin` and supporting it in `yaml.WithPlugin`.
The development workspace supplies that API from this checkout.
Before publishing this module, update its core requirement from the existing
release-candidate baseline to the first released version containing the API.
The plugin can ship before or after v4.0.0; bridge removal is a separate change.
