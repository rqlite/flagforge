package flagforge

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

func Test_NewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Fatalf("expected non-nil parser")
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
	cli = "-node-id"
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
