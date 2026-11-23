package modulejars

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

// fixture is a modules/ directory holding every kind of entry the three readers once judged on their own.
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("fox-rt.jar")
	write("baja.jar")
	write(".hidden.jar")
	write("readme.txt")
	write("fox-rt.jar.bak")
	if err := os.Mkdir(filepath.Join(dir, "dir.jar"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("baja.jar", filepath.Join(dir, "alias.jar")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nowhere.jar", filepath.Join(dir, "dangling.jar")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNamesAreTheNonDotJarEntriesInNameOrder(t *testing.T) {
	got, err := Names(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alias.jar", "baja.jar", "dangling.jar", "dir.jar", "fox-rt.jar"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Names = %v, want %v", got, want)
	}
}

func TestRegularDropsADirectoryAndADanglingLinkButKeepsALinkToAFile(t *testing.T) {
	got, err := Regular(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alias.jar", "baja.jar", "fox-rt.jar"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Regular = %v, want %v", got, want)
	}
}

func TestAMissingDirectoryIsTheListErrorNotAnEmptyList(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "modules")
	for name, list := range map[string]func(string) ([]string, error){"Names": Names, "Regular": Regular} {
		got, err := list(missing)
		if !errors.Is(err, syscall.ENOENT) || len(got) != 0 {
			t.Errorf("%s(missing) = %v, %v; want no names and ENOENT", name, got, err)
		}
	}
}

func TestAFileInPlaceOfTheDirectoryIsENOTDIR(t *testing.T) {
	file := filepath.Join(t.TempDir(), "modules")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Names(file); !errors.Is(err, syscall.ENOTDIR) {
		t.Errorf("Names(file) error = %v, want ENOTDIR", err)
	}
}
