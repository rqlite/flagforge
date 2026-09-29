package flagforge

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func Test_NewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Fatalf("expected non-nil parser")
	}
}

func Test_Generator_UsageEscapes(t *testing.T) {
	for _, tc := range []struct {
		name, toml, want string
	}{
		{"legacy literal", `'\nUsage:\n\texample [flags]\n'`, "\nUsage:\n\texample [flags]\n"},
		{"basic string", `"\nUsage:\n\texample [flags]\n"`, "\nUsage:\n\texample [flags]\n"},
		{"literal quotes", `'\nUse "example"\n'`, "\nUse \"example\"\n"},
		{"escaped quotes", `'\nUse \"example\"\n'`, "\nUse \"example\"\n"},
		{"literal backslash", `'Print \\n literally'`, `Print \n literally`},
		{"unknown escape", `'Keep \path'`, `Keep \path`},
		{"unicode and byte escapes", `'\u2192 \xff'`, "→ \xff"},
		{"multiline", "'''Usage:\n  example [flags]\n'''", "Usage:\n  example [flags]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewParser().ParseReader(strings.NewReader("[go]\nflag_set_usage = " + tc.toml + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			g, err := NewGenerator(cfg)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := g.Execute(Go, &output); err != nil {
				t.Fatal(err)
			}
			file, err := goparser.ParseFile(token.NewFileSet(), "generated.go", output.Bytes(), 0)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				fn, ok := call.Fun.(*ast.Ident)
				if !ok || fn.Name != "usage" {
					return true
				}
				found = true
				literal, ok := call.Args[0].(*ast.BasicLit)
				if !ok {
					t.Fatal("usage argument is not a string literal")
				}
				got, err := strconv.Unquote(literal.Value)
				if err != nil || got != tc.want {
					t.Errorf("usage text = %q, want %q; error: %v", got, tc.want, err)
				}
				return true
			})
			if !found {
				t.Fatal("no usage call generated")
			}
		})
	}
}

func Test_Generator_SingleArgument(t *testing.T) {
	toml := `
	[[arguments]]
	name = "DataDir"
	type = "string"
	required = true
	short_help = "Path to data directory"
	long_help = "Path to the directory where the node stores its data"
	`

	tomlFile := mustWriteToTempTOMLFile(toml)
	defer os.Remove(tomlFile)

	parser := NewParser()
	cfg, err := parser.ParsePath(tomlFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gen, err := NewGenerator(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tempFD := mustTempFD()
	defer os.Remove(tempFD.Name())
	defer tempFD.Close()
	err = gen.Execute(Go, tempFD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_Generator_SingleFlag(t *testing.T) {
	toml := `
	[[flags]]
	name = "NodeID"
	cli = "node-id"
	type = "string"
	default = ""
	short_help = "Node ID"
	long_help = "Unique node identifier"
	`

	tomlFile := mustWriteToTempTOMLFile(toml)
	defer os.Remove(tomlFile)

	parser := NewParser()
	cfg, err := parser.ParsePath(tomlFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gen, err := NewGenerator(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tempFD := mustTempFD()
	defer os.Remove(tempFD.Name())
	defer tempFD.Close()
	err = gen.Execute(Go, tempFD)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_Generator_GoldenFiles(t *testing.T) {
	for _, f := range []struct {
		in  string
		out string
	}{
		{
			in:  "single-flag/in.toml",
			out: "single-flag/out.go",
		},
		{
			in:  "multi-flag/in.toml",
			out: "multi-flag/out.go",
		},
		{
			in:  "multi-argument-flag/in.toml",
			out: "multi-argument-flag/out.go",
		},
		{
			in:  "rqlite/in.toml",
			out: "rqlite/out.go",
		},
		{
			in:  "hide/in.toml",
			out: "hide/out.go",
		},
		{
			in:  "edge-cases/in.toml",
			out: "edge-cases/out.go",
		},
	} {
		in := "testdata/" + f.in
		out := "testdata/" + f.out

		parser := NewParser()
		cfg, err := parser.ParsePath(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		gen, err := NewGenerator(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		buf := new(bytes.Buffer)
		err = gen.Execute(Go, buf)
		if err != nil {
			t.Fatalf("unexpected error testing %s: %v", in, err)
		}

		if !bytes.Equal(buf.Bytes(), mustReadFile(out)) {
			t.Errorf("generated output does not match %s\n", out)
			fmt.Println(buf.String())
			t.Fatal()
		}
	}
}

func Test_Generator_HTMLGoldenFiles(t *testing.T) {
	for _, f := range []struct {
		in  string
		out string
	}{
		{
			in:  "single-flag/in.toml",
			out: "single-flag/out.html",
		},
		{
			in:  "sections/in.toml",
			out: "sections/out.html",
		},
		{
			in:  "hide/in.toml",
			out: "hide/out.html",
		},
		{
			in:  "edge-cases/in.toml",
			out: "edge-cases/out.html",
		},
	} {
		in := "testdata/" + f.in
		out := "testdata/" + f.out

		parser := NewParser()
		cfg, err := parser.ParsePath(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		gen, err := NewGenerator(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		buf := new(bytes.Buffer)
		if err := gen.Execute(HTML, buf); err != nil {
			t.Fatalf("unexpected error testing %s: %v", in, err)
		}

		if !bytes.Equal(buf.Bytes(), mustReadFile(out)) {
			t.Errorf("generated output does not match %s\n", out)
			fmt.Println(buf.String())
			t.Fatal()
		}
	}
}

func Test_Generator_MarkdownGoldenFiles(t *testing.T) {
	for _, name := range []string{"single-flag", "sections", "hide", "edge-cases"} {
		t.Run(name, func(t *testing.T) {
			cfg, err := NewParser().ParsePath("testdata/" + name + "/in.toml")
			if err != nil {
				t.Fatal(err)
			}
			g, err := NewGenerator(cfg)
			if err != nil {
				t.Fatal(err)
			}
			var b bytes.Buffer
			if err := g.Execute(Markdown, &b); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b.Bytes(), mustReadFile("testdata/"+name+"/out.md")) {
				t.Fatalf("unexpected Markdown output:\n%s", b.String())
			}
		})
	}
}

// Test_Generator_SectionsIgnoredByGo checks that adding sections to a
// configuration file has no effect on the generated Go code.
func Test_Generator_SectionsIgnoredByGo(t *testing.T) {
	withSections := `
	[[flags]]
	name = "NodeID"
	cli = "node-id"
	type = "string"
	default = ""
	short_help = "Node ID"
	section = "General"

	[[flags]]
	name = "HTTPAddr"
	cli = "http-addr"
	type = "string"
	default = "localhost:4001"
	short_help = "HTTP server bind address"
	section = "HTTP API"
	`
	withoutSections := strings.ReplaceAll(
		strings.ReplaceAll(withSections, "\tsection = \"General\"\n", ""),
		"\tsection = \"HTTP API\"\n", "")

	generate := func(toml string) []byte {
		tomlFile := mustWriteToTempTOMLFile(toml)
		defer os.Remove(tomlFile)

		parser := NewParser()
		cfg, err := parser.ParsePath(tomlFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		gen, err := NewGenerator(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		buf := new(bytes.Buffer)
		if err := gen.Execute(Go, buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return buf.Bytes()
	}

	if !bytes.Equal(generate(withSections), generate(withoutSections)) {
		t.Fatal("sections changed the generated Go code")
	}
}

func Test_GroupBySection(t *testing.T) {
	flags := func(sections ...string) []Flag {
		var f []Flag
		for i, s := range sections {
			f = append(f, Flag{CLI: fmt.Sprintf("flag-%d", i), Section: s})
		}
		return f
	}

	t.Run("NoSections", func(t *testing.T) {
		got, err := groupBySection(flags("", "", ""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 section, got %d", len(got))
		}
		if got[0].Name != "" {
			t.Errorf("expected anonymous section, got %q", got[0].Name)
		}
		if len(got[0].Flags) != 3 {
			t.Errorf("expected 3 flags, got %d", len(got[0].Flags))
		}
	})

	t.Run("PartialSections", func(t *testing.T) {
		_, err := groupBySection(flags("General", "", "General"))
		if err == nil {
			t.Fatal("expected an error for a partially sectioned config")
		}
		if !strings.Contains(err.Error(), "flag-1") {
			t.Errorf("error should name the unassigned flag, got %q", err.Error())
		}
	})

	t.Run("OrderOfFirstAppearance", func(t *testing.T) {
		// "Clustering" appears before "General" is repeated, so the section
		// order follows first appearance, not the order flags are listed in.
		got, err := groupBySection(flags("General", "Clustering", "General"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 sections, got %d", len(got))
		}
		if got[0].Name != "General" || got[1].Name != "Clustering" {
			t.Fatalf("unexpected section order: %q, %q", got[0].Name, got[1].Name)
		}
		if len(got[0].Flags) != 2 {
			t.Errorf("expected 2 flags in General, got %d", len(got[0].Flags))
		}
		if got[0].Flags[0].CLI != "flag-0" || got[0].Flags[1].CLI != "flag-2" {
			t.Errorf("General holds the wrong flags: %v", got[0].Flags)
		}
	})
}

func mustWriteToTempTOMLFile(contents string) string {
	f, err := os.CreateTemp("", "generator_test-*.toml")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err := f.WriteString(contents); err != nil {
		panic(err)
	}
	return f.Name()
}

func mustTempFD() *os.File {
	f, err := os.CreateTemp("", "generator_test")
	if err != nil {
		panic(err)
	}
	return f
}

func mustReadFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}
