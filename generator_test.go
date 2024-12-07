package gen

import (
	"os"
	"testing"
)

func Test_NewGenerator(t *testing.T) {
	gen, err := NewGenerator("pkg", "name", "path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gen == nil {
		t.Fatalf("expected non-nil generator")
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

	gen, err := NewGenerator("pkg", "name", tomlFile)
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
	tempFD.Close()
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
