# Reference parser plugin for go-yaml

This optional Go module adapts
[`yamlstar-plugin-parser-reference`](https://github.com/yamlstar/yamlstar-plugin-parser-reference)
to `yaml.ParserPlugin`.

```go
import reference "go.yaml.in/yaml/v4/plugin/parser/reference"

err := yaml.Load(input, &value, yaml.WithPlugin(reference.New()))
```

Call `reference.Register()` before using `yaml.OptsYAML` with
`plugin: {parser: reference@v0.2.5}`.
