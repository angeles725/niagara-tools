package rcscan

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUnfingerprintedDedup: a reference repeated on one line (the same URL twice, or as an attribute and a
// url()) is one entry, as staticRows reports it, and the same URL on another line is another.
func TestUnfingerprintedDedup(t *testing.T) {
	t.Parallel()
	rc := filepath.Join(t.TempDir(), "rc")
	if err := os.MkdirAll(rc, 0o755); err != nil {
		t.Fatal(err)
	}
	page := "<img src=\"img/a.png\"> <img src=\"img/a.png\"> <i style=\"background:url(img/a.png)\"></i>\n" +
		"<img src=\"img/a.png\"> <img src=\"img/b.png?v=\"> <img src=\"img/c.png?v=1\">\n"
	if err := os.WriteFile(filepath.Join(rc, "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Unfingerprinted(rc, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []RefFile{{"rc/index.html", 1, "img/a.png"}, {"rc/index.html", 2, "img/a.png"}, {"rc/index.html", 2, "img/b.png?v="}}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
