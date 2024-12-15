package main

import (
	"flag"
	"fmt"
	"os"

	gen "github.com/rqlite/flagforge"
)

func main() {
	var (
		formatStr string
		out       string
	)

	flag.StringVar(&formatStr, "f", "go", "output format: go|markdown|html")
	flag.StringVar(&out, "o", "", "output file")
	flag.Parse()

	if flag.NArg() < 1 {
		printExit("no input TOML file provided\n")
	}
	inputPath := flag.Arg(0)

	var f gen.Format
	switch formatStr {
	case "go":
		f = gen.Go
	case "markdown":
		f = gen.Markdown
	case "html":
		f = gen.HTML
	default:
		printExit("unknown format: %s\n", formatStr)
	}

	p := gen.NewParser()
	cfg, err := p.ParsePath(inputPath)
	if err != nil {
		printExit("failed to parse input file: %v\n", err)
	}

	g, err := gen.NewGenerator(cfg)
	if err != nil {
		printExit("failed to create generator: %v\n", err)
	}

	w := os.Stdout
	if out != "" {
		w, err = os.Create(out)
		if err != nil {
			printExit("failed to create output file: %v\n", err)
		}
		defer w.Close()
	}
	if err := g.Execute(f, w); err != nil {
		printExit("failed to generate output: %v\n", err)
	}
}

func printExit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
