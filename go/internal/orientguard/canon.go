package orientguard

import (
	"errors"
	"os"
	"strings"
	"syscall"
)

// maxLinks bounds the symlinks followed while resolving one path: a loop ends there.
const maxLinks = 255

// canonical is `realpath <name>` (GNU, no option): every symlink resolved, "." and ".." applied, and
// all components but the last must exist. cwd is the physical working directory. ok is false where
// realpath fails.
func canonical(name, cwd string) (string, bool) { return resolve(name, cwd, false) }

// reachable is name resolved as far as it exists: symlinks are followed while components exist and the
// rest is applied by text (`realpath -m`). ok is false when even that cannot be told (a link loop, a
// directory that cannot be searched).
func reachable(name, cwd string) (string, bool) { return resolve(name, cwd, true) }

// Reachable is reachable for other path guards (the mirror tool's guard A runs on it before it creates
// anything): the longest existing ancestor of name resolved, the rest applied by its text.
func Reachable(name, cwd string) (string, bool) { return reachable(name, cwd) }

// resolve walks name one component at a time, as gnulib's canonicalize does.
func resolve(name, cwd string, pastMissing bool) (string, bool) {
	if name == "" {
		return "", false
	}
	var parts []string
	if !strings.HasPrefix(name, "/") {
		parts = components(cwd)
	}
	rest, links, missing := name, 0, false
	for {
		var comp string
		if rest = strings.TrimLeft(rest, "/"); rest == "" {
			break
		}
		comp, rest = cutComponent(rest)
		switch comp {
		case ".":
			continue
		case "..":
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
			continue
		}
		parts = append(parts, comp)
		if missing {
			continue
		}
		path := join(parts)
		target, err := os.Readlink(path)
		if err == nil {
			if links++; links > maxLinks {
				return "", false
			}
			if parts = parts[:len(parts)-1]; strings.HasPrefix(target, "/") {
				parts = nil
			}
			rest = target + rest
			continue
		}
		switch {
		case needsDirectory(rest):
			_, err = os.Stat(path + "/")
		case errors.Is(err, syscall.EINVAL): // it exists and is not a symlink
			err = nil
		}
		absent := errors.Is(err, syscall.ENOENT)
		switch {
		case err == nil:
		case pastMissing && (absent || errors.Is(err, syscall.ENOTDIR)):
			missing = true
		case pastMissing || !absent || strings.TrimLeft(rest, "/") != "":
			return "", false
		}
	}
	return join(parts), true
}

// needsDirectory reports whether rest, what follows a component, only works when that component is a
// directory: a final slash, a final "/.", or a "/.." anywhere next (gnulib suffix_requires_dir_check).
func needsDirectory(rest string) bool {
	for strings.HasPrefix(rest, "/") {
		if rest = strings.TrimLeft(rest, "/"); rest == "" {
			return true
		}
		if rest[0] != '.' {
			return false
		}
		if rest = rest[1:]; rest == "" || rest == "." || strings.HasPrefix(rest, "./") {
			return true
		}
	}
	return false
}

// cutComponent splits rest, which does not start with a slash, into its first component and what follows it
// (starting with a slash, or empty).
func cutComponent(rest string) (comp, after string) {
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return rest[:i], rest[i:]
	}
	return rest, ""
}

func components(abs string) []string {
	return strings.FieldsFunc(abs, func(c rune) bool { return c == '/' })
}

func join(parts []string) string { return "/" + strings.Join(parts, "/") }
