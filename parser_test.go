package flagforge

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func Test_Parser_ReaderMatchesPath(t *testing.T) {
	const input = `[Go]
package = "example"
[[flags]]
name = "Enabled"
cli = "enabled"
type = "bool"
default = false
hide = true
[[arguments]]
name = "Path"
type = "string"
required = true
`
	path := filepath.Join(t.TempDir(), "config") // TOML does not require a filename extension.
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	p := NewParser()
	fromPath, err := p.ParsePath(path)
	if err != nil {
		t.Fatal(err)
	}
	fromReader, err := p.ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromPath, fromReader) {
		t.Fatalf("path and reader differ: %+v / %+v", fromPath, fromReader)
	}
	if len(fromReader.Flags) != 1 || !fromReader.Flags[0].Hide || len(fromReader.Arguments) != 1 || !fromReader.Arguments[0].Required {
		t.Fatalf("configuration was not read: %+v", fromReader)
	}
	if fromReader.GoConfig.Package != "example" || fromReader.GoConfig.ConfigTypeName != "Config" || fromReader.GoConfig.FlagErrorHandling != "ExitOnError" {
		t.Fatalf("incorrect Go settings: %+v", fromReader.GoConfig)
	}
	// The same parser must not retain the previous input's configuration.
	empty, err := p.ParseReader(strings.NewReader(""))
	if err != nil || len(empty.Flags) != 0 || empty.GoConfig.Package != "pkg" {
		t.Fatalf("parser retained state: %+v, %v", empty, err)
	}
}

func Test_Parser_RejectsInvalidInput(t *testing.T) {
	for _, input := range []string{
		"[broken",
		"flgas = []",
		"[go]\npackge = 'pkg'",
		"[[flags]]\nhidde = true",
		"[[arguments]]\nrequred = true",
		"[[flags]]\nhide = 'true'",
		"[[flags]]\nname = 123",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := NewParser().ParseReader(strings.NewReader(input)); err == nil {
				t.Fatal("expected malformed input, unknown key, or invalid type to be rejected")
			}
		})
	}
}
