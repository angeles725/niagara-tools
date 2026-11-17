package mirrorhome

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// checkSource refuses a source that is not a niagara_home.
func (m *mirror) checkSource() int {
	if !lint.IsDir(m.src + "/" + repository) {
		return m.stop(ExitUsage, msgNotHome, m.src)
	}
	return proceed
}

// guardSource is guard A: the mirror dir is never the source install, a place inside it, or a place
// around it. Both are compared resolved and by whole components (the final slash).
func (m *mirror) guardSource() int {
	if refusal := sourceRefusal(m.srcAbs, m.dirAbs); refusal != "" {
		return m.stop(ExitRefused, refusal, m.dir)
	}
	return proceed
}

// sourceRefusal is guard A's message for a mirror dir against the resolved source ("" when it passes): the
// source or inside it, or around it, by whole components (the final slash). The mirror dir is resolved, or
// reachable when it is not there yet (its longest existing ancestor resolved, the rest by its text).
func sourceRefusal(srcAbs, dirAbs string) string {
	src, dir := strings.TrimSuffix(srcAbs, "/")+"/", strings.TrimSuffix(dirAbs, "/")+"/"
	switch {
	case strings.HasPrefix(dir, src):
		return msgInsideSrc
	case strings.HasPrefix(src, dir):
		return msgContainsSrc
	}
	return ""
}

// guardWipe is guard B: an existing mirror dir is wiped only when it is a directory that is empty or
// holds the marker file this tool writes. A symlink is never one, whatever its spelling or target. It looks
// at at, where the mirror will be written (the message names the mirror dir as given).
func (m *mirror) guardWipe() int {
	if m.isLink() {
		return m.stop(ExitRefused, msgLink, m.dir)
	}
	if !m.exists {
		return proceed
	}
	if !lint.IsDir(m.at) {
		return m.stop(ExitRefused, msgNotDir, m.dir)
	}
	held, err := os.ReadDir(m.at)
	if err != nil {
		return m.fault(err)
	}
	if len(held) > 0 && !lint.IsFile(m.at+"/"+Marker) {
		return m.stop(ExitRefused, msgNotMirror, m.dir)
	}
	return proceed
}

// isLink reports whether the mirror dir is a symlink, as spelled or where guard A judged it ("missing/../link"
// keeps link lexically, unresolved).
func (m *mirror) isLink() bool { return isLink(m.dir) || isLink(m.at) }

// isLink reports whether dir names a symlink however it is spelled: final slashes and final "/." are
// dropped before the link itself is looked at (Lstat), so "link/" and "link/." are the link too.
func isLink(dir string) bool {
	name := dir
	for {
		trimmed := strings.TrimSuffix(strings.TrimRight(name, "/"), "/.")
		if trimmed == name || trimmed == "" {
			break
		}
		name = trimmed
	}
	st, err := os.Lstat(name)
	return err == nil && st.Mode()&os.ModeSymlink != 0
}
