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
| `generator.go` | Format dispatch, templates, visibility, section grouping, and output escaping |
| `validation.go` | Configuration validation and default normalization on private data |
| `cmd/flagforge/flagforge.go` | CLI options, input selection, output destination, optional prefix, process exit |
| `*_test.go` | Parser, validation, repeatability, golden-file, and section tests |
| `cmd/flagforge/flagforge_test.go` | Direct CLI calls covering file preservation, prefixes, formats, and errors |
| `testdata/` | TOML examples and expected Go/HTML output, including a substantial rqlite configuration |
| `.circleci/config.yml` | Formatting, vet, and test checks |

The module declares Go 1.23.3 and directly depends on Viper 1.19.0 and
mapstructure 1.5.0 for configuration decoding. Rendering uses the
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
order. Only string arguments are supported; other types are rejected. `required`
defaults to false, and required arguments must precede optional arguments.
Argument `long_help` is decoded but is not currently rendered.

Each `[[flags]]` entry has these fields:

| Key | Meaning |
| --- | --- |
| `name` | Generated Go struct field name |
| `cli` | Name passed verbatim to the Go flag registration method |
| `type` | Selects the generated field type and registration method |
| `default` | Decoded value validated and normalized to the flag type |
| `delimiter` | Separator for `[]string` flags; defaults to a comma when creating the generator |
| `short_help` | Struct comment, CLI usage text, and documentation summary |
| `long_help` | Additional documentation text |
| `section` | Documentation grouping name |
| `hide` | Omits flag registration and documentation, but retains the Go configuration field |

Use CLI names such as `http-addr`, without a leading dash. The standard Go flag
package supplies the command-line dash syntax. Visible CLI names must be unique,
nonempty, and free of leading dashes, whitespace, and `=`.

`ParsePath` and `ParseReader` each create a fresh Viper instance with the format
explicitly set to TOML. Path parsing also works without a `.toml` extension.
Both paths decode the entire configuration with `UnmarshalExact`, reporting
unknown keys, including misspellings within flags and arguments. Weak type
coercion is disabled. Read and unmarshal failures are wrapped with context.

## Library and CLI interfaces

A library caller uses `NewParser().ParsePath(path)`, passes the result to
`NewGenerator(cfg)`, and calls `Execute(format, writer)`. The formats are `Go`,
`Markdown`, and `HTML`; an unknown format returns an error. The writer belongs
to the caller and is not closed by the library.

`NewGenerator` rejects nil input, copies the settings and slices, and validates
the private configuration. It checks Go identifiers, generated-name conflicts,
duplicate fields and CLI names, supported types, argument ordering, error policy,
and default types and ranges. This validation applies to all output formats.
Defaults are normalized on the private copy. Subsequent generation does not
mutate the generator or caller configuration, so repeated and mixed-format calls
are independent of execution order.

Build and run the executable from the repository root with:

```sh
go build -o flagforge ./cmd/flagforge
./flagforge -f go -o config_flags.go flags.toml
./flagforge -f markdown flags.toml
./flagforge -f html -p introduction.md -o flags.md flags.toml
```

The CLI defaults to Go output on stdout. It requires exactly one positional argument
as the input path and accepts the lowercase format names shown above. Its own options must precede the input path
because it uses the standard `flag` parser.

The `-p` option copies a file's bytes before generated output, without inserting
a newline or interpreting the content. This supports documentation front matter
and introductions; it is a CLI feature, not part of `Generator`. Errors are
printed to stderr and terminate the process with status 1.

The CLI reads the prefix and renders the complete output before writing. For
`-o`, it writes a file in a private temporary directory beside the destination,
checks both the write and close, then renames the file over the destination. Earlier
failures preserve existing content. The prefix may be the output file itself.
Existing regular-file permissions are preserved, existing symbolic links are
resolved to their targets, and new files use normal creation permissions (`0666`
masked by the process umask). Nonregular destinations are rejected. Temporary
files are cleaned up on failure. Filesystem rename guarantees apply; this is not a promise of
power-loss durability. Stdout is written after rendering and propagates write
errors, but a failing stdout writer may already have received some bytes.

CLI logic lives in `run(args, stdout, stderr)`, allowing direct tests without
subprocesses. Only `main` prints returned errors and exits the process.

## Generated Go behavior

Go generation renders a fixed `text/template` into a buffer and runs
`go/format.Source` before writing it. String literals use `strconv.Quote`, and
multiline help comments receive a comment prefix on every line.
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

Hidden flags retain their fields, types, comments, and applicable struct tags in
the generated configuration. They are not registered with the flag set or
populated during parsing, so their fields retain Go zero values, even when a
default is configured. Hidden CLI names, delimiters, and defaults do not
participate in registration or default validation; their field names and types
are still validated. Applications can populate these fields programmatically.

The filepath tag is metadata only; generated code does not check, expand, or
normalize paths. Slice flags split literally on the configured delimiter, with
no trimming or escaping. An empty string becomes nil. Repeated occurrences use
the final string value rather than appending elements.

Omitted defaults become `""` for string, filepath, and slice flags; `false` for
boolean flags; `0` for integer flags; and `"0s"` for duration flags. Defaults with
incompatible types are rejected. Integer validation accepts signed or unsigned
integer values within the target type's range; `int` uses the generator's host
word size. Explicit duration defaults are checked with `time.ParseDuration`.

Every call to `Forge` creates a fresh configuration and `flag.FlagSet`, registers
flags, optionally installs a usage function, and calls `fs.Parse(arguments)`.
After successful parsing, required arguments are checked using `fs.NArg()`.
Optional arguments may be absent. The function copies `fs.Arg(i)` into argument
fields, splits slice flags, and returns the flag set and configuration. Returned parse errors yield `(nil, nil, err)`.
The configured flag error policy still applies: the default `ExitOnError` can
terminate the process rather than return an error.

Custom usage writes its text to stderr and then calls `fs.PrintDefaults()`.
For compatibility with existing configurations, usage text interprets Go escape
sequences before the result is quoted for generated source. Thus a TOML literal
usage string containing `\n` still produces actual line breaks. Actual newlines
and quotes also work, and unknown escape sequences are preserved. This usage
compatibility rule does not apply to flag defaults or help text.
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
newlines in flag text into `<br>`. It appends nonempty `long_help` independently of
the default value, adding punctuation between the summary and details when
needed. The default value itself is never displayed. Other Markdown syntax and
section names are emitted without escaping.

HTML emits a fragment: optional Markdown `##` headings followed by HTML tables.
The mixed format is intentional for embedding in a static-site page whose
Markdown processor supplies heading anchors and table-of-contents entries.
Tables use `rq-flags`; header cells use `col-cli` and `col-usage`. Styling belongs
to the embedding site. Each CLI name gains a leading dash and appears in a
`<code>` element. Nonempty short help gains a period unless it ends in `.`, `!`,
or `?`; nonempty long help follows `<br><br>`, independently of the default value.
Flag names and help text are
HTML-escaped, but section headings are emitted as raw Markdown.

Go and HTML buffer their rendered output before writing it. Markdown writes
one section at a time, so a later writer failure can leave earlier sections
written. All formats propagate writer errors.

## Scope and limitations

Generated applications use the standard Go flag package's parsing and error
policies. There is no support for custom flag types, numeric positional
arguments, runtime configuration-file loading, or environment-variable overrides.
Hidden fields remain available for application code to populate directly.

Generation checks its supported schema and formats Go source, but does not
compile the generated code together with the consuming application. Applications
must avoid collisions with generated declarations such as `Forge` and the helper
functions. Documentation renders section names as Markdown and does not provide
a general Markdown sanitization layer. Output writers remain caller-owned.

## Verification and maintenance

Go golden tests cover single and multiple flags, positional arguments, the
rqlite example, hidden fields, zero defaults, quoting, multiline comments, and
required/optional argument output. HTML and Markdown golden tests cover visibility,
sections, escaping, punctuation, and help without defaults. Golden outputs are
produced by executing the generator on each fixture's TOML input.

Unit tests cover path/reader equivalence, strict decoding, invalid configurations,
repeatable generation, caller-data isolation, and section grouping. CLI tests
call `run` directly to check output preservation, prefix/output reuse, permissions,
argument errors, formats, and writer errors. Tests do not start subprocesses or
compile and invoke generated `Forge` functions.

The configured CI uses Go 1.23.4 and runs formatting checks, `go vet ./...`, and
`go test -v ./...`. From the repository root, the principal checks are:

```sh
gofmt -l .
go vet ./...
go test ./...
```

Changes to the output contract should update the corresponding golden files.
New supported types require coordinated changes to field emission, registration,
default handling, and tests. Keep normalization confined to the private
configuration so generation remains repeatable across output formats.
