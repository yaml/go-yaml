# go-yaml Internals

This guide describes the v4 implementation and its processing boundaries.
For a conceptual introduction, see [How go-yaml Works](how-go-yaml-works.md).

## Entry Points and Options

The public package wraps `internal/libyaml` and installs the selected error
formatter and version defaults.
The internal package owns the processing stages.

| Public API | Processing |
|---|---|
| `Load`, `NewLoader` | Compose, resolve, construct |
| `Dump`, `NewDumper` | Represent, desolve, serialize |
| `Unmarshal`, `NewDecoder` | Load pipeline with v3 compatibility settings |
| `Marshal` | Dump pipeline with v3 settings and unlimited line width |
| `NewEncoder` | Dump pipeline with v3 output settings and v4 error formatting |
| `Node.Load`, `Node.Decode` | Construct an already composed node graph |
| `Node.Dump`, `Node.Encode` | Build nodes through a YAML serialization round trip |

`Load` requires exactly one document by default.
`WithAllDocuments` loads documents into a slice.
`WithSingleDocument` stops after the first document and leaves later documents
unread and unvalidated.
`Unmarshal` also keeps legacy handling for empty input and trailing content.
Version presets do not reproduce every legacy code path or undo v4 fixes.

`NewLoader` and `NewDumper` return both a value and an error.
Check the error before using the stream.
A loader reads documents until `io.EOF`.
A dumper writes document separators between values; check `Dump` and `Close`
errors so output failures are not lost.

Options are applied in order.
Loading and dumping options affect only the relevant stages.
Strict struct-field checking is off by default; duplicate-key checking is on.
Each `Node.Load` starts with fresh defaults, independently of an outer loader.
Inside custom hooks, pass options such as `WithKnownFields()` explicitly.

## Load Stack

```text
Bytes -> Reader -> Scanner -> Parser -> Composer -> Nodes
Nodes -> Resolver -> Tagged nodes -> Constructor -> Go values
```

### Reader and Scanner

`reader.go` detects supported input encodings and supplies UTF-8 data to the
scanner.
The scanner in `scanner.go` tracks indentation, flow collections, scalar
styles, comments, and source marks while producing tokens.
Reader and scanner state are held in the same `Parser` object used by the
parser stage.
There are no separate Reader or Scanner objects in this implementation.

`Mark.Line` and `Mark.Column` use one-based positions.
`Mark.Index` tracks decoded characters, not UTF-8 byte offsets; CRLF advances
it by two.
Do not slice the original input using `Index` as a byte offset.

### Parser

`Parser.Parse` in `parser.go` consumes scanner tokens and returns structural
events: stream and document boundaries, scalars, collections, and aliases.
Events carry tags, anchors, styles, positions, and comments.
The parser expands tag handles from directives; scalar type inference belongs
to the resolver.
Anchor lookup belongs to the composer rather than the parser.

### Composer

`Composer.Compose` in `composer.go` consumes events and builds a node graph.
Mappings contain alternating key and value children.
Document nodes contain a document root.
An alias node has an `Alias` pointer to its target.
The composer defines collection anchors before composing their children.

The default Alias-Data context keeps anchors within each document.
Explicit policies can supply external anchors, retain anchors across documents,
resolve dotted paths, or reject alias references.
A fresh context is created for each loading operation.
See [Alias-Data](../plugin/alias-data.md) for source precedence and callbacks.

`WithStreamNodes` exposes stream boundary metadata alongside document nodes.
Ordinary node loading returns document nodes without these extra boundaries.

### Resolver

`Resolver.Resolve` in `resolver.go` walks nodes and determines implicit tags.
Plain scalars such as `true`, `42`, and `null` acquire their YAML types.
Explicit tags and scalar styles affect resolution.
Selected YAML 1.1 behavior remains for compatibility; v4 does not claim complete
YAML 1.2 conformance.

### Constructor

`Constructor.Construct` in `constructor.go` converts resolved nodes into Go
values using reflection and the destination type.
It handles maps, slices, structs, scalar conversions, inline fields, and custom
unmarshaling hooks.
Generic mappings with string keys use `map[string]any`; other comparable key
types can require `map[any]any`.

Recognized v3 node-based unmarshaler hooks produce a migration error rather
than being invoked with incompatible node types.
The legacy callback hook remains supported.
`encoding.TextUnmarshaler` applies to scalar input.

Merge-Key selects effective mapping entries during construction.
Loading raw nodes and invoking custom unmarshaler hooks preserves their
original mapping structure.
Merge options must be passed again when a custom hook calls `Node.Load`.
See [Merge-Key](../plugin/merge-key.md) for conflict rules and replacement
algorithms.

## Dump Stack

```text
Go values -> Representer -> Tagged nodes -> Desolver -> Nodes
Nodes -> Serializer -> Events -> Emitter -> Writer -> Bytes
```

### Representer

`Representer.Represent` in `representer.go` converts Go values to nodes.
It handles reflection, field tags, `omitempty`, custom `MarshalYAML` hooks,
and `encoding.TextMarshaler`.
Scalar and collection style decisions begin here.
Existing nodes retain supported styles and comments.
Arbitrary Go pointer sharing does not automatically create YAML anchors or
support cyclic Go values.

### Desolver

`Desolver.Desolve` in `desolver.go` removes inferable tags.
For a string such as `"42"`, it can remove `!!str` and select quoting to keep
the value a string when loaded again.
Explicitly tagged nodes and custom tags are retained.
It does not remove every tag indiscriminately.

### Serializer, Emitter, and Writer

`Serializer.Serialize` in `serializer.go` walks nodes and pushes events to
`Emitter.Emit` in `emitter.go`.
It transfers anchors, aliases, tags, comments, and styles from the node graph.
The emitter chooses valid YAML presentations, indentation, preferred wrapping,
and document markers.
`writer.go` flushes output through the configured destination.

Line width is a preference, not a guaranteed maximum.
Wrapping preserves scalar content rather than inserting semantic newlines.
Node output preserves supported styles and comments, not the original bytes.
Indentation and document-marker options are not stored on a node; apply them
when writing YAML even if the node was built with `Node.Dump`.

## Comments

The scanner collects comments and source positions.
The parser associates comments with structural events.
The composer transfers them to `HeadComment`, `LineComment`, and `FootComment`,
including mapping-specific reassignment near collection boundaries.
`TAIL_COMMENT_EVENT` carries comments that need special treatment at a block
mapping boundary.

On output, the serializer transfers node comments to events, and the emitter
renders them near the associated content.
Constructing ordinary Go values discards comment metadata; load nodes when
comments matter.

## Safety Checks

The default depth limit is 10,000.
The native scanner checks flow and block nesting.
External parser event validation also checks nesting, but it cannot limit work
already done inside an external parser.
Depth settings do not retroactively validate arbitrary precomposed node graphs.

The constructor checks alias-related work against total construction work.
The default is a ratio-based heuristic with an initial allowance, not a fixed
maximum number of alias references in the source.
`loaderlimits.AliasValue` instead selects a fixed count of construction work
performed through aliases.
Recursive aliases and merge cycles are checked separately.
These checks also matter when custom unmarshaler hooks decode nodes directly.
See [LoaderLimits](../plugin/loader-limits.md) for policy overrides.

## Errors and Extension Points

Stages produce structured `LoadError`, `LoadErrors`, and `DumpError` values.
Use `errors.As` to inspect them rather than parsing formatted messages.
The ErrorFormat plugin controls presentation; it does not change the underlying
conversion or grammar rules.
The built-in v4 formatter has fixed messages and configurable position styles.

The public plugin APIs are YAML-parser, LoaderLimits, Merge-Key, Alias-Data,
Tab-indent, and Error-format.
There is no public resolver or emitter plugin interface.
Custom marshalers, unmarshalers, nodes, and ordinary options provide additional
application-level control.
See the [plugin guide](../plugin.md) for interfaces and registration contracts.

External parser input is buffered and its complete event stream is validated
before composition.
Only linked implementations can be selected by the CLI at runtime.
Optional parsers stay in separate modules so the core library does not acquire
their dependencies.

## Tests and Diagrams

Stage-specific tests and YAML fixtures live in `internal/libyaml`.
The shared fixture framework lives in `internal/testutil/datatest`.
The [internal package guide](../../internal/libyaml/README.md) explains the
fixture format and focused test commands.
Root tests exercise public options, compatibility, plugins, and node methods.
The YAML Test Suite integration tracks its known-failing baseline separately.

- [Pipeline overview](pipeline-overview.mmd)
- [Call hierarchy](call-hierarchy.mmd)
- [Comment flow](comment-flow.mmd)
