package prepush

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guard installs a client-guard that records its arguments and stdin and exits rc.
func guard(r *repo, rc string) (log string) {
	log = filepath.Join(filepath.Dir(r.dir), "guard")
	r.tool("client-guard", `printf 'args: %s\n' "$*" > '`+log+`'; cat >> '`+log+`'; echo guard-out; echo guard-err >&2; exit `+rc)
	return log
}

func TestClientGuardGetsTheHookArgumentsAndEveryRefLine(t *testing.T) {
	r := newRepo(t)
	log := guard(r, "0")
	res := r.push("(delete) "+zeroSha+" refs/heads/a aaaa", "(delete) "+zeroSha+" refs/heads/b bbbb")
	res.want(t, 0, "guard-err")
	if res.out != "guard-out\n" {
		t.Errorf("the guard's stdout is not passed on: %q", res.out)
	}
	got, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	want := "args: origin https://example.invalid/repo.git\n" +
		"(delete) " + zeroSha + " refs/heads/a aaaa\n(delete) " + zeroSha + " refs/heads/b bbbb\n"
	if string(got) != want {
		t.Errorf("client-guard saw\n%s\nwant\n%s", got, want)
	}
}

func TestClientGuardRefusalRefusesBeforeAnyOtherGate(t *testing.T) {
	for _, rc := range []string{"1", "2"} {
		r := newRepo(t)
		guard(r, rc)
		// a deletion passes every other gate: only the guard can refuse it
		res := r.push("(delete) " + zeroSha + " refs/heads/a aaaa")
		res.want(t, 1)
		if strings.Contains(res.err, "pre-push:") {
			t.Errorf("guard exit %s: the hook adds a line of its own: %s", rc, res.err)
		}
	}
}

func TestNoClientGuardWithADenylistRefuses(t *testing.T) {
	r := newRepo(t)
	if err := os.WriteFile(os.Getenv("NIAGARA_CLIENT_DENYLIST"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	res := r.push("(delete) " + zeroSha + " refs/heads/a aaaa")
	want := "pre-push: client-guard is not on PATH but a client denylist exists; refusing to push unchecked.\n" +
		"          install it from a niagara-go checkout: GOBIN=$HOME/.local/bin go install ./cmd/client-guard\n"
	if res.code != 1 || res.err != want {
		t.Errorf("exit %d, stderr\n%s\nwant 1 and\n%s", res.code, res.err, want)
	}
}

func TestNoClientGuardWithTheDefaultDenylistRefuses(t *testing.T) {
	r := newRepo(t)
	t.Setenv("NIAGARA_CLIENT_DENYLIST", "") // unset or empty: $HOME/.config/niagara-private/client-denylist.txt
	list := filepath.Join(os.Getenv("HOME"), ".config", "niagara-private", "client-denylist.txt")
	if err := os.MkdirAll(filepath.Dir(list), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(list, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r.push("(delete) "+zeroSha+" refs/heads/a aaaa").want(t, 1, "client-guard is not on PATH")
}

func TestNoClientGuardAndNoDenylistWarnsAndGoesOn(t *testing.T) {
	r := newRepo(t)
	res := r.push("(delete) " + zeroSha + " refs/heads/a aaaa")
	if res.code != 0 || res.guardErr != guardSkipped {
		t.Errorf("exit %d, stderr %q; want 0 and %q", res.code, res.guardErr, guardSkipped)
	}
}

func TestClientGuardThatCannotBeStartedRefusesNamingIt(t *testing.T) {
	r := newRepo(t)
	guard := filepath.Join(r.bin, "client-guard")
	if err := os.WriteFile(guard, []byte("\x7fELF not a program\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.push("(delete) "+zeroSha+" refs/heads/a aaaa").want(t, 1, "pre-push: cannot run "+guard+": ")
}

// A denylist that cannot be examined (its directory denies search) counts as present: an internal
// error never lets a push go unchecked.
func TestNoClientGuardWithAnUnexaminableDenylistRefuses(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	r := newRepo(t)
	dir := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NIAGARA_CLIENT_DENYLIST", filepath.Join(dir, "client-denylist.txt"))
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	res := r.push("(delete) " + zeroSha + " refs/heads/a aaaa")
	if res.code != 1 || !strings.Contains(res.err, "a client denylist exists; refusing to push unchecked") {
		t.Errorf("exit %d, stderr %q: an unexaminable denylist let the push go on", res.code, res.err)
	}
}
