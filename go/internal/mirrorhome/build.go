package mirrorhome

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"syscall"
)

// plan reads what the mirror will link, before anything is wiped: the source's top-level entries but
// modules/, and the jars under modules/ that exist and are not excluded. Dot files are not linked (the
// bash tool's globs skipped them).
//
// It refuses what the bash tool reported as "mirror ready": a relative source, whose links (the source
// as given + the name) dangle from inside the mirror, and a source that cannot be listed.
func (m *mirror) plan() int {
	if !strings.HasPrefix(m.src, "/") {
		return m.stop(ExitUsage, msgRelative, m.src)
	}
	top, err := visible(m.src)
	if err != nil {
		return m.fault(err)
	}
	for _, name := range top {
		if name != modules {
			m.entries = append(m.entries, name)
		}
	}
	jars, err := visible(m.src + "/" + modules)
	if err != nil && !errors.Is(err, syscall.ENOENT) && !errors.Is(err, syscall.ENOTDIR) {
		return m.fault(err)
	}
	for _, name := range jars {
		if !strings.HasSuffix(name, jarSuffix) || slices.Contains(m.exclude, name) {
			continue
		}
		if _, err := os.Stat(m.src + "/" + modules + "/" + name); err == nil { // a dangling link is no jar
			m.jars = append(m.jars, name)
		}
	}
	return proceed
}

// visible is the names in dir that do not start with a dot, sorted.
func visible(dir string) ([]string, error) {
	held, err := os.ReadDir(dir)
	var names []string
	for _, e := range held {
		if !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, err
}

// wipe removes the mirror dir the guards allowed. A symlink to one is removed itself and its target is
// left alone; spelled with a final slash it is the target that is emptied and the mirror is built there,
// through the link (what `rm -rf <link>/` and `mkdir -p <link>/modules` did).
func (m *mirror) wipe() int {
	if !m.exists {
		return proceed
	}
	if !strings.HasSuffix(m.dir, "/") {
		if err := os.RemoveAll(m.dir); err != nil {
			return m.fault(err)
		}
		return proceed
	}
	held, err := os.ReadDir(m.dir)
	if err != nil {
		return m.fault(err)
	}
	for _, e := range held {
		if err := os.RemoveAll(m.dir + e.Name()); err != nil {
			return m.fault(err)
		}
	}
	return proceed
}

// build makes the mirror: the real modules/, the marker, the links (each to the source as given + the
// name), then the ready line.
func (m *mirror) build() int {
	if err := os.MkdirAll(m.dir+"/"+modules, 0o777); err != nil {
		return m.fault(err)
	}
	marker := fmt.Sprintf(markerFormat, m.srcAbs, m.now().Format(markerTime))
	if err := os.WriteFile(m.dir+"/"+Marker, []byte(marker), 0o666); err != nil {
		return m.fault(err)
	}
	for _, name := range m.entries {
		if err := os.Symlink(m.src+"/"+name, m.dir+"/"+name); err != nil {
			return m.fault(err)
		}
	}
	for _, name := range m.jars {
		if err := os.Symlink(m.src+"/"+modules+"/"+name, m.dir+"/"+modules+"/"+name); err != nil {
			return m.fault(err)
		}
	}
	excluded := strings.Join(m.exclude, " ")
	if excluded == "" {
		excluded = noneExcluded
	}
	fmt.Fprintf(m.s.Out, msgReady, m.dir, len(m.jars), excluded)
	return ExitOK
}
