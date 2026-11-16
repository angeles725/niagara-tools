package prepush

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const zeroSha = "0000000000000000000000000000000000000000"

// The stubs stand where the hook looks for the toplevel's own wrappers. Each appends its argv to
// $STUB_CALLS; the resolver prints what $CITE_OUT holds and exits $CITE_RC, kit-state fails on the
// subcommand named by $KIT_STATE_FAIL_ON, and sweep exits $SWEEP_RC after checking it got two files.
const (
	stubResolver = "printf 'resolve %s\\n' \"$*\" >> \"$STUB_CALLS\"\n" +
		"[ -z \"${CITE_OUT:-}\" ] || cat \"$CITE_OUT\"\n" +
		"[ -z \"${CITE_ERR:-}\" ] || echo \"$CITE_ERR\" >&2\n" +
		"exit \"${CITE_RC:-0}\"\n"
	stubKitState = "printf 'kit-state %s\\n' \"$*\" >> \"$STUB_CALLS\"\n" +
		"[ \"$1\" != \"${KIT_STATE_FAIL_ON:-}\" ] || { echo \"kit-state: $1 violation\" >&2; exit 1; }\n" +
		"echo \"generated $1\"\n"
	// the CI-lints stub logs to its own file, so the exact call lists of the other stubs stay as they were
	stubCILocal = "printf '%s\\n' \"$*\" >> \"$CI_LOCAL_CALLS\"\n" +
		"[ -z \"${CI_LOCAL_OUT:-}\" ] || echo \"$CI_LOCAL_OUT\"\n" +
		"exit \"${CI_LOCAL_RC:-0}\"\n"
	stubSweep = "printf 'sweep %s\\n' \"$*\" >> \"$STUB_CALLS\"\n" +
		"[ -f \"$1\" ] && [ -f \"$3\" ] || { echo 'sweep: generated input missing' >&2; exit 9; }\n" +
		"echo 'sweep stdout line'\n" +
		"exit \"${SWEEP_RC:-0}\"\n"
)

// repo is a throwaway repository on branch main whose first commit is also origin/main. The process
// runs inside it with a PATH that holds only git, bash and cat, so no host tool (a real client-guard
// above all) leaks into a test.
type repo struct {
	t        *testing.T
	dir, bin string // the work tree, the only PATH directory
	git      string // the real git, for fixture commands
	calls    string // the file the stubs append their argv to
	ciCalls  string // the file the scripts/ci-local.sh stub appends its argv to
	base     string // the first commit
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := &repo{t: t, dir: filepath.Join(tmp, "r"), bin: filepath.Join(tmp, "bin"), calls: filepath.Join(tmp, "calls"),
		ciCalls: filepath.Join(tmp, "ci-local-calls")}
	for _, tool := range []string{"git", "bash", "cat"} {
		real, err := exec.LookPath(tool)
		if err != nil {
			t.Fatalf("%s is needed on PATH: %v", tool, err)
		}
		if tool == "git" {
			r.git = real
		}
		r.tool(tool, "exec "+real+` "$@"`)
	}
	for k, v := range map[string]string{
		"PATH": r.bin, "HOME": filepath.Join(tmp, "home"), "STUB_CALLS": r.calls, "CI_LOCAL_CALLS": r.ciCalls,
		"NIAGARA_CLIENT_DENYLIST": filepath.Join(tmp, "no-client-denylist"),
		"GIT_CONFIG_GLOBAL":       os.DevNull, "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.invalid",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.invalid",
	} {
		t.Setenv(k, v)
	}
	r.write("build-n4-module-kit/toolbelt/kit-cite-resolve.sh", stubResolver)
	r.write("build-n4-module-kit/toolbelt/kit-state.sh", stubKitState)
	r.write("build-n4-module-kit/toolbelt/sweep-build-state.sh", stubSweep)
	r.write("scripts/ci-local.sh", stubCILocal)
	r.write("build-n4-module-kit/types/logic.md", "# logic\n")
	r.write("README.md", "# readme\n")
	r.run("init", "-q", "-b", "main")
	r.base = r.commit("base")
	r.run("update-ref", "refs/remotes/origin/main", "HEAD")
	t.Chdir(r.dir)
	return r
}

// tool puts an executable sh script named name in the only PATH directory.
func (r *repo) tool(name, body string) {
	r.t.Helper()
	if err := os.MkdirAll(r.bin, 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		r.t.Fatal(err)
	}
}

func (r *repo) write(rel, text string) {
	r.t.Helper()
	path := filepath.Join(r.dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *repo) run(args ...string) string {
	r.t.Helper()
	out, err := exec.Command(r.git, append([]string{"-C", r.dir}, args...)...).CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit stages the whole work tree and commits it with msg; it returns the commit id.
func (r *repo) commit(msg string) string {
	r.t.Helper()
	r.run("add", "-A")
	r.run("commit", "-q", "--allow-empty", "-m", msg)
	return r.run("rev-parse", "HEAD")
}

// change writes the files (path, text pairs) and commits them with msg.
func (r *repo) change(msg string, files ...string) string {
	r.t.Helper()
	for i := 0; i+1 < len(files); i += 2 {
		r.write(files[i], files[i+1])
	}
	return r.commit(msg)
}

// result is what one hook run printed and returned. err leaves out the line every run of a fixture
// without client-guard starts with (guardSkipped); guardErr is the whole stderr.
type result struct {
	code               int
	out, err, guardErr string
}

const guardSkipped = "pre-push: client-guard not installed and no client denylist; client-identifier check skipped\n"

// push runs the hook as `pre-push origin <url>` with the ref lines on stdin.
func (r *repo) push(lines ...string) result {
	r.t.Helper()
	var out, errb bytes.Buffer
	stdin := strings.NewReader(strings.Join(lines, "\n") + "\n")
	code := Run([]string{"origin", "https://example.invalid/repo.git"}, stdin, cli.Stdio{Out: &out, Err: &errb})
	return result{code, out.String(), strings.TrimPrefix(errb.String(), guardSkipped), errb.String()}
}

// update is the ref line of a push that moves an existing remote branch from the base commit to sha.
func (r *repo) update(sha string) string {
	return "refs/heads/x " + sha + " refs/heads/x " + r.base
}

// stubCalls is every stub invocation so far, one per line.
func (r *repo) stubCalls() string {
	b, err := os.ReadFile(r.calls)
	if err != nil && !os.IsNotExist(err) {
		r.t.Fatal(err)
	}
	return string(b)
}

func (res result) want(t *testing.T, code int, stderrParts ...string) {
	t.Helper()
	if res.code != code {
		t.Errorf("exit %d, want %d\nstdout: %s\nstderr: %s", res.code, code, res.out, res.err)
	}
	for _, p := range stderrParts {
		if !strings.Contains(res.err, p) {
			t.Errorf("stderr misses %q\nstderr: %s", p, res.err)
		}
	}
}

// Trailers that let a range through the gates a test does not pin.
const (
	noChangelog = "\n\nChangelog: none (fixture)"
	trivial     = "\n\nRetro: none (trivial: fixture)"
)
