# Reporting typed loading issues

Include the library revision, input, destination type, command, and actual
output in a report.
For a fixture containing `type: '[]string'` and
`yaml: '[before, null, after]'`, the corresponding command is:

```bash
printf '%s\n' '[before, null, after]' | go-yaml --type '[]string' -j
```

Use `--decode` or `--unmarshal` to exercise the classic loading APIs.
The default uses Loader.Load and accepts loading options through `-o`.
Each API receives a pointer to the specified type directly.

Custom UnmarshalYAML methods and application-defined types still require a
Go reproducer.
Anonymous structs and fixture-registry integration are not supported.

On library revision `2008d62`, with this CLI addition, the example above
produced `["before","after"]`.
This reproduces the null-element loss motivating PR #503.
The CLI change does not include that constructor fix.
