package stationpull

import (
	"hash/crc32"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func put(t *testing.T, root, rel, body string) uint32 {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return crc32.ChecksumIEEE([]byte(body))
}

func TestVerifyMatchesByNameIncludingNamesWithSpaces(t *testing.T) {
	root := t.TempDir()
	a := put(t, root, "config.bog", "alpha")
	b := put(t, root, "sub/my file.bog", "bravo")
	n, problems := Verify(root, []Row{{"config.bog", a}, {"my file.bog", b}, {"sub/my file.bog", b}})
	if n != 3 || len(problems) != 0 {
		t.Fatalf("verified %d problems %v", n, problems)
	}
}

func TestVerifyNamesAMissingFileAndACRCMismatch(t *testing.T) {
	root := t.TempDir()
	a := put(t, root, "a.txt", "alpha")
	put(t, root, "b.txt", "bravo")
	n, problems := Verify(root, []Row{{"a.txt", a}, {"b.txt", 12345}, {"gone.txt", 7}, {"gone zero.txt", 0}})
	want := []string{"crc mismatch: b.txt (listed 12345, copied " + strconv.FormatUint(uint64(crc32.ChecksumIEEE([]byte("bravo"))), 10) + ")",
		"missing: gone zero.txt", "missing: gone.txt"}
	if n != 1 || !reflect.DeepEqual(problems, want) {
		t.Fatalf("verified %d problems\n got %q\nwant %q", n, problems, want)
	}
}

func TestVerifyAZeroCRCNeedsTheFileButIsNotCounted(t *testing.T) {
	root := t.TempDir()
	put(t, root, "z.txt", "zed")
	if n, problems := Verify(root, []Row{{"z.txt", 0}}); n != 0 || len(problems) != 0 {
		t.Fatalf("verified %d problems %v", n, problems)
	}
}

// the same name in two directories (a daemon lists by name) is matched one row to one copied file.
func TestVerifyDuplicateNamesMatchOneRowPerFile(t *testing.T) {
	root := t.TempDir()
	a := put(t, root, "x/config.bog", "one")
	b := put(t, root, "y/config.bog", "two")
	if n, problems := Verify(root, []Row{{"config.bog", b}, {"config.bog", a}}); n != 2 || len(problems) != 0 {
		t.Fatalf("verified %d problems %v", n, problems)
	}
	if n, problems := Verify(root, []Row{{"config.bog", a}, {"config.bog", a}}); n != 1 || len(problems) != 1 {
		t.Fatalf("verified %d problems %v", n, problems)
	}
}
