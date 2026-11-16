package scaffoldmodule

import (
	"os"
	"path/filepath"
	"strings"
)

// verbatimFixtureFile says which fixture files are copied byte for byte: jars and the Gradle wrapper
// scripts are never rewritten.
func verbatimFixtureFile(src string) bool {
	return strings.HasSuffix(src, ".jar") || strings.HasSuffix(src, "/gradlew") || strings.HasSuffix(src, "/gradlew.bat")
}

func rewriteAll(string) bool { return false }

// copyTree copies every regular file under srcRoot to dstRoot. The relative path goes through pathRules;
// the content goes through textRules unless verbatim(src). Directories are created as needed.
//
// Modes follow the bash tool under the process umask: a rewritten file is created 0666, a verbatim copy
// keeps its source's permission bits, and <dstRoot>/gradlew is executable.
func copyTree(srcRoot, dstRoot string, pathRules, textRules []rule, verbatim func(src string) bool) error {
	files, err := regularFiles(srcRoot, "")
	if err != nil {
		return err
	}
	for _, rel := range files {
		src := srcRoot + "/" + rel
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o666)
		if verbatim(src) {
			st, err := os.Stat(src)
			if err != nil {
				return err
			}
			mode = st.Mode().Perm()
		} else {
			data = []byte(apply(string(data), textRules))
		}
		dstRel := apply(rel, pathRules)
		if dstRel == rootWrapperScript {
			mode |= 0o111
		}
		if err := writeFile(dstRoot+"/"+dstRel, data, mode); err != nil {
			return err
		}
	}
	return nil
}

// createFile writes one emitted file; a test replaces it to inject a write fault.
var createFile = os.WriteFile

// writeFile creates path (and its directory) with data. The destination tree is new, so there is no
// earlier file to keep whole: a plain create, with the mode the caller asks for under the umask.
// A held signal (emitting) ends the copy before the next file.
func writeFile(path string, data []byte, mode os.FileMode) error {
	if emitting.caught() {
		return errInterrupted
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	return createFile(path, data, mode)
}

// regularFiles lists the regular files under root/rel as slash paths relative to root, like
// `find root -type f`: symbolic links are neither listed nor followed. Paths are built by concatenation,
// never cleaned: root is <toolbelt>/../..., and a lexical clean would leave a symlinked toolbelt's kit.
func regularFiles(root, rel string) ([]string, error) {
	dir := root
	if rel != "" {
		dir = root + "/" + rel
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		child := e.Name()
		if rel != "" {
			child = rel + "/" + child
		}
		switch {
		case e.IsDir():
			below, err := regularFiles(root, child)
			if err != nil {
				return nil, err
			}
			files = append(files, below...)
		case e.Type().IsRegular():
			files = append(files, child)
		}
	}
	return files, nil
}
