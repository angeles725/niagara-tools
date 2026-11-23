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

// A directory that cannot be listed is the list error (EACCES), never an empty list: the jars it holds are
// not known, and a caller that reads "no jars" from it would ship or lock-check nothing without saying so.
func TestAnUnlistableDirectoryIsTheListErrorNotAnEmptyList(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a directory without permissions is still listed")
	}
	dir := fixture(t)
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	for name, list := range map[string]func(string) ([]string, error){"Names": Names, "Regular": Regular} {
		got, err := list(dir)
		if !errors.Is(err, syscall.EACCES) || len(got) != 0 {
			t.Errorf("%s(unlistable) = %v, %v; want no names and EACCES", name, got, err)
		}
	}
}

// A directory that is listed but not searched (r--): Names still names the entries, and Regular, which must
// stat each, keeps none, so a jar it cannot see is not one it counts.
func TestAListableButUnsearchableDirectoryShipsNothing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a directory without permissions is still searched")
	}
	dir := fixture(t)
	if err := os.Chmod(dir, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	if names, err := Names(dir); err != nil || len(names) != 5 {
		t.Errorf("Names = %v, %v; want the five names and no error", names, err)
	}
	if files, err := Regular(dir); err != nil || len(files) != 0 {
		t.Errorf("Regular = %v, %v; want no jar and no error", files, err)
	}
}

// The layout the three readers share is spelled once, here.
func TestTheSharedLayoutIsModulesAndDotJar(t *testing.T) {
	if Dir != "modules" || Suffix != ".jar" {
		t.Errorf("Dir %q, Suffix %q", Dir, Suffix)
	}
}
