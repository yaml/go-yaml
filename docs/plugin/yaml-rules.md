# YAML Rules

The `yaml-rules` API selects declarative tag resolution and guides safe tag
omission and scalar quoting on dump.
It leaves native construction, content validation, parser syntax, and formatting
options to their existing stages.

```go
import "go.yaml.in/yaml/v4/plugin/yamlrules"

rules, err := yamlrules.New("yaml-1.2-core", nil)
if err != nil {
    return err
}
err = yaml.Load(input, &result, yaml.WithPlugin(rules))
```

Built-in names are `yamlstar`, `go-yaml-v2`, `go-yaml-v3`, `go-yaml-v4`,
`yaml-1.1-core`, `yaml-1.2-core`, `yaml-1.2-json`, `failsafe`, and `json`.
Names are opaque, lowercase identifiers; dots have no special meaning.
The existing default behavior is unchanged.
Boolean spellings such as `yes` resolve implicitly under `go-yaml-v2` and
`yaml-1.1-core`, while v3 and v4 resolve them as strings.
The v3 and v4 resolution names currently share behavior, independently of their
formatting presets.
`yamlstar` uses decimal integers, including leading zeros.
`json` requires a digit after a decimal point; `yaml-1.2-json` accepts `0.`.
Both still use the independently selected YAML parser.

```bash
go-yaml --plugin=yaml-rules=failsafe -j input.yaml
```

Define custom rules inline or select a file:

```yaml
plugin:
  yaml-rules:
    name: yaml-1.2-core
    .rules:
    - .when: {x: +Num, y: +Num}
      .tag: '!point'
    .root:
      id:
        .rules: +Str
      rows:
        .list:
          id:
            .rules: +Str
      .pairs:
      - .when:
          .key: +Str ~"if .*"
          .value: +Map
        .tags: {key: '!condition', value: '!!map'}
```

`.rules` at the definition root is an ordered global rule list.
`.root` mirrors document locations, and `.list` visits every sequence item.
Missing fields do not fail matching.
A `+Str` structural key applies to otherwise unmentioned mapping values.
Optional `.role` limits a flat rule to `root`, `item`, `key`, `value`, or `pair`.
Flat pair rules use `.role: pair` and the same `.key`/`.value` predicates as
structural `.pairs` rules.

Matchers support YAMLSchema-style compact built-in types and explicit `.type`,
`.match`, `.const`, `.size`, `.list`, `.any`, `.all`, `.one`, `.not`, and `.open`
predicates.
Built-in types are `+Any`, `+Str`, `+Int`, `+Float`, `+Num`, `+Bool`, `+Null`,
`+Map`, and `+Seq`.
`+Str[]` matches repeated strings; `+Str ~"pattern"` performs a full match.
`.size` accepts an integer or `[minimum, maximum]`; `[minimum]` is unbounded.
Property names ending in `?` are optional.
Extra properties are allowed unless `.open: false` is selected.
This is a documented matching subset, not the full YAMLSchema validator.
Patterns use Go regular expression syntax; unsupported expressions fail during
plugin creation.

Rules evaluate an immutable snapshot of original content and base-resolved tags.
They never run constructors.
Explicit YAML tags win over pair assignments, structural rules, flat rules,
and the selected base, in that order.
The first matching rule in a list wins.
A pair rule decides both assignments before either constructor runs.
Custom function tags require a separately registered Function-Tag constructor.

`file: rules.yaml` loads a separate definition using the current directory for
relative paths.
Inline fields override corresponding top-level file entries.
A definition's `.base` selects its inherited rules.
The shared structural tree reserves `.schema` and `.format` sections for future
independent actions at the same locations.
Those sections do not execute validation or formatting today.

The host uses the selected rules again when dumping and retains explicit tags
or quoting whenever omission would change a node's meaning, including custom
structural matches.
It does not execute Function-Tag functions during dumping.

Third-party implementations can implement `yaml.YAMLRulesPlugin` using public
`yaml.RuleInput` and `yaml.RuleDecision` types.
Node inputs, pair context, and paths are read-only.
No shared-library ABI extension is required.
