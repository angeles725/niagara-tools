package safety

import (
	"os"
	"os/user"
	"strconv"
	"strings"
)

// Join is os.path.join: it adds a "/" only where none ends the path, restarts at an absolute
// part and never cleans "." or ".." (the path a reply or error shows is the one Python shows).
func Join(path string, parts ...string) string {
	for _, p := range parts {
		switch {
		case strings.HasPrefix(p, "/"):
			path = p
		case path == "" || strings.HasSuffix(path, "/"):
			path += p
		default:
			path += "/" + p
		}
	}
	return path
}

// Dirname is os.path.dirname: everything before the last "/", trailing slashes dropped unless it
// is all slashes.
func Dirname(path string) string {
	head := path[:strings.LastIndexByte(path, '/')+1]
	if trimmed := strings.TrimRight(head, "/"); trimmed != "" {
		return trimmed
	}
	return head
}

// ExpandUser is os.path.expanduser: a leading ~ is $HOME (else the passwd home of the current
// user), ~name is that user's home; a path it cannot expand is returned unchanged.
func ExpandUser(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	i := strings.IndexByte(path[1:], '/') + 1
	if i == 0 {
		i = len(path)
	}
	var home string
	if i == 1 {
		var ok bool
		if home, ok = os.LookupEnv("HOME"); !ok {
			u, err := user.LookupId(strconv.Itoa(os.Getuid()))
			if err != nil {
				return path
			}
			home = u.HomeDir
		}
	} else {
		u, err := user.Lookup(path[1:i])
		if err != nil {
			return path
		}
		home = u.HomeDir
	}
	if out := strings.TrimRight(home, "/") + path[i:]; out != "" {
		return out
	}
	return "/"
}
