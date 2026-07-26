# flagforge

[![Circle CI](https://circleci.com/gh/rqlite/flagforge/tree/master.svg?style=svg)](https://circleci.com/gh/rqlite/flagforge/tree/master)

_flagforge_ allows you to automatically generate Go [flag](https://pkg.go.dev/flag) code, as well as the associated Markdown and HTML documentation for those flags, all using a single configuration file. This means you only have to define your command-line options once in a TOML file, and _flagforge_ will do the rest.

## Running _flagforge_
Clone the repo and execute `go build`. Pass `-h` to `flagforge` to learn how to use it.
```bash
flagforge -f go|markdown|html <TOML file>
```

Pass `-header <file>` to copy the contents of a file to the output before the generated content. This is how a generated documentation page keeps hand-written material -- front matter, an introduction -- that would otherwise be lost every time the page is regenerated.

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

Sections appear in the order they first appear in the TOML file, and a flag joins a section that has already appeared rather than opening a new one, so flags belonging to the same section need not be adjacent. If any flag declares a section then every flag must; a partially sectioned file is an error, since otherwise each newly added flag would silently collect in an unnamed group.

`section` affects documentation only -- the generated Go code is unchanged by it.

The HTML output is a fragment rather than a complete document, so that it can be embedded in a page that supplies its own styling. Each table carries the class `rq-flags`, and section headings are emitted as Markdown `##` headings so that a static site generator gives them anchors and a table-of-contents entry.

## Example usage
[rqlite](https://www.rqlite.io) uses flagforge to generate the code and documentation for its extensive set of command-line flags:
- [rqlite TOML file](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/flags.toml)
- [Generated Go code](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/config_flags.go) for command-line flag parsing, and then [calling the generated code](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/flags.go#L297) from rqlite.
- Example of [automatically generated HTML documentation](https://rqlite.io/docs/guides/config/) for the flags deployed to production site. You can review the generated HTML [here](https://raw.githubusercontent.com/rqlite/rqlite.io/refs/heads/master/content/en/docs/Guides/config/_index.md).
