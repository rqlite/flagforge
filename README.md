# flagforge

[![Circle CI](https://circleci.com/gh/rqlite/flagforge/tree/master.svg?style=svg)](https://circleci.com/gh/rqlite/flagforge/tree/master)

_flagforge_ allows you to automatically generate Go [flag](https://pkg.go.dev/flag) code, as well as the associated Markdown and HTML documentation for those flags, all using a single configuration file. This means you only have to define your command-line options once in a TOML file, and _flagforge_ will do the rest.

## Running _flagforge_
Clone the repo and build the command from the repository root. Pass `-h` to
`flagforge` to learn how to use it.
```bash
go build -o flagforge ./cmd/flagforge
./flagforge -f go -o config_flags.go flags.toml
```

The supported formats are `go` (the default), `markdown`, and `html`. Omit `-o`
to write to stdout. Supply exactly one input TOML file, with options before its
path. File output is replaced only after the prefix and generated content have
been prepared and the replacement file has been successfully written and closed.

Pass `-p <file>` to copy the contents of a file to the output before the generated content. This is how a generated documentation page keeps hand-written material -- front matter, an introduction -- that would otherwise be lost every time the page is regenerated.

## Configuration rules

Unknown configuration keys are errors. Go package, configuration type, and field
names must be valid Go identifiers; field names must be unique across arguments
and flags. CLI names must be unique and must omit leading dashes, whitespace,
and `=`. Use `cli = "node-id"`, not `cli = "-node-id"`.

Supported flag types are `string`, `filepath`, `bool`, `int`, `int64`, `uint64`,
`time.Duration`, and `[]string`. Omitted defaults use the type's zero value.
String, filepath, and slice defaults must be strings; duration defaults must be
valid duration strings such as `"10s"`; boolean and integer defaults must have
their corresponding TOML types. `[]string` uses a comma delimiter unless one is
specified. An empty slice default produces a nil slice.

Positional arguments support `type = "string"`. Mark required arguments with
`required = true` and place them before optional arguments. Required arguments
are checked after parsing flags. `flag_error_handling` accepts `ContinueOnError`,
`ExitOnError` (the default), or `PanicOnError`.

Flag values and help text are preserved as decoded from TOML. For compatibility,
`flag_set_usage` also interprets Go escape sequences, so both the original
`flag_set_usage = 'Usage:\n  example [flags]\n'` form and TOML basic or multiline
strings produce line breaks. Use `\\` in a TOML literal usage string for a literal
backslash. Unknown usage escape sequences are preserved.

## Grouping flags into sections
Give a flag an optional `section` key and the generated Markdown and HTML documentation will group flags under a heading of that name:

```toml
[[flags]]
name = "HTTPAddr"
cli = "http-addr"
type = "string"
default = "localhost:4001"
short_help = "HTTP server bind address"
section = "HTTP API"
```

Sections appear in the order they first appear in the TOML file, and a flag joins a section that has already appeared rather than opening a new one, so flags belonging to the same section need not be adjacent. If any visible flag declares a section then every visible flag must; a partially sectioned file is an error, since otherwise each newly added flag would silently collect in an unnamed group.

`section` affects documentation only -- the generated Go code is unchanged by it.

The HTML output is a fragment rather than a complete document, so that it can be embedded in a page that supplies its own styling. Each table carries the class `rq-flags`, and section headings are emitted as Markdown `##` headings so that a static site generator gives them anchors and a table-of-contents entry.

## Hiding flags

Set `hide = true` on a flag to omit it from command-line flag registration and
generated documentation while keeping its field in the generated configuration
struct. The field retains its Go zero value, even if the flag has a configured
default, and can be set programmatically.

## Example usage
[rqlite](https://www.rqlite.io) uses flagforge to generate the code and documentation for its extensive set of command-line flags:
- [rqlite TOML file](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/flags.toml)
- [Generated Go code](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/config_flags.go) for command-line flag parsing, and then [calling the generated code](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/flags.go#L297) from rqlite.
- Example of [automatically generated HTML documentation](https://rqlite.io/docs/guides/config/) for the flags deployed to production site. You can review the generated HTML [here](https://raw.githubusercontent.com/rqlite/rqlite.io/refs/heads/master/content/en/docs/Guides/config/_index.md).
