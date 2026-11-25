package puretest

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// siblingJars are the build/libs jars of the parts beside the part dir rt (the other <Mod>-rt|ux|wb
// directories of the module, as build.sh leaves them), in name order. rt's own jar is left out: it holds an
// older build of the very sources being compiled. A wb or ux part's adapters use the rt part's types, so
// without these the compile fails although the module builds (issue #1075).
func siblingJars(rt string) []string {
	self, err := filepath.Abs(rt)
	if err != nil {
		self = rt
	}
	entries, err := os.ReadDir(filepath.Dir(self))
	if err != nil {
		return nil
	}
	var jars []string
	for _, e := range entries {
		dir := filepath.Join(filepath.Dir(self), e.Name())
		if !e.IsDir() || dir == self {
			continue
		}
		found, _ := filepath.Glob(dir + "/build/libs/*.jar")
		jars = append(jars, found...)
	}
	sort.Strings(jars)
	return jars
}

// adapterClassPath is the -cp value of the adapter compile: the jars of <home>/modules, then the sibling jars.
func adapterClassPath(home string, siblings []string) string {
	return strings.Join(append([]string{home + "/" + modulesJars}, siblings...), string(os.PathListSeparator))
}

// classPathNote names what the class path held, for a compile error.
func classPathNote(siblings []string) string {
	if len(siblings) == 0 {
		return "class path NIAGARA_HOME/modules and the sibling part jars (none built yet)"
	}
	return "class path NIAGARA_HOME/modules and the sibling part jars " + strings.Join(siblings, ", ")
}
