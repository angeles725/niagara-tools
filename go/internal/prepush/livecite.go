package prepush

import (
	"bytes"
	"os/exec"
	"strings"
)

// liveCiteGate is the live-doctrine citation gate: a range that changes a live card or the resolver
// runs `kit-cite-resolve --strict` on the live set of the pushed tree. Findings (exit 1) or any tool
// failure refuse; only exit 3 naming an absent research/modules/clients/memory root skips, with a
// named stderr line. It is local-only: CI runners hold none of those roots. It returns 0 to go on.
func (h hook) liveCiteGate(changed []string, localSha string) int {
	if !touchesLiveDoctrine(changed) {
		return 0
	}
	tree, ok := gitOut(h.s.Err, "ls-tree", "-r", "--name-only", localSha)
	if !ok {
		return refuse(h.s.Err, msgLsTreeFailed, localSha)
	}
	var cards []string
	for _, path := range lines(tree) {
		if liveDoctrine.MatchString(path) {
			cards = append(cards, path)
		}
	}
	if len(cards) == 0 {
		return 0
	}
	// the resolver reads the working tree: it must hold the pushed bytes of every path it reads or runs
	same := append(append([]string{"diff", "--quiet", localSha, "--"}, cards...), resolverPaths...)
	if !gitOK(h.s.Err, same...) {
		return refuse(h.s.Err, msgTreeDiffers, localSha)
	}
	out, rc := h.resolve(cards)
	switch rc {
	case 0:
		return 0
	case 1:
		printFindings(h, out)
		return refuse(h.s.Err, msgCiteFindings)
	case 3:
		// a research/modules/clients/memory root absent on this machine is the one named skip
		for _, line := range strings.Split(out, "\n") {
			if m := absentRoot.FindStringSubmatch(line); m != nil {
				printf(h.s.Err, msgCiteSkipped, m[1], m[2])
				return 0
			}
		}
	}
	all := strings.Split(out, "\n")
	for _, line := range all[max(0, len(all)-toolFailureTail):] {
		printf(h.s.Err, "%s\n", line)
	}
	return refuse(h.s.Err, msgCiteToolFailed, rc)
}

// touchesLiveDoctrine reports whether a changed path is a live card or part of the resolver.
func touchesLiveDoctrine(changed []string) bool {
	for _, path := range changed {
		if liveDoctrine.MatchString(path) || liveCiteTool.MatchString(path) {
			return true
		}
	}
	return false
}

// resolve runs the pushed tree's own resolver wrapper from the toplevel on the cards. It returns
// stdout and stderr together, without the trailing newlines, and the exit status.
func (h hook) resolve(cards []string) (string, int) {
	var out bytes.Buffer
	cmd := exec.Command("bash", append(append([]string{citeResolver}, resolverArgs...), cards...)...)
	cmd.Dir = h.toplevel
	cmd.Stdout, cmd.Stderr = &out, &out
	rc := exitStatus(citeResolver, cmd.Run(), &out)
	return strings.TrimRight(out.String(), "\n"), rc
}

// printFindings prints the first rows of the resolver's TSV whose status is a finding.
func printFindings(h hook, out string) {
	printed := 0
	for _, row := range strings.Split(out, "\n") {
		fields := strings.Split(row, "\t")
		if len(fields) < findingMinFields || notFinding.MatchString(fields[statusField]) {
			continue
		}
		if printed++; printed > maxFindingRows {
			return
		}
		printf(h.s.Err, "%s\n", row)
	}
}
