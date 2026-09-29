package flagforge

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"
)

func testConfig() *ParsedConfig {
	return &ParsedConfig{
		GoConfig: GoConfig{Package: "example", ConfigTypeName: "Config", FlagSetName: "example", FlagErrorHandling: "ContinueOnError"},
		Flags:    []Flag{{Name: "Value", CLI: "value", Type: "string"}},
	}
}

func Test_NewGenerator_Validation(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*ParsedConfig)
		want string
	}{
		{"package", func(c *ParsedConfig) { c.GoConfig.Package = "bad-name" }, "package name"},
		{"type keyword", func(c *ParsedConfig) { c.GoConfig.ConfigTypeName = "type" }, "type name"},
		{"type collision", func(c *ParsedConfig) { c.GoConfig.ConfigTypeName = "Forge" }, "conflicts"},
		{"import collision", func(c *ParsedConfig) { c.GoConfig.ConfigTypeName = "flag" }, "conflicts"},
		{"parameter collision", func(c *ParsedConfig) { c.GoConfig.ConfigTypeName = "arguments" }, "conflicts"},
		{"error policy", func(c *ParsedConfig) { c.GoConfig.FlagErrorHandling = "Bogus" }, "error handling"},
		{"field name", func(c *ParsedConfig) { c.Flags[0].Name = "_" }, "field name"},
		{"duplicate field", func(c *ParsedConfig) { c.Flags = append(c.Flags, c.Flags[0]) }, "duplicate field"},
		{"argument field collision", func(c *ParsedConfig) { c.Arguments = []Argument{{Name: "Value", Type: "string"}} }, "duplicate field"},
		{"empty CLI", func(c *ParsedConfig) { c.Flags[0].CLI = "" }, "CLI name"},
		{"leading dash", func(c *ParsedConfig) { c.Flags[0].CLI = "-value" }, "CLI name"},
		{"equals", func(c *ParsedConfig) { c.Flags[0].CLI = "value=x" }, "CLI name"},
		{"whitespace", func(c *ParsedConfig) { c.Flags[0].CLI = "value\u00a0name" }, "CLI name"},
		{"duplicate CLI", func(c *ParsedConfig) { c.Flags = append(c.Flags, Flag{Name: "Other", CLI: "value", Type: "string"}) }, "duplicate CLI"},
		{"flag type", func(c *ParsedConfig) { c.Flags[0].Type = "float64" }, "unsupported type"},
		{"argument type", func(c *ParsedConfig) { c.Arguments = []Argument{{Name: "Count", Type: "int"}} }, "unsupported type"},
		{"argument order", func(c *ParsedConfig) {
			c.Arguments = []Argument{{Name: "First", Type: "string"}, {Name: "Second", Type: "string", Required: true}}
		}, "follows an optional"},
		{"string default", func(c *ParsedConfig) { c.Flags[0].Default = 123 }, "must be a string"},
		{"bool default", func(c *ParsedConfig) { c.Flags[0].Type = "bool"; c.Flags[0].Default = "false" }, "must be a boolean"},
		{"float default", func(c *ParsedConfig) { c.Flags[0].Type = "int"; c.Flags[0].Default = 1.5 }, "must be an integer"},
		{"unsigned default", func(c *ParsedConfig) { c.Flags[0].Type = "uint64"; c.Flags[0].Default = -1 }, "out of range"},
		{"overflow", func(c *ParsedConfig) { c.Flags[0].Type = "int64"; c.Flags[0].Default = uint64(math.MaxUint64) }, "out of range"},
		{"duration default", func(c *ParsedConfig) { c.Flags[0].Type = "time.Duration"; c.Flags[0].Default = "invalid" }, "invalid duration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			tc.edit(cfg)
			if _, err := NewGenerator(cfg); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted error containing %q, got %v", tc.want, err)
			}
		})
	}
	if _, err := NewGenerator(nil); err == nil {
		t.Fatal("nil configuration should return an error")
	}
}

func Test_Generator_IndependentAndRepeatable(t *testing.T) {
	cfg := testConfig()
	cfg.Flags = []Flag{
		{Name: "Interval", CLI: "interval", Type: "time.Duration"},
		{Name: "List", CLI: "list", Type: "[]string", LongHelp: "Details without a default"},
		{Name: "Hidden", Type: "time.Duration", Default: "unused invalid duration", Hide: true},
	}
	want := append([]Flag(nil), cfg.Flags...)
	g, err := NewGenerator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	generate := func(format Format) string {
		t.Helper()
		var b bytes.Buffer
		if err := g.Execute(format, &b); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	before := generate(Markdown)
	first := generate(Go)
	if first != generate(Go) || before != generate(Markdown) {
		t.Fatal("output depends on earlier generation calls")
	}
	if !reflect.DeepEqual(cfg.Flags, want) {
		t.Fatalf("caller configuration changed: %+v", cfg.Flags)
	}
	cfg.Flags[0].Name = "Changed"
	if first != generate(Go) {
		t.Fatal("generator retained the caller's flags slice")
	}
	if !strings.Contains(before, "Details without a default") {
		t.Fatal("Markdown omitted help without a default")
	}
}
