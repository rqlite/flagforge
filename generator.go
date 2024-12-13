package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"strings"
	"text/template"

	"github.com/spf13/viper"
)

const flagTemplate = `
// Code generated by go generate; DO NOT EDIT.
package {{ .Pkg }}

import (
	"flag"
	{{- if .Args }}
	"fmt"
	{{- end }}
	{{- range .Flags }}
	{{- if eq .Type "time.Duration" }}
	"time"
	{{- end }}
	{{- end }}
)

// Config represents all configuration options.
type Config struct {
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
func Forge(arguments []string) (*flag.FlagSet, *Config, error) {
	config := &Config{}
	fs := flag.NewFlagSet("{{ .Name }}", flag.ExitOnError)
{{- range $index, $element := .Args }}
	if len(arguments) < {{ $index }} {
		return nil, nil, fmt.Errorf("missing required argument: {{ $element.Name }}")
	}
	{{- if eq .Type "string" }}
	config.{{ .Name }} = arguments[{{ $index }}]
	{{- end }}
{{- end }}
{{- range .Flags }}
	{{- if eq .Type "string" }}
	fs.StringVar(&config.{{ .Name }}, "{{ .CLI }}", "{{ .Default }}", "{{ .ShortHelp }}")
	{{- else if eq .Type "bool" }}
	fs.BoolVar(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- else if eq .Type "int" }}
	fs.IntVar(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- else if eq .Type "time.Duration" }}
	fs.DurationVar(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- end }}
{{- end }}
    if err := fs.Parse(arguments); err != nil {
	    return nil, nil, err
    }
	return fs, config, nil
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

// Argument represents a single argument configuration.
type Argument struct {
	Name      string `mapstructure:"name"`
	Type      string `mapstructure:"type"`
	Required  bool   `mapstructure:"required"`
	ShortHelp string `mapstructure:"short_help"`
	LongHelp  string `mapstructure:"long_help"`
}

// Flag represents a single flag configuration.
type Flag struct {
	Name      string      `mapstructure:"name"`
	CLI       string      `mapstructure:"cli"`
	Type      string      `mapstructure:"type"`
	Default   interface{} `mapstructure:"default"`
	ShortHelp string      `mapstructure:"short_help"`
	LongHelp  string      `mapstructure:"long_help"`
}

// Generator represents a flag, HTML, or Markdown generator.
type Generator struct {
	pkg  string
	name string
	path string

	args  []Argument
	flags []Flag
}

// NewGenerator creates a new generator with the given package name, name, and
// path to the TOML configuration file.
func NewGenerator(pkg, name, path string) (*Generator, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read TOML file: %w", err)
	}

	var args []Argument
	if err := viper.UnmarshalKey("arguments", &args); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}
	var flags []Flag
	if err := viper.UnmarshalKey("flags", &flags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flags: %w", err)
	}

	return &Generator{
		pkg:   pkg,
		name:  name,
		path:  path,
		args:  args,
		flags: flags,
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

	// Execute the template with the flags data.
	var output bytes.Buffer
	if err := tmpl.Execute(&output, struct {
		Pkg   string
		Name  string
		Args  []Argument
		Flags []Flag
	}{
		Pkg:   g.pkg,
		Name:  g.name,
		Args:  g.args,
		Flags: g.flags,
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
			builder.WriteString(fmt.Sprintf("\n%s", escapeMarkdown(flag.LongHelp)))
		}
		builder.WriteString("|")
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
