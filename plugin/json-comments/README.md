# JSON-comments plugin for go-yaml

This separate Go module implements `yaml.EventSourcePlugin` using the generated
parser in `github.com/yamlstar/yamlstar-plugin-json-comments/parser`.
It requires Go 1.24 or newer and works with `CGO_ENABLED=0`.

```go
import (
    "go.yaml.in/yaml/v4"
    jsoncomments "go.yaml.in/yaml/v4/plugin/json-comments"
)

var value any
err := yaml.Load(input, &value, yaml.WithPlugin(jsoncomments.New()))
```

Input is buffered and must be UTF-8.
Comments are discarded and node positions are unknown.
The reference parser determines syntax; go-yaml resolves and constructs values.
Depth checks happen after reference parsing; alias checks remain active during
construction.

For named YAML configuration, call `jsoncomments.Register()` once at startup.
Registration does not enable the plugin until selected.

From the go-yaml repository root, use the sample under
`example/json-comments`:

```bash
make test-json-comments
make test-json-comments-race
make cli CONFIG=example/json-comments/options.yaml
./go-yaml -j example/json-comments/data.yaml
```

The build resolves the newest tagged Go release of
`github.com/yamlstar/yamlstar-plugin-json-comments` each time.
To select a specific release, use:

```yaml
plugin:
  json-comments:
    name: json-comments
    version: 0.1.8
```

The version selects code while building and remains in the embedded runtime
configuration so the binary can verify the linked release.
The binary must be rebuilt to change the linked version.
`version: v0.1.8` is also accepted.
For local development, place the plugin checkout under
`repos/yamlstar-plugin-json-comments` and set `JSON-COMMENTS-LOCAL=1`.
This override validates the configured version against the local manifest.
The temporary CLI module and workspace live under `.cache/`; downloaded
module sources use the Go module cache.
The CLI target produces `./go-yaml`; `make cli` restores the ordinary build.
See [plugin documentation](../../docs/plugins.md) for the event interface,
configuration, supported CLI modes, and limitations.

Concurrent callers are supported.
The generated parser shares mutable state, so the parser package serializes
Go and EDN parsing calls with a common lock.

To build with JSON-comments enabled by default, put `json-comments` under
`plugin` in an options file and run `make cli CONFIG=FILE`.
The binary embeds that configuration, so no `-C` or `--plugin` flag is needed.
Runtime `-C` replaces the embedded configuration; `-o` overrides its options.
