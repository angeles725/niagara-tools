package javasrc

import (
	"reflect"
	"testing"
)

func TestFile(t *testing.T) {
	raw := []string{"class A { // c", "  void f() {", "    x(); /* open"}
	f := NewFile("src/A.java", raw)
	if f.Path != "src/A.java" || !reflect.DeepEqual(f.Raw, raw) {
		t.Fatalf("NewFile kept %q %q", f.Path, f.Raw)
	}
	wantStripped, wantEnd := Strip(raw)
	if got := f.Stripped(); !reflect.DeepEqual(got, wantStripped) {
		t.Errorf("Stripped() = %q, want %q", got, wantStripped)
	}
	if got := f.End(); got != wantEnd || got != OpenComment {
		t.Errorf("End() = %v, want %v", got, wantEnd)
	}
	if got, want := f.Methods(), Methods(wantStripped); !reflect.DeepEqual(got, want) {
		t.Errorf("Methods() = %v, want %v", got, want)
	}
	// Computed once: the cached slice is returned on every call.
	if a, b := f.Stripped(), f.Stripped(); &a[0] != &b[0] {
		t.Error("Stripped() recomputed instead of returning the cached lines")
	}
}

func TestFileEmpty(t *testing.T) {
	f := NewFile("E.java", nil)
	if len(f.Stripped()) != 0 || f.End() != Closed || len(f.Methods()) != 0 {
		t.Errorf("empty file: %q %v %v", f.Stripped(), f.End(), f.Methods())
	}
}
