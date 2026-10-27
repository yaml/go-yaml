# Load alias data with go-yaml

## CLI

From the repository root, build the CLI and run the example:

```bash
make cli
./go-yaml -C example/function-tag/options.yaml \
  -y example/function-tag/input.yaml
```

The output is:

```yaml
all:
  bar: 42
  hash2:
    inherited: true
    key: base
foo: 42
inherited: true
map1:
  inherited: true
  key: value
```

[options.yaml](options.yaml) enables Alias-Data path lookups and registers
`load` from the `alias-data` function library.
[input.yaml](input.yaml) loads [vars.yaml](vars.yaml) and uses its data:

| Input | Behavior |
|---|---|
| `!load vars: vars.yaml` | Load the file into alias `vars`; omit this entry |
| `all: *vars` | Copy the entire imported value |
| `foo: *vars.bar` | Look up a mapping key |
| `inherited: *vars.hash2.inherited` | Look up a nested mapping key |
| `<<: *vars.hash2` | Merge the imported mapping into `map1` |
| `key: value` | Override the merged `key: base` entry |

Relative import filenames resolve beside the input file, so the command can
run from any directory.
For stdin input, relative imports use the working directory.
The imported file is plain YAML data; its function tags do not execute.

For JSON output, run:

```bash
./go-yaml -C example/function-tag/options.yaml \
  -J example/function-tag/input.yaml
```

## Go API

From the repository root, run [main.go](main.go):

```bash
cd example/function-tag
go run main.go
```

Or use the repository's Go toolchain from the repository root:

```bash
make shell CMD='cd example/function-tag && go run .'
```

The program configures both plugins directly in Go with `aliasdata.New`,
`functiontag.New`, and `yaml.WithPlugin`.
`aliasdata.WithPath()` enables dotted alias lookups.
The explicit binding `functiontag.Functions{"load": aliasdata.Load}` assigns
the `!load` tag to Alias-Data's import function.
The optional second argument to `functiontag.New` supplies named libraries;
this example needs only the direct binding.
Function-Tag itself has no built-in `load` function.
It reads `input.yaml`, loads the data with `yaml.Load`, and prints the same
YAML output shown above with `yaml.Dump`.
Run it from the example directory so the Go API can find `vars.yaml` relative
to the working directory.
