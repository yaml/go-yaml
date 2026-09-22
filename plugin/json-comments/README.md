# JSON-comments plugin for go-yaml

This separate Go module implements `yaml.JSONCommentsPlugin` with the sanitizer
from `github.com/yamlstar/yamlstar-plugin-json-comments`.
It requires Go 1.24 or newer and works with `CGO_ENABLED=0`.
It does not use a shared library.

```go
import (
    "go.yaml.in/yaml/v4"
    jsoncomments "go.yaml.in/yaml/v4/plugin/json-comments"
)

var value any
err := yaml.Load(input, &value, yaml.WithPlugin(jsoncomments.New()))
```

The plugin buffers UTF-8 input, removes recognized JSON-style comments, and
passes the sanitized text to the selected YAML parser.
Line endings and non-comment text are preserved.
The parser still supplies node positions, styles, tags, anchors, and aliases.
Concurrent calls are supported.

Call `jsoncomments.Register()` once to use named configuration:

```yaml
plugin:
  json-comments: sanitizer@v0.1.9
```

The API name is `json-comments` and the implementation name is `sanitizer`.
A bare `json-comments` selector uses this default implementation.

From the repository root:

```bash
make test-json-comments
make test-json-comments-race
make cli PLUGIN=json-comments
./go-yaml --plugin=json-comments -j example/json-comments/data.yaml
```

A build without an explicit version resolves the newest tagged release.
Use `json-comments@v0.1.9` or
`json-comments=sanitizer@v0.1.9` to select a specific release.
For local development, use the checkout under
`repos/yamlstar-plugin-json-comments` by setting `JSON-COMMENTS-LOCAL=1`.

See the [plugin documentation](../../docs/plugins.md) for configuration and
CLI build details.
See the upstream [syntax rules] for recognized comment forms.

[syntax rules]: https://github.com/yamlstar/yamlstar-plugin-json-comments/blob/main/Syntax.md
