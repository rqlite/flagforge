package gen

import "testing"

func Test_NewGenerator(t *testing.T) {
	gen, err := NewGenerator("pkg", "name", "path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gen == nil {
		t.Fatalf("expected non-nil generator")
	}
}
