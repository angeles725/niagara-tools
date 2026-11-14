package gitcmdtest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServeFake turns the test binary into the fake when it was started under the name git, and returns
// at once otherwise. Call it before m.Run in TestMain.
func ServeFake() {
	if filepath.Base(os.Args[0]) != fakeName {
		return
	}
	os.Exit(serve())
}

// serve records this start beside the fake binary and plays the behaviour found there. A fake that
// cannot record or read its script exits 99, a status no test scripts.
func serve() int {
	self, err := os.Executable()
	if err != nil {
		return fakeBroken(err)
	}
	dir := filepath.Dir(self)
	cwd, err := os.Getwd()
	if err != nil {
		return fakeBroken(err)
	}
	call, err := json.Marshal(Call{Args: os.Args[1:], Env: Environ(), Dir: cwd})
	if err != nil {
		return fakeBroken(err)
	}
	n := 1
	for ; ; n++ {
		if _, err := os.Stat(callPath(dir, n)); err != nil {
			break
		}
	}
	if err := os.WriteFile(callPath(dir, n), call, 0o644); err != nil {
		return fakeBroken(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, behaviourFile))
	if err != nil {
		return fakeBroken(err)
	}
	var b Behaviour
	if err := json.Unmarshal(raw, &b); err != nil {
		return fakeBroken(err)
	}
	fmt.Fprint(os.Stdout, b.Stdout)
	fmt.Fprint(os.Stderr, b.Stderr)
	return b.Exit
}

func fakeBroken(err error) int {
	fmt.Fprintln(os.Stderr, "gitcmdtest fake:", err)
	return 99
}
