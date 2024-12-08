package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"text/template"

	"github.com/spf13/viper"
)

const flagTemplate = `
// Code generated by go generate; DO NOT EDIT.
package {{ .Pkg }}

import (
	"flag"
)

// Config represents all configuration options.
type Config struct {
{{- range .Flags }}
	// {{ .ShortHelp }}
	{{ .Name }} {{ .GoType }}
{{- end }}
}

// Forge sets up and parses command-line flags.
func Forge(arguments []string) (*flag.FlagSet, *Config, error) {
	config := &Config{}
	fs := flag.NewFlagSet("{{ .Name }}", flag.ExitOnError)
{{- range .Flags }}
	{{- if eq .Type "string" }}
	fs.StringVar(&config.{{ .Name }}, "{{ .CLI }}", "{{ .Default }}", "{{ .ShortHelp }}")
	{{- else if eq .Type "bool" }}
	fs.BoolVar(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- else if eq .Type "int" }}
	fs.IntVar(&config.{{ .Name }}, "{{ .CLI }}", {{ .Default }}, "{{ .ShortHelp }}")
	{{- end }}
{{- end }}
    if err := fs.Parse(arguments); err != nil {
	    return nil, nil, err
    }
	return fs, config, nil
}

`

type Format int

const (
	Go Format = iota
	Markdown
	HTML
)

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

// Flag represents a single flag configuration.
type Flag struct {
	Name      string      `mapstructure:"name"`
	CLI       string      `mapstructure:"cli"`
	Type      string      `mapstructure:"type"`
	Default   interface{} `mapstructure:"default"`
	ShortHelp string      `mapstructure:"short_help"`
	LongHelp  string      `mapstructure:"long_help"`
}

// GoType converts the flag type to Go type.
func (f Flag) GoType() string {
	switch f.Type {
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "int":
		return "int"
	case "uint64":
		return "uint64"
	case "time.Duration":
		return "time.Duration"
	default:
		panic(fmt.Sprintf("unknown type: %s", f.Type))
	}
}

type Generator struct {
	pkg  string
	name string
	path string

	flags []Flag
}

func NewGenerator(pkg, name, path string) (*Generator, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var flags []Flag
	if err := viper.UnmarshalKey("flags", &flags); err != nil {
		return nil, err
	}

	return &Generator{
		pkg:   pkg,
		name:  name,
		path:  path,
		flags: flags,
	}, nil
}

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
	tmpl, err := template.New("flags").Funcs(template.FuncMap{
		"GoType": Flag.GoType,
	}).Parse(flagTemplate)
	if err != nil {
		return err
	}

	// Execute the template with the flags data.
	var output bytes.Buffer
	if err := tmpl.Execute(&output, struct {
		Pkg   string
		Name  string
		Flags []Flag
	}{
		Pkg:   g.pkg,
		Name:  g.name,
		Flags: g.flags,
	}); err != nil {
		return err
	}

	// Format the Go source.
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return err
	}

	// Write the output to flags.go.
	_, err = w.Write(formatted)
	return err
}

func (g *Generator) doMarkdown(w io.Writer) error {
	_ = w
	return nil
}

func (g *Generator) doHTML(w io.Writer) error {
	_ = w
	return nil
}
