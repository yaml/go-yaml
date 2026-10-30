# V3 hook rejection tests

This test-only module lives under `testdata` so ordinary package discovery
excludes it.
It uses both real v3 packages without adding them to the core module's
dependency graph.
Its local `replace` directive always selects the current core checkout.

Run it from the repository root:

```bash
make test-v3-hooks
```

Or run directly from this directory with project dependencies available:

```bash
GOWORK=off go test ./...
```

The Make target also disables workspace mode, so an ambient `go.work` cannot
change the tested modules.
The data cases cover each legacy package across all loading APIs.
