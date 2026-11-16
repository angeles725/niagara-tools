package ntplmanifest

import (
	"fmt"
	"os"
	"strings"
)

// home is a Niagara installation the lint may ask which modules it ships: its path and what named it
// (homeOption or homeEnv), "" when nobody gave one.
type home struct{ dir, from string }

// homeOf is the installation of a run: the option's value when given, else the environment's.
func homeOf(option string, getenv func(string) string) home {
	if option != "" {
		return home{option, homeOption}
	}
	if dir := getenv(homeEnv); dir != "" {
		return home{dir, homeEnv}
	}
	return home{}
}

// shipped is the set of module parts h ships: the name of each <home>/modules/<name>.jar that is a regular
// file the lint can open (a link to one counts; a dangling link, a directory or an unreadable file ships
// nothing), part suffix kept (fox-rt). nil with no installation. An installation path with no modules/ directory, or whose modules/
// cannot be listed, is an error: the list is then not known, which is never the same as an empty one.
func (h home) shipped() (map[string]bool, error) {
	if h.dir == "" {
		return nil, nil
	}
	dir := h.dir + "/" + modulesDir
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf(homeNoModules, h.from, h.dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf(homeUnread, h.from, h.dir, err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		if name := e.Name(); strings.HasSuffix(name, jarSuffix) && !strings.HasPrefix(name, ".") && readableFile(dir+"/"+name) {
			names[strings.TrimSuffix(name, jarSuffix)] = true
		}
	}
	return names, nil
}

// readableFile reports whether path (followed through a link) is a regular file that can be opened.
func readableFile(path string) bool {
	if st, err := os.Stat(path); err != nil || !st.Mode().IsRegular() {
		// Mutation: NTPL-jar-regular -- counting a jar by its name alone ships a module from a dangling link or a directory
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
