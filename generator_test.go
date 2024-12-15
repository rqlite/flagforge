package gen

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func Test_NewGenerator(t *testing.T) {
	path := mustWriteToTempTOMLFile("")
	defer os.Remove(path)
	gen, err := NewGenerator("pkg", "name", "Config", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gen == nil {
		t.Fatalf("expected non-nil generator")
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

	gen, err := NewGenerator("pkg", "name", "Config", tomlFile)
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

	gen, err := NewGenerator("pkg", "name", "Config", tomlFile)
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
	} {
		in := "testdata/" + f.in
		out := "testdata/" + f.out

		gen, err := NewGenerator("pkg", "name", "Config", in)
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
