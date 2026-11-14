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
	src, dir := m.srcAbs+"/", m.dirAbs+"/"
	if strings.HasPrefix(dir, src) {
		return m.stop(ExitRefused, msgInsideSrc, m.dir)
	}
	if strings.HasPrefix(src, dir) {
		return m.stop(ExitRefused, msgContainsSrc, m.dir)
	}
	return proceed
}

// guardWipe is guard B: an existing mirror dir is wiped only when it is a directory that is empty or
// holds the marker file this tool writes.
func (m *mirror) guardWipe() int {
	if !m.exists {
		return proceed
	}
	if !lint.IsDir(m.dir) {
		return m.stop(ExitRefused, msgNotDir, m.dir)
	}
	held, err := os.ReadDir(m.dir)
	if err != nil {
		return m.fault(err)
	}
	if len(held) > 0 && !lint.IsFile(m.dir+"/"+Marker) {
		return m.stop(ExitRefused, msgNotMirror, m.dir)
	}
	return proceed
}
