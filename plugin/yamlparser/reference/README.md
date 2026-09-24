# Reference YAML-parser plugin for go-yaml

This optional Go module adapts
[`yamlstar-plugin-parser-reference`](https://github.com/yamlstar/yamlstar-plugin-parser-reference)
to `yaml.ParserPlugin`.

```go
import reference "go.yaml.in/yaml/v4/plugin/yamlparser/reference"

err := yaml.Load(input, &value, yaml.WithPlugin(reference.New()))
```

Call `reference.Register()` before using `yaml.OptsYAML` with
`plugin: {yaml-parser: reference@v0.2.5}`.

The API name is `yaml-parser` and the implementation name is `reference`.
This implementation is optional and is not selected by a bare `yaml-parser`
selector, which continues to use the built-in `go-yaml` implementation.

From the repository root:

```bash
make test-reference-parser
make cli PLUGIN=yaml-parser=reference@v0.2.5
./go-yaml --plugin=yaml-parser=reference@0.2.5 -j data.yaml
```

For local development, use the checkout under
`repos/yamlstar-plugin-parser-reference` by setting
`REFERENCE-PARSER-LOCAL=1`.

See the [plugin documentation](../../../docs/plugins.md) for configuration and
CLI build details.
