# Report and discuss behavior with the go-yaml CLI

**Please use the `go-yaml` CLI in bug reports, pull requests, and discussions
about YAML behavior.**
A small input, an exact command, and its output give everyone a shared example
they can run, change, and compare.
The CLI uses the go-yaml library and exposes its processing stages as well as
its final output.
You do not need to understand the internals to report a bug.

Use the CLI to demonstrate the behavior, then paste the example into your
GitHub issue, pull request, or discussion.
The CLI does not submit reports to GitHub.
If a problem depends on Go types, struct tags, or custom marshalers, include a
small runnable Go program too.
If the CLI cannot reproduce it, say so and show the Go reproducer instead.
Documentation and other changes without YAML behavior do not need a CLI example.

## Install or build

Install the released CLI with Go:

```bash
go install go.yaml.in/yaml/v4/cmd/go-yaml@latest
go-yaml -h
```

Or build the current checkout:

```bash
make cli
./go-yaml -h
```

The Makefile installs its build tools under `.cache/`.
The examples below use `go-yaml`; use `./go-yaml` for a checkout build so you
exercise that checkout rather than an older executable on your PATH.
Run `make cli` again after changing code or switching revisions.

## Make a small, runnable report

Reduce the input to the smallest example that still shows the problem.
Paste input and output as fenced text, preserving indentation, blank lines,
comments, and quoting.
Use a quoted here-document to prevent the shell from expanding YAML content:

```bash
cat > input.yaml <<'YAML'
x-keys:
  18_24: a
YAML
go-yaml -y input.yaml
```

Output from the current implementation:

```yaml
x-keys:
  1824: a
```

Explain what you expected and why.
For example, if you want the key's original spelling retained, show that
expected output separately from what the command actually printed.
This example comes from the [mapping-key discussion][key-example].
An observed behavior is not necessarily a bug; an example also makes a feature
request or question concrete.

For a short example, pipe input directly:

```bash
printf '%s\n' ': : :' | go-yaml -y
```

That is the reproduction used in an [empty-key parsing report][empty-key].
Include the output you get with your version, including any error.
Do not replace it with the output from an older report.
For whitespace-sensitive problems, provide a command such as `printf` that
recreates tabs, carriage returns, or missing final newlines exactly.

Include these details with your example:

- The exact command, minimal input, actual output or error, and expected result.
- All options, configuration file contents, and plugin selections used.
- CLI version (shown by `go-yaml -h`), Go version, and operating system.
- The installed module version or checkout commit and any local changes.

For an installed executable, collect build information with:

```bash
go version
go version -m "$(command -v go-yaml)"
```

For a checkout build, use:

```bash
git rev-parse HEAD
git status --short
make shell CMD='go version'
make shell CMD='go version -m ./go-yaml'
```

If you built with `CONFIG`, `PLUGIN`, or `PLUGIN-PACKAGE`, include the build
command and configuration too.

## Choose the output that shows the problem

Start with the final output you care about.
Put flags before the input file name and run each output mode separately.

| Goal | Command |
|---|---|
| Show ordinary YAML encoding | `go-yaml -y input.yaml` |
| Inspect YAML styles and comments through nodes | `go-yaml -Y input.yaml` |
| Show resolved values as JSON | `go-yaml -J input.yaml` |
| Inspect the node tree | `go-yaml -n input.yaml` |
| Include scalar tags and styles | `go-yaml -N input.yaml` |
| Inspect parser events | `go-yaml -e input.yaml` |
| Include event source positions | `go-yaml -E input.yaml` |
| Inspect scanner tokens | `go-yaml -t input.yaml` |
| Include token source positions | `go-yaml -T input.yaml` |

`-y` and `-Y` use different paths; choose the one relevant to your report.
`-Y` retains supported styles and comments but does not promise identical
formatting.
JSON cannot represent every YAML value, so use YAML or node output when JSON
conversion itself would obscure the problem.

For a missing or misplaced comment, compare tokens, events, and nodes:

```bash
go-yaml -t input.yaml
go-yaml -e input.yaml
go-yaml -n input.yaml
go-yaml -Y input.yaml
```

In this [comment-loss investigation][comments], the comment appeared in tokens
and events but disappeared from nodes.
Comparing stages narrowed down where the problem occurred.
You can use the same technique for tags, anchors, aliases, and document
boundaries; include only the output relevant to your report.

## Compare options and APIs

Make the settings part of the reproduction:

```bash
go-yaml -y input.yaml
go-yaml -y -o v3 input.yaml
go-yaml -y -o indent=4 input.yaml
go-yaml -Y -C options.yaml input.yaml
```

Include `options.yaml` when using `-C`.
Use `go-yaml -o help` to discover options.
The `v3` preset selects compatibility defaults in the v4 library; it does not
run the v3 library or undo all v4 behavior changes.

The default CLI path uses Load/Dump.
To compare the classic Decoder/Encoder APIs, use:

```bash
go-yaml --decode --encode -y input.yaml
```

For comparisons involving `Unmarshal` and `Marshal`, use a small Go reproducer
that calls those functions directly.
The CLI's `--unmarshal --marshal -y` path currently calls `Load` and `Dump`,
so it does not demonstrate the classic APIs' behavior or defaults.

Classic modes have restrictions on options and plugins.
See the [CLI reference](../cmd/go-yaml/README.md) for supported combinations
and custom plugin builds.
A CLI comparison does not exercise your application's concrete Go types;
include Go code when those types affect the result.

## Show a pull request's before and after

For a bug fix or behavior change, put a CLI example in the PR description:

1. Give the minimal input and exact command.
2. Show the output from the base revision and identify that revision.
3. Rebuild with `make cli` on the PR revision and run the same example.
4. Show the new output, explain why it is correct, and identify the revision.
5. Add a regression case, preferably in the existing YAML `testdata/` style,
   and list the tests you ran.

Keep input and flags identical unless changing those is part of the proposal.
CLI examples help reviewers understand the change; they complement regression
tests in the repository.
**We strongly encourage using the existing `testdata/` style for tests.**
Turn the minimal reproduction into a YAML case in the appropriate fixture file,
following nearby cases and the corresponding test handler.
Use a Go test when the fixture harness cannot express the required setup or
assertions, and explain that choice in the PR.
See [Writing testdata cases](../CONTRIBUTING.md#testing) for fixture locations,
an example, and test commands.
For stage-specific tests, read the
[fixture format](../internal/libyaml/README.md#test-layout) and
[focused test commands](../internal/libyaml/README.md#running-tests).

For example, this [scalar-formatting PR review][pr-example] compares CLI output
for a top-level scalar and a scalar inside a collection before and after the
patch.
Testing both contexts made the scope of the formatting change visible.

## Ask questions and propose behavior

Use the same input/command/output format in general discussions and feature
requests.
Show today's behavior and the output you would like, and explain the use case.
Clearly label proposed flags or output as proposals so readers do not mistake
them for working commands.
For conceptual questions without a runnable example, describe what you are
trying to achieve.

Open an [issue](https://github.com/yaml/go-yaml/issues/new), start a
[discussion](https://github.com/yaml/go-yaml/discussions), or share the example
with the [maintainers on Slack](https://cloud-native.slack.com/archives/C08PPAT8PS7).
For the development workflow, see [CONTRIBUTING.md](../CONTRIBUTING.md).

[key-example]: https://github.com/yaml/go-yaml/issues/491#issuecomment-6691961980
[empty-key]: https://github.com/yaml/go-yaml/issues/492#issuecomment-6691609216
[comments]: https://github.com/yaml/go-yaml/issues/416#issuecomment-5752257550
[pr-example]: https://github.com/yaml/go-yaml/pull/398#issuecomment-5486141276
