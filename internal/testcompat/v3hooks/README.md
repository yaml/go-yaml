# V3 hook rejection tests

This test-only module uses both real v3 packages without adding them to the
core module's dependency graph.
Run it from the repository root:

```bash
make test-v3-hooks
```

The target creates a workspace under `.cache/` using the current checkout.
The data cases cover each legacy package across all loading APIs.
