package gen

import "io"

type Format int

const (
	Go Format = iota
	Markdown
	HTML
)

type Generator struct {
	path string
}

func NewGenerator(path string) (*Generator, error) {
	return &Generator{
		path: path,
	}, nil
}

func (g *Generator) Do(fmt Format, w io.Writer) (n int64, err error) {
	return 0, nil
}
