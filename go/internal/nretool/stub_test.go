package nretool

import (
	"os"
	"path/filepath"
	"testing"
)

// stubHome is a Niagara home whose bin/<exe> is the shell body: it records its working directory and arguments in
// args.txt first. A fake wslpath (prints W:<path>) leads PATH so the Windows-path conversion is deterministic.
func stubHome(t *testing.T, exe, body string) (home, argsFile string) {
	t.Helper()
	home = t.TempDir()
	argsFile = filepath.Join(home, "args.txt")
	write(t, filepath.Join(home, "bin", exe), "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > '"+argsFile+"'\n"+body+"\n", 0o755)
	tools := t.TempDir()
	write(t, filepath.Join(tools, "wslpath"), "#!/bin/sh\nprintf 'W:%s\\n' \"$2\"\n", 0o755)
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	return home, argsFile
}

func write(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

func readFile(path string) (string, error) { b, err := os.ReadFile(path); return string(b), err }
