# flagforge

[![Circle CI](https://circleci.com/gh/rqlite/flagforge/tree/master.svg?style=svg)](https://circleci.com/gh/rqlite/flagforge/tree/master)

_flagforge_ allows you to automatically generate Go [flag](https://pkg.go.dev/flag) code, as well as the associated Markdown and HTML documentation for those flags, all using a single configuration file. This means you only have to define your command-line options once in a TOML file, and _flagforge_ will do the rest.

## Running _flagforge_
Clone the repo and execute `go build`. Pass `-h` to `flagforge` to learn how to use it.
```bash
flagforge -f go|markdown|html <TOML file>
```

## Example usage
[rqlite](https://www.rqlite.io) uses flagforge to generate the code and documentation for its extensive set of command-line flags:
- [rqlite TOML file](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/flags.toml)
- [Generated Go code](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/config_flags.go) for command-line flag parsing, and then [calling the generated code](https://github.com/rqlite/rqlite/blob/v8.36.8/cmd/rqlited/flags.go#L297) from rqlite.
- Example of [automatically generated HTML documentation](https://rqlite.io/docs/guides/config/) for the flags.
