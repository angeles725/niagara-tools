// Package prepush is the logic of the kit's .githooks/pre-push, the gate a push goes through before it
// leaves the clone. The hook file is a launcher (go/go-tool.sh builds and runs cmd/pre-push); every
// allow/refuse decision and every message lives here, ported from the bash hook it replaced with the
// same order, exits and text.
//
// One push, in order:
//
//  1. the client-identifier guard, on the whole stdin (clientguard.go);
//  2. every ref line of stdin (refs.go), each through gateRef (gate.go): the pushed range (range.go),
//     then the live-doctrine citation gate (livecite.go), the changelog gate (go/internal/changelog,
//     in process) and the retro gate (retro.go) with its content half (sweep.go).
//
// The first ref that fails stops the push with its status. paths.go holds every path class and trailer
// pattern as data, messages.go every line the hook prints, git.go the one place git is started.
//
// What runs where: a tool the bash hook resolved from its own checkout (the changelog gate) is called
// in process; a tool it ran from the pushed work tree (<toplevel>/build-n4-module-kit/toolbelt/
// kit-cite-resolve.sh, kit-state.sh, sweep-build-state.sh) is still executed from there, so a push is
// checked by the resolver and generators of the tree it pushes.
package prepush

import (
	"bytes"
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// hook is one run: the streams and the work tree being pushed from.
type hook struct {
	s        cli.Stdio
	toplevel string // `git rev-parse --show-toplevel`
	kit      string // <toplevel>/build-n4-module-kit
}

// Run implements `pre-push <remote> <url>` (the git pre-push contract): args are the hook arguments,
// stdin holds one "<local ref> <local sha> <remote ref> <remote sha>" line per pushed ref. It returns
// 0 to let the push go on and non-zero to refuse it.
func Run(args []string, stdin io.Reader, s cli.Stdio) int {
	// Without a work tree nothing can be checked: a bare repository is named and passes, anything else
	// refuses with git's own error (the bash hook left here with status 0 and no word).
	var gitErr bytes.Buffer
	toplevel, ok := gitOut(&gitErr, "rev-parse", "--show-toplevel")
	if !ok {
		return noWorkTree(s.Err, gitErr.String())
	}
	h := hook{s: s, toplevel: toplevel, kit: toplevel + "/" + kitDir}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return refuse(s.Err, msgStdinUnreadable, err)
	}
	pushed := strings.TrimRight(string(data), "\n")

	if rc := h.clientGuard(args, pushed); rc != 0 {
		return rc
	}
	for _, line := range strings.Split(pushed, "\n") {
		ref := parseRefLine(line)
		if ref.localSha == "" {
			continue
		}
		if rc := h.gateRef(ref.localSha, ref.remoteSha); rc != 0 {
			// the refusal names its ref: in a multi-ref push the reason lines alone do not say which one failed
			printf(s.Err, msgRefused, orUnknown(ref.localRef), orUnknown(ref.remoteRef), rc)
			return rc
		}
	}
	return 0
}

// noWorkTree is the status when `git rev-parse --show-toplevel` failed with stderr gitErr: 0 and a one-line
// notice in a bare repository, else the refusal carrying git's error.
func noWorkTree(w io.Writer, gitErr string) int {
	if bare, ok := gitOut(nil, "rev-parse", "--is-bare-repository"); ok && bare == "true" {
		dir, _ := gitOut(nil, "rev-parse", "--absolute-git-dir")
		printf(w, msgBareRepo, dir)
		return 0
	}
	why := strings.TrimSpace(strings.ReplaceAll(gitErr, "\n", " "))
	if why == "" {
		why = "no error text"
	}
	return refuse(w, msgNoToplevel, why)
}
