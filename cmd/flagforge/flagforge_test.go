package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validInput = `[[flags]]
name = "Value"
cli = "value"
type = "string"
short_help = "A value"
`

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0640); err != nil {
		t.Fatal(err)
	}
}

func Test_Run_PreservesOutputOnError(t *testing.T) {
	for _, failure := range []string{"missing prefix", "invalid configuration", "invalid sections", "extra input", "unknown format"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			input, output := filepath.Join(dir, "in.toml"), filepath.Join(dir, "out")
			writeTestFile(t, input, validInput)
			writeTestFile(t, output, "original output")
			args := []string{"-o", output}
			switch failure {
			case "missing prefix":
				args = append(args, "-p", filepath.Join(dir, "missing"))
			case "invalid configuration":
				writeTestFile(t, input, strings.ReplaceAll(validInput, "name = \"Value\"", "name = \"bad-name\""))
			case "invalid sections":
				writeTestFile(t, input, validInput+"section = 'One'\n[[flags]]\nname = 'Other'\ncli = 'other'\ntype = 'string'\n")
				args = append(args, "-f", "html")
			case "unknown format":
				args = append(args, "-f", "unknown")
			}
			args = append(args, input)
			if failure == "extra input" {
				args = append(args, "extra.toml")
			}
			var stdout bytes.Buffer
			if err := run(args, &stdout, io.Discard); err == nil {
				t.Fatal("expected error")
			}
			b, err := os.ReadFile(output)
			if err != nil || string(b) != "original output" || stdout.Len() != 0 {
				t.Fatalf("output changed on failure: %q, stdout=%q, err=%v", b, stdout.String(), err)
			}
		})
	}
}

func Test_Run_PrefixCanBeOutput(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "in.toml"), filepath.Join(dir, "out.md")
	writeTestFile(t, input, validInput)
	writeTestFile(t, output, "Introduction\n\n")
	if err := run([]string{"-f", "html", "-p", output, "-o", output, input}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(output)
	if err != nil || !strings.HasPrefix(string(b), "Introduction\n\n<table") {
		t.Fatalf("prefix lost: %q, %v", b, err)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("output permissions changed: %v, %v", info, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".flagforge-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary output files remain: %v, %v", matches, err)
	}
}

func Test_Run_FormatsAndErrors(t *testing.T) {
	input := filepath.Join(t.TempDir(), "in.toml")
	writeTestFile(t, input, validInput)
	for _, format := range []string{"go", "markdown", "html"} {
		var stdout bytes.Buffer
		if err := run([]string{"-f", format, input}, &stdout, io.Discard); err != nil {
			t.Fatal(err)
		}
		if stdout.Len() == 0 {
			t.Fatalf("no %s output", format)
		}
	}
	var help bytes.Buffer
	if err := run([]string{"-h"}, io.Discard, &help); err != nil || !strings.Contains(help.String(), "output format") {
		t.Fatalf("help failed: %v, %q", err, help.String())
	}
	for _, args := range [][]string{nil, {"-unknown"}, {"-o"}, {input, "-f", "html"}} {
		if err := run(args, io.Discard, io.Discard); err == nil {
			t.Fatalf("expected an error for %v", args)
		}
	}
	want := errors.New("write failure")
	if err := run([]string{input}, failingWriter{want}, io.Discard); !errors.Is(err, want) {
		t.Fatalf("write error was lost: %v", err)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func Test_WriteOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out")
	if err := writeOutput(path, []byte("new output")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "new output" {
		t.Fatalf("incorrect output: %q, %v", b, err)
	}
	if err := writeOutput(dir, []byte("must not replace directory")); err == nil {
		t.Fatal("expected directory output to fail")
	}
	if err := writeOutput(filepath.Join(dir, "missing", "out"), nil); err == nil {
		t.Fatal("expected missing parent directory to fail")
	}
}

func Test_WriteOutput_Symlink(t *testing.T) {
	dir := t.TempDir()
	target, link := filepath.Join(dir, "target"), filepath.Join(dir, "link")
	writeTestFile(t, target, "original")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := writeOutput(link, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(target)
	if err != nil || string(b) != "replacement" {
		t.Fatalf("target was not updated: %q, %v", b, err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink was replaced: %v, %v", info, err)
	}
}
