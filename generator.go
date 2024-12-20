package flagforge

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"strings"
	"text/template"
	"time"
)

const flagTemplate = `
// Code generated by go generate; DO NOT EDIT.
package {{ .Pkg }}

import (
	"flag"
	"fmt"
{{- if .FSUsage }}
	"os"
{{- end }}
	"strings"
	"time"
)

// StringSlice wraps a string slice and implements the flag.Value interface.
type StringSliceValue struct {
	ss *[]string
}

// NewStringSliceValue returns an initialized StringSliceValue.
func NewStringSliceValue(ss *[]string) *StringSliceValue {
	return &StringSliceValue{ss}
}

// String returns a string representation of the StringSliceValue.
func (s *StringSliceValue) String() string {
	if s.ss == nil {
		return ""
	}
	return fmt.Sprintf("%v", *s.ss)
}

// Set sets the value of the StringSliceValue.
func (s *StringSliceValue) Set(value string) error {
	*s.ss = strings.Split(value, ",")
	return nil
}

// {{ .ConfigType }} represents all configuration options.
type {{ .ConfigType }} struct {
{{- range .Args }}
	// {{ .ShortHelp }}
	{{ .Name }} {{ .Type }}
{{- end }}
{{- range .Flags }}
	// {{ .ShortHelp }}
	{{ .Name }} {{ .Type }}
{{- end }}
}

// Forge sets up and parses command-line flags.
func Forge(arguments []string) (*flag.FlagSet, *{{ .ConfigType }}, error) {
	config := &{{ .ConfigType }}{}
	fs := flag.NewFlagSet("{{ .FSName }}", flag.{{ .FSErrorHandling }})
{{- range $index, $element := .Args }}
	if len(arguments) <= {{ $index }} {
		return nil, nil, fmt.Errorf("missing required argument: {{ $element.Name }}")
	}
	{{- if eq .Type "string" }}
	config.{{ .Name }} = fs.Arg({{ $index }})
	{{- end }}
{{- end }}
{{- range .Flags }}
	{{- if eq .Type "string" }}
	fs.StringVar(&config.{{ .Name }}, "{{ .CLI }}", "{{ .Default }}", "{{ .ShortHelp }}")
	{{- else if eq .Type "bool" }}
	fs.BoolVar(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- else if eq .Type "int" }}
	fs.IntVar(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- else if eq .Type "uint64" }}
	fs.Uint64Var(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- else if eq .Type "int64" }}
	fs.Int64Var(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- else if eq .Type "time.Duration" }}
	fs.DurationVar(&config.{{ .Name }}, "{{ .CLI }}", mustParseDuration("{{ .Default }}"), "{{ .ShortHelp }}")
	{{- else if eq .Type "[]string" }}
	fs.Var(NewStringSliceValue(&config.{{ .Name }}), "{{ .CLI }}", "{{ .ShortHelp }}")
	{{- end }}
{{- end }}
{{- if .FSUsage }}
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "{{ .FSUsage }}")
		fs.PrintDefaults()
	}
{{- end }}
    if err := fs.Parse(arguments); err != nil {
	    return nil, nil, err
    }
	return fs, config, nil
}

func mustParseDuration(d string) time.Duration {
	td, err := time.ParseDuration(d)
	if err != nil {
		panic(err)
	}
	return td
}

`

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
<style>
table {
	width: 100%;
	border-collapse: collapse;
}
th, td {
	border: 1px solid #ddd;
	padding: 8px;
}
th {
	background-color: #f2f2f2;
	text-align: left;
}
.col-cli { width: 30%; }
.col-usage { width: 70%; }
</style>
</head>
<body>

<table>
	<tr>
		<th class="col-cli">Flag</th>
		<th class="col-usage">Usage</th>
	</tr>
	{{- range .Flags }}
	<tr>
		<td><code>{{ .CLI | html }}</code></td>
		<td>{{ .ShortHelp | html }}.
		{{- if .LongHelp }}
		    <br><br>{{ .LongHelp | html }}
		{{- end }}</td>
	</tr>
	{{- end }}
</table>

</body>
</html>
`

// Format represents the output format of the generator.
type Format int

const (
	Go Format = iota
	Markdown
	HTML
)

// String returns the string representation of the format.
func (f Format) String() string {
	switch f {
	case Go:
		return "Go"
	case Markdown:
		return "Markdown"
	case HTML:
		return "HTML"
	default:
		return "Unknown"
	}
}

// Generator represents a flag, HTML, or Markdown generator.
type Generator struct {
	pkg            string
	configTypeName string

	flagSetUsage         string
	flagSetName          string
	flagSetErrorHandling string

	args  []Argument
	flags []Flag
}

// NewGenerator creates a new generator with the given package name, name, and
// path to the TOML configuration file.
func NewGenerator(cfg *ParsedConfig) (*Generator, error) {
	return &Generator{
		pkg:                  cfg.GoConfig.Package,
		configTypeName:       cfg.GoConfig.ConfigTypeName,
		flagSetUsage:         cfg.GoConfig.FlagSetUsage,
		flagSetName:          cfg.GoConfig.FlagSetName,
		flagSetErrorHandling: cfg.GoConfig.FlagErrorHandling,
		args:                 cfg.Arguments,
		flags:                cfg.Flags,
	}, nil
}

// Execute generates the output in the given format and writes it to the given
// writer.
func (g *Generator) Execute(f Format, w io.Writer) error {
	switch f {
	case Go:
		return g.doGo(w)
	case Markdown:
		return g.doMarkdown(w)
	case HTML:
		return g.doHTML(w)
	default:
		return fmt.Errorf("unsupported format: %s", f)
	}
}

func (g *Generator) doGo(w io.Writer) error {
	// Parse the template.
	tmpl, err := template.New("flags").Parse(flagTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Perform some checks of the flags.
	for _, flag := range g.flags {
		if flag.Type == "time.Duration" {
			if flag.Default == nil {
				flag.Default = 0
			} else {
				s, ok := flag.Default.(string)
				if !ok {
					return fmt.Errorf("time.Duration flag %s has non-string default", flag.Name)
				}
				if _, err := time.ParseDuration(s); err != nil {
					return fmt.Errorf("time.Duration flag %s has invalid default: %v", flag.Name, err)
				}
			}
		}
	}

	// Execute the template with the flags data.
	var output bytes.Buffer
	if err := tmpl.Execute(&output, struct {
		Pkg             string
		FSUsage         string
		FSName          string
		FSErrorHandling string
		ConfigType      string
		Args            []Argument
		Flags           []Flag
	}{
		Pkg:             g.pkg,
		FSUsage:         g.flagSetUsage,
		FSName:          g.flagSetName,
		FSErrorHandling: g.flagSetErrorHandling,
		ConfigType:      g.configTypeName,
		Args:            g.args,
		Flags:           g.flags,
	}); err != nil {
		return err
	}

	// Format the Go source.
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format source: %w", err)
	}

	// Write the output to flags.go.
	_, err = w.Write(formatted)
	return err
}

func (g *Generator) doMarkdown(w io.Writer) error {
	// Write the markdown table header.
	_, err := w.Write([]byte("| Flag | Usage |\n|-|-|\n"))
	if err != nil {
		return err
	}

	// Write each flag as a row in the table.
	for _, flag := range g.flags {
		builder := strings.Builder{}
		builder.WriteString("|")
		builder.WriteString(escapeMarkdown(flag.CLI))
		builder.WriteString("|")
		builder.WriteString(escapeMarkdown(flag.ShortHelp))
		if flag.Default != nil {
			if !strings.HasSuffix(flag.ShortHelp, ".") {
				builder.WriteString(".")
			}
			builder.WriteString(fmt.Sprintf(" %s", escapeMarkdown(flag.LongHelp)))
		}
		builder.WriteString("|\n")
		_, err = w.Write([]byte(builder.String()))
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) doHTML(w io.Writer) error {
	// Parse the template.
	tmpl, err := template.New("htmlTable").Funcs(template.FuncMap{
		"html": func(s string) string {
			return template.HTMLEscapeString(s)
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("Error parsing HTML template: %v", err)
	}

	// Execute the template with the flags data.
	var output bytes.Buffer
	if err := tmpl.Execute(&output, struct {
		Flags []Flag
	}{Flags: g.flags}); err != nil {
		return fmt.Errorf("Error executing HTML template: %v", err)
	}

	if _, err := w.Write(output.Bytes()); err != nil {
		return fmt.Errorf("Error writing HTML file: %v", err)
	}
	return nil
}

// escapeMarkdown escapes markdown special characters.
func escapeMarkdown(text string) string {
	text = strings.ReplaceAll(text, "|", "\\|")
	text = strings.ReplaceAll(text, "\n", "<br>")
	return text
}
