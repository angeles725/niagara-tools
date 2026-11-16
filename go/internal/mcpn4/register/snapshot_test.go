package register

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// walkFiles calls f with each regular file under root and its bytes.
func walkFiles(root string, f func(string, []byte)) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		b, err := os.ReadFile(p)
		if err == nil {
			f(p, b)
		}
		return err
	})
}

func sameFiles(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// snapshot is every regular file under the fake home, by path, with its bytes.
func (h *fakeHome) snapshot(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := walkFiles(h.dir, func(p string, b []byte) { files[p] = string(b) })
	if err != nil {
		t.Fatal(err)
	}
	return files
}
