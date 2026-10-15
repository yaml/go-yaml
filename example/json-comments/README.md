# JSON-comments plugin example

This example shows how to use the JSON-comments plugin from Go code and from
the `go-yaml` command.
Both forms use version 0.1.9 of
`github.com/yamlstar/yamlstar-plugin-json-comments`.

## Go program

`main.go` registers the sanitizer implementation, selects version `v0.1.9`
through `yaml.OptsYAML`, loads `data.yaml`, and writes the value as JSON.
The registry checks the requested version against the linked implementation.

Run it from the repository root:

```bash
make shell CMD='go -C example/json-comments run .'
```

It prints the sample data as one line of JSON.

This example is a separate Go module because the optional JSON-comments plugin
has dependencies that are not part of the core go-yaml module.
Its `go.mod` pins the released sanitizer module and uses local replacements for
this go-yaml checkout and its JSON-comments adapter.

## go-yaml command

`options.yaml` selects the `sanitizer` implementation of the JSON-comments
plugin API and requires version 0.1.9.
The version chooses the Go module at build time and is retained so the binary
can validate its linked release at runtime.
`data.yaml` contains line comments, block comments, and comment markers that
must remain part of scalar values.

Build the configured command and parse the sample:

```bash
make cli CONFIG=example/json-comments/options.yaml
./go-yaml -j example/json-comments/data.yaml
```

The plugin selector DSL can build the same optional implementation without
embedding the rest of `options.yaml`:

```bash
make cli PLUGIN=json-comments@v0.1.9
./go-yaml --plugin=json-comments -j example/json-comments/data.yaml
```

The resulting `go-yaml` binary embeds the options and links the selected
plugin version.
The options file is not needed when running that binary.
It can also be supplied through `-C` because its version matches the binary:

```bash
./go-yaml -C example/json-comments/options.yaml \
  -j example/json-comments/data.yaml
```

Run `make cli` without `CONFIG` to restore the ordinary command.
