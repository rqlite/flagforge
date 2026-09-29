package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	gen "github.com/rqlite/flagforge"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run keeps CLI behavior testable without starting another process.
func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("flagforge", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	formatStr := fs.String("f", "go", "output format: go|markdown|html")
	out := fs.String("o", "", "output file")
	header := fs.String("p", "", "path to a file to copy to the output before the generated content")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(stderr)
			fs.Usage()
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("expected exactly one input TOML file, got %d", fs.NArg())
	}
	var format gen.Format
	switch *formatStr {
	case "go":
		format = gen.Go
	case "markdown":
		format = gen.Markdown
	case "html":
		format = gen.HTML
	default:
		return fmt.Errorf("unknown format: %s", *formatStr)
	}
	cfg, err := gen.NewParser().ParsePath(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("failed to parse input file: %w", err)
	}
	g, err := gen.NewGenerator(cfg)
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}
	// Read and render everything before touching an existing destination. This
	// also allows a prefix file to be the output file.
	var output bytes.Buffer
	if *header != "" {
		b, err := os.ReadFile(*header)
		if err != nil {
			return fmt.Errorf("failed to read header file: %w", err)
		}
		output.Write(b)
	}
	if err := g.Execute(format, &output); err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}
	if *out != "" {
		return writeOutput(*out, output.Bytes())
	}
	if _, err := output.WriteTo(stdout); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

// writeOutput replaces the destination only after a successful write and close.
func writeOutput(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect output file: %w", err)
	}
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("output is not a regular file: %s", path)
		}
		// Preserve the behavior of writing through existing symbolic links.
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("failed to resolve output file: %w", err)
		}
	}
	// A private temporary directory lets the file use normal creation
	// permissions (0666 masked by the process umask) without a name collision.
	dir, err := os.MkdirTemp(filepath.Dir(path), ".flagforge-*")
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	defer os.RemoveAll(dir)
	f, err := os.OpenFile(filepath.Join(dir, "output"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()
	if info != nil {
		if err := f.Chmod(info.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to preserve output permissions: %w", err)
		}
	}
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close output: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("failed to replace output: %w", err)
	}
	return nil
}
