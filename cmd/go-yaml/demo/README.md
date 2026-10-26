# Plugin demos

Run these demos from a fresh go-yaml clone without any `repos/*` checkouts.
Each demo prints its number, description, command, output, and result.
Expected rejections count as passes.

From the repository root:

```bash
./cmd/go-yaml/demo/demo-plugins.sh
./cmd/go-yaml/demo/demo-plugins.sh --list
./cmd/go-yaml/demo/demo-plugins.sh 3 5-7 9
./cmd/go-yaml/demo/demo-plugins.sh -i 10-
```

You can also run `./demo-plugins.sh` from this directory.
Numbers follow the catalog order, starting at 1.
Selections run in that order; `5-7` includes both endpoints, and `10-` runs
demo 10 through the end.
`--list` shows commands without building or running the CLI.
Non-interactive output is piped through `less -FRX`.

## Interactive controls

With `-i`, readline displays each command for editing before execution:

- Press Enter to run the displayed command.
- Clear the line with Ctrl+U and press Enter to recall the last command run.
- Clear the line and enter a demo number to load that demo's command.
- After a recalled or numbered command runs, the pending demo returns.
- Clear the line and press Ctrl+D to quit, or press Ctrl+C to interrupt.

## Builds and dependencies

The first selected demo builds the CLI with the reference parser and external
JSON-comments, RapidYAML, and TOML adapters.
The build runs once per invocation.
Makes installs build dependencies, and Go modules fetch the plugin packages.
The RapidYAML adapter downloads its shared library when first needed.
Bash, GNU Make, curl, Git, and `less` must be available.
If `ys` is missing from `PATH`, the script installs the YAMLScript catalog
reader using in-1:

```bash
source <(curl -s https://in-1.cc) ys
```

The demo build selects the reference parser by default.
Run `make cli` from the repository root afterward to restore the ordinary
native CLI.

To use an existing executable and skip the build, set `GO_YAML_BIN`:

```bash
GO_YAML_BIN=/path/to/go-yaml ./cmd/go-yaml/demo/demo-plugins.sh
```

That executable must already contain the plugins you want to demonstrate.
Missing executables or implementations are reported as skips.
Unexpected failures are reported, and the remaining demos continue.
Exit status is 0 on success, 1 for unexpected failures, and 2 for invalid
arguments or when no demos pass.

## Catalog

[demo-plugins.yaml](demo-plugins.yaml) contains the demo definitions.
The four-letter keys `desc` and `cmnd` identify the description and shell
command.
Optional `make` arguments build the CLI, `impl` identifies the implementation,
and `fail` specifies text expected in a rejection.
The script reads these values using `ys`.
Commands invoke `go-yaml` directly.
The script routes that command to the selected executable.

See the [plugin guide](../../../docs/plugin.md) for the API and configuration
contracts.
