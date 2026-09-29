# flagforge design

flagforge uses one configuration file to generate Go command-line parsing code
and the corresponding Markdown or HTML documentation. This document describes
the implementation in this repository, including its current limitations.

## Structure and data flow

The root package, `github.com/rqlite/flagforge`, is a library. The executable is
in `cmd/flagforge`; it handles command-line options and file I/O while delegating
parsing and rendering to the library.

```text
TOML file
    │
    ▼
Parser.ParsePath → Viper → ParsedConfig
                              │
                              ▼
                         NewGenerator
                              │
                              ▼
                       Execute(format, writer)
                         /      |       \
                       Go    Markdown   HTML
```

| Location | Responsibility |
| --- | --- |
| `parser.go` | Configuration types, file/reader parsing, Go configuration defaults |
| `generator.go` | Format dispatch, templates, type-specific generation, visibility and section grouping |
| `cmd/flagforge/flagforge.go` | CLI options, input selection, output destination, optional prefix, process exit |
| `generator_test.go` | Generation smoke tests, golden-file comparisons, section behavior tests |
| `testdata/` | TOML examples and expected Go/HTML output, including a substantial rqlite configuration |
| `.circleci/config.yml` | Formatting, vet, and test checks |

The module declares Go 1.23.3 and directly depends on Viper 1.19.0. Viper and
its transitive dependencies handle configuration decoding. Rendering uses the
standard library. Generated Go code also uses only the standard library, so
applications do not need flagforge or Viper at runtime.

## Configuration model

`ParsedConfig` contains a `GoConfig`, an ordered slice of `Argument` values, and
an ordered slice of `Flag` values. Viper unmarshals the `go`, `arguments`, and
`flags` keys into these structures using `mapstructure` tags. Fixtures also use
`[Go]`, reflecting Viper's case-insensitive configuration keys.

The `[go]` settings control the generated source:

| Key | Default | Use |
| --- | --- | --- |
| `package` | `pkg` | Go package name |
| `config_type_name` | `Config` | Generated configuration struct name |
| `flag_set_name` | `name` | Name passed to `flag.NewFlagSet` |
| `flag_error_handling` | `ExitOnError` | Selector emitted after `flag.` |
| `flag_set_usage` | Empty string | Optional custom usage text |

Each `[[arguments]]` entry has `name`, `type`, `required`, `short_help`, and
`long_help`. Arguments appear as struct fields before flags, in configuration
order. Only string arguments receive generated assignments from parsed
positional arguments. `required` and argument `long_help` are decoded but do
not control generation; the argument checks are described below.

Each `[[flags]]` entry has these fields:

| Key | Meaning |
| --- | --- |
| `name` | Generated Go struct field name |
| `cli` | Name passed verbatim to the Go flag registration method |
| `type` | Selects the generated field type and registration method |
| `default` | Untyped decoded value used by the Go template |
| `delimiter` | Separator for `[]string` flags; defaults to a comma during Go generation |
| `short_help` | Struct comment, CLI usage text, and documentation summary |
| `long_help` | Additional documentation text |
| `section` | Documentation grouping name |
| `hide` | Removes the flag from generated code and documentation |

Use CLI names such as `http-addr`, without a leading dash: registration passes
the name through unchanged, while the standard Go flag package supplies the
command-line dash syntax. Some older fixtures contain names such as
`-node-id`; their generated code passes formatting checks but panics when
`Forge` attempts to register those flags.

`ParsePath` creates a fresh Viper instance and calls `SetConfigFile` followed by
`ReadInConfig`. Although the interface and error messages describe TOML, this
path does not explicitly force TOML; format selection is delegated to Viper.
The parser wraps read and unmarshal errors with context. It performs no general
schema validation for missing fields, duplicate names, supported types, or
valid Go identifiers.

## Library and CLI interfaces

A library caller uses `NewParser().ParsePath(path)`, passes the result to
`NewGenerator(cfg)`, and calls `Execute(format, writer)`. The formats are `Go`,
`Markdown`, and `HTML`; an unknown format returns an error. The writer belongs
to the caller and is not closed by the library.

`NewGenerator` currently always returns a nil error for a non-nil configuration;
it copies the Go settings and retains the argument and flag slices. It does
not validate the configuration or deep-copy its slices. Passing nil panics.

Build and run the executable from the repository root with:

```sh
go build -o flagforge ./cmd/flagforge
./flagforge -f go -o config_flags.go flags.toml
./flagforge -f markdown flags.toml
./flagforge -f html -p introduction.md -o flags.md flags.toml
```

The CLI defaults to Go output on stdout. It uses the first positional argument
as the input path, accepts the lowercase format names shown above, and ignores
additional positional arguments. Its own options must precede the input path
because it uses the standard `flag` parser.

The `-p` option copies a file's bytes before generated output, without inserting
a newline or interpreting the content. This supports documentation front matter
and introductions; it is a CLI feature, not part of `Generator`. Errors are
printed to stderr and terminate the process with status 1.

An `-o` destination is created or truncated after parsing the input, but before
reading the prefix or generating output. Writes are not atomic: failure can
leave an empty file, a prefix alone, or partial output. Deferred close errors
are not checked, and `os.Exit` bypasses deferred cleanup on error paths.

## Generated Go behavior

Go generation validates and normalizes selected defaults, renders a fixed
`text/template` into a buffer, and runs `go/format.Source` before writing it.
The output contains a generated-code notice, the configuration struct,
`Forge(arguments []string) (*flag.FlagSet, *Config, error)` (using the configured
struct name), and helper functions for durations, splitting, errors, and usage.
There is no template override or generated `main` function.

| Flag type | Generated behavior |
| --- | --- |
| `string` | `string` field registered with `StringVar` |
| `filepath` | `string` field with a `filepath:"true"` struct tag, registered with `StringVar` |
| `bool` | `BoolVar` |
| `int` | `IntVar` |
| `uint64` | `Uint64Var` |
| `int64` | `Int64Var` |
| `time.Duration` | `DurationVar`, with a default parsed by the generated `mustParseDuration` helper |
| `[]string` | Temporary string registered with `StringVar`, then split after parsing |

The filepath tag is metadata only; generated code does not check, expand, or
normalize paths. Slice flags split literally on the configured delimiter, with
no trimming or escaping. An empty string becomes nil. Repeated occurrences use
the final string value rather than appending elements.

Every call to `Forge` creates a fresh configuration and `flag.FlagSet`, checks
the raw input length for each declared argument, registers flags, optionally
installs a usage function, and calls `fs.Parse(arguments)`. On success, it copies
`fs.Arg(i)` into string argument fields, splits slice flags, and returns the
flag set and configuration. Returned parse errors yield `(nil, nil, err)`.
The configured flag error policy still applies: the default `ExitOnError` can
terminate the process rather than return an error.

Custom usage writes its text to stderr and then calls `fs.PrintDefaults()`.
Ordinary flag parsing behavior is inherited from Go, including stopping at the
first positional argument. Extra positional arguments remain available through
the returned flag set.

## Documentation generation

Both documentation formats operate on visible flags only. They omit positional
arguments and do not print type or default-value columns.

`groupBySection` preserves section order by first appearance and flag order
within each section. Nonadjacent flags with the same section are combined.
With no sections, it returns one anonymous group; this also produces an empty
table when there are no visible flags. If any visible flag has a section, all
visible flags must have one, otherwise generation fails with the unassigned CLI
names. Hidden flags do not participate in this check. Go generation ignores
sections entirely.

Markdown emits an optional `##` heading and a two-column `Flag | Usage` table
per section. It preserves the CLI name exactly, escapes pipes, and turns
newlines in flag text into `<br>`. It currently appends `long_help` only when
`default` is non-nil, adding a period to short help if needed. The default value
itself is never displayed. Other Markdown syntax and section names are emitted
without escaping.

HTML emits a fragment: optional Markdown `##` headings followed by HTML tables.
The mixed format is intentional for embedding in a static-site page whose
Markdown processor supplies heading anchors and table-of-contents entries.
Tables use `rq-flags`; header cells use `col-cli` and `col-usage`. Styling belongs
to the embedding site. Each CLI name gains a leading dash and appears in a
`<code>` element. Short help always gains a period; nonempty long help follows
`<br><br>`, independently of the default value. Flag names and help text are
HTML-escaped, but section headings are emitted as raw Markdown.

Go and HTML buffer their rendered output before writing it. Markdown writes
one section at a time, so a later writer failure can leave earlier sections
written. All formats propagate writer errors.

## Implementation limitations

These are observations of the current code, not intended guarantees:

- `ParseReader` creates a local Viper instance but calls the package-level
  `viper.ReadConfig(r)`, then unmarshals the untouched local instance. It also
  does not set a configuration type. It therefore does not provide a working
  equivalent of `ParsePath`: it can fail on the missing format, or, with global
  Viper configured externally, read into the wrong instance.
- Positional argument checks use `len(arguments)` before flag parsing and
  ignore `required`. An optional argument can be reported missing, while a
  flags-only invocation can satisfy the length check and leave a required
  string argument empty. Non-string argument fields are not populated.
- Go template values are interpolated directly into source, including quoted
  string literals and comments. There is no Go string-literal escaping step.
  Quotes, backslashes, or newlines in configuration text can change the emitted
  meaning or cause formatting failures.
- Formatting checks syntax, not types or runtime behavior. Unsupported flag
  types still produce fields but no registration; invalid type names, duplicate
  fields, bad defaults, or invalid error-policy selectors may survive generation
  and fail to compile or execute. Missing scalar defaults are not generally
  replaced with appropriate Go zero values.
- Go generation normalizes `[]string` defaults/delimiters and duration defaults
  in the retained flag slice. This changes the caller's configuration and can
  change later output: Markdown's long-help condition depends on a non-nil
  default. A missing duration default becomes integer `0` on the first Go run,
  but a second run rejects that value as a non-string default. Reusing a
  generator is therefore not reliably idempotent or safe for concurrent use.
- Duration checks run before hidden flags are filtered, so an invalid hidden
  duration can still fail Go generation despite being absent from its output.

## Verification and maintenance

The existing Go golden tests cover a single flag, multiple flag types,
arguments with flags, the rqlite example, and hiding flags. HTML golden tests
cover a single flag, grouping, and hiding. Separate tests cover section order,
partial-section rejection, and sections leaving generated Go unchanged.

These tests compare output bytes or check that generation succeeds. They do
not compile or invoke the generated `Forge` functions. There are no Markdown
output tests, reader-parser tests, or CLI integration tests. The leading-dash
fixtures illustrate why golden output alone does not establish runtime validity.

The configured CI uses Go 1.23.4 and runs formatting checks, `go vet ./...`, and
`go test -v ./...`. From the repository root, the principal checks are:

```sh
gofmt -l .
go vet ./...
go test ./...
```

Changes to the output contract should update the corresponding golden files.
New supported types require coordinated changes to field emission, registration,
default handling, and tests. Parser validation and compilation/runtime tests of
generated code would address gaps that formatting and golden comparisons cannot
detect. Shared slice mutation is also a constraint to resolve before promising
repeatable multi-format generation from one generator.
