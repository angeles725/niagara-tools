package exitdoc

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCodesAreTheExitIntegerConstantsInFileOrder(t *testing.T) {
	src := "package p\nconst (\n\tExitOK = 0 // fine\n\tExitEnv = 10\n\tnotExit = 3\n\tExitName = \"x\"\n)\nconst ExitLast, other = 54, 1\nvar ExitVar = 9\n"
	file := filepath.Join(t.TempDir(), "exit.go")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Codes(file)
	want := []Code{{"ExitOK", 0}, {"ExitEnv", 10}, {"ExitLast", 54}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Codes = %v, %v; want %v", got, err, want)
	}
	if _, err := Codes(filepath.Join(t.TempDir(), "missing.go")); err == nil {
		t.Error("a missing file is no error")
	}
}
