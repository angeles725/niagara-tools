package lint

// The operand checks every parse mode ends with: each root must be a directory (or a file Spec.FileRoot accepts),
// and Mode FileFirst's file operand must be a regular file.

import (
	"errors"
	"os"
)

func checkRoots(name string, roots []string, fileRoot func(string) bool) error {
	for _, root := range roots { // a directory, or a regular file fileRoot (Spec.FileRoot) accepts
		st, err := os.Stat(root)
		if err != nil || !(st.IsDir() || st.Mode().IsRegular() && fileRoot != nil && fileRoot(root)) {
			return errors.New(name + ": not a directory: " + root + "\n")
		}
	}
	return nil
}

// parseFileFirst is the bash `case "${1:-}" in --strict) STRICT=1; shift ;; esac; [ $# -ge 2 ] || usage;
// [ -f "$1" ] || <not found>; for d in "${@:2}"; do [ -d "$d" ] || <not a directory>; done`.
func parseFileFirst(name, usage, noun string, args []string) (Args, error) {
	var a Args
	if len(args) > 0 && args[0] == "--strict" {
		a.Strict, args = true, args[1:]
	}
	if len(args) < 2 {
		return Args{}, &UsageError{usage + "\n"}
	}
	if st, err := os.Stat(args[0]); err != nil || !st.Mode().IsRegular() {
		return Args{}, errors.New(name + ": " + noun + " not found: " + args[0] + "\n")
	}
	a.Option, a.Roots = args[0], args[1:]
	if err := checkRoots(name, a.Roots, nil); err != nil {
		return Args{}, err
	}
	return a, nil
}
