# Agent guidance for go-yaml

## Project and workflow

- The active library import path is `go.yaml.in/yaml/v4`.
  Use `main` as the default PR base.
- Read [CONTRIBUTING.md](CONTRIBUTING.md) for the development workflow.
  Check `GNUmakefile` for current targets rather than relying on older notes.
- Use Makes-managed tools: `make shell CMD='<command>'`.
  Dependencies are installed under `.cache/`.
- Related repositories must be under `repos/<name>`.
  If a required checkout is missing, report it rather than using a sibling
  checkout outside this worktree.
- Preserve unrelated work and keep fixes within the requested scope.
  Do not turn a documentation review into a CLI implementation change.

## Reproduce and discuss behavior with the CLI

Prefer the [go-yaml CLI](cmd/go-yaml/README.md) instead of Go code to demonstrate
YAML behavior when possible, including in issues, PRs, and discussions.
Follow [Reporting with the CLI](docs/reporting.md).

- Build with `make cli` and run `./go-yaml` to exercise this checkout.
  Rebuild after code or revision changes.
- Include minimal input, the exact command, actual output or error, expected
  output, revision, and relevant options, configuration, and plugins.
- For behavior changes, compare identical input and flags before and after the
  patch and identify both revisions.
- Run output modes separately, with flags before the input file.
  Use `-t`/`-T` for tokens, `-e`/`-E` for events, `-n`/`-N` for nodes,
  `-j`/`-J` for JSON, and `-y`/`-Y` for YAML.
  Do not use the obsolete `-p` flag or combine output flags to inspect stages.
- `-y` encodes ordinary values; `-Y` uses nodes to retain supported styles and
  comments, without promising exact formatting preservation.
- The `v3` option preset selects compatibility defaults within v4.
  It does not run the v3 library or undo all v4 behavior changes.

### Verify API paths before claiming a comparison

Trace the active handler rather than trusting flag names or comments.
At the time this guidance was added, `--unmarshal --marshal -y` calls
`yaml.Load` and `yaml.Dump` in `cmd/go-yaml/yaml.go`, without classic defaults.
Do not describe that command as exercising `Unmarshal` and `Marshal`.
Use a small Go reproducer calling those functions directly.
The `--decode --encode -y` path does use Decoder/Encoder.
Recheck these paths if the implementation changes.

The CLI currently loads generic values or nodes and cannot choose a concrete
Go destination such as `[]string`.
For typed construction bugs, struct tags, or custom marshalers, use a typed
fixture or a runnable Go reproducer when needed.
For example, PR #503's disappearing null entries in `[]string` cannot be
reproduced by generic JSON output, which preserves the null entries.
Inspecting tokens alone does not reproduce a typed-construction failure.
A proposed `--type` option is not an available feature.

## Strongly prefer testdata tests

Write regression tests as YAML cases in the existing `testdata/` style whenever
the fixture harness can express the behavior.
Turn the minimal reproduction into a fixture, following nearby cases and the
corresponding Go handler.
Use a Go test when the required setup or assertions cannot be expressed by the
harness, and explain the reason in the PR.

Read these before adding cases:

- [Writing testdata cases](CONTRIBUTING.md#testing): fixture locations, an
  example, and test commands.
- [Stage fixture format](internal/libyaml/README.md#test-layout): scanner,
  parser, and emitter cases and their supported fields.
- [Focused stage tests](internal/libyaml/README.md#running-tests).

Root fixtures include `testdata/decode.yaml`, `testdata/encode.yaml`, and
`testdata/node.yaml`.
Stage fixtures live under `internal/libyaml/testdata/`.
Fixture schemas differ; inspect the file header and test handler.
The shared framework lives in `internal/testutil/datatest/`.
Quote fixture strings when YAML would otherwise resolve them to another type.

The `decode` handler for `testdata/decode.yaml` currently calls `yaml.Unmarshal`.
Do not claim a fixture covers both Load and Unmarshal without checking the
handlers and adding the required coverage.
CLI examples complement automated regression tests; they do not replace them.

## Validation and reporting

- Use focused checks while developing: `make test-main`, `make test-internal`,
  `make test-cmd`, or a selected test through `make shell CMD='go test ...'`.
- For parser/scanner changes, also run `make test-yts-all`.
  `make get-test-data` obtains the YAML Test Suite fixtures.
- `make test` runs the full test set.
  `make check` runs formatting, module tidying, linting, and tests; it can modify
  files, so review the resulting diff.
- Run ShellCheck after changing shell scripts.
- For documentation-only changes, verify commands and link targets, and run
  `git diff --check`; avoid unrelated formatting or code changes.
- Assess review comments against current source and reproduce relevant behavior.
  State which findings are valid and what was changed.
- Distinguish local validation, committed changes, and externally published
  state in reports.
  Do not suggest `git push` in this environment.

## Formatting and commits

- Do not use em dashes.
- Prefer lines within 80 columns and start each documentation sentence on a
  new line, respecting established file conventions.
- In YAML, do not indent sequences beneath their mapping key.
  Quote scalars only when needed to preserve their intended meaning or type.
- Use Perl instead of sed or awk.
- Follow [Commit Conventions](CONTRIBUTING.md#commit-conventions) and validate
  the PR commit range with `util/check-commit-messages <base>..HEAD`.
- Subjects must be 20 to 50 characters, inclusive, start with a capital letter,
  and have no final period, consecutive spaces, or trailing spaces.
  Do not use subject prefixes such as `fix:` or `docs:`, or bracketed prefixes.
- Separate the subject and body with a blank line.
  Body lines must not have trailing spaces.
  Do not add agent co-authorship.
  `Signed-off-by` trailers are allowed but are not required by CI.
- Include a descriptive body, start each sentence on a new line, and wrap at
  72 columns; these are preferences beyond the CI-enforced checks.
- Do not use merge commits; a separate CI check rejects them.
- Stage only the intended changes and report the commit hash and working-tree
  status after committing.
