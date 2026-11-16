package spy

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PathError is a path refused before any request.
type PathError struct{ Msg string }

func (e *PathError) Error() string { return e.Msg }

func refuse(path, why string) error {
	return &PathError{Msg: fmt.Sprintf("refusing spy path %q: %s; give a path below the spy root such as "+
		"\"metrics\" or \"util/JMX Info\" (empty for the index)", path, why)}
}

// CleanPath is path as the segments below the spy root joined by "/" ("" is the index), or the
// refusal: only plain names may be read, so a path can neither leave the spy root nor carry a
// query, and the known action pages are never fetched.
func CleanPath(path string) (string, error) {
	clean, err := cleanNames(path)
	if err != nil {
		return "", err
	}
	for _, action := range actionPaths {
		if strings.EqualFold(clean, action) {
			return "", refuse(path, "it is an action page, and this tool only reads")
		}
	}
	return clean, nil
}

// cleanNames is CleanPath without the action pages rule: what may be named, not fetched.
func cleanNames(path string) (string, error) {
	if len(path) > MaxPathLen {
		return "", refuse(strings.ToValidUTF8(path[:40], "?")+"...", fmt.Sprintf("longer than %d bytes", MaxPathLen))
	}
	if !utf8.ValidString(path) {
		return "", refuse(strings.ToValidUTF8(path, "?"), "it is not valid UTF-8")
	}
	for _, r := range path {
		if unicode.IsControl(r) {
			return "", refuse(path, "it holds a control character")
		}
	}
	for _, rule := range refusedChars {
		if strings.ContainsAny(path, rule.chars) {
			return "", refuse(path, "it holds "+rule.why)
		}
	}
	if strings.Contains(path, "//") {
		return "", refuse(path, "it holds an empty segment")
	}
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", nil
	}
	for _, seg := range strings.Split(trimmed, "/") {
		switch {
		case strings.TrimSpace(seg) != seg:
			return "", refuse(path, "a segment starts or ends with a space")
		case strings.Trim(seg, ".") == "":
			return "", refuse(path, "it holds a '.' or '..' segment")
		}
	}
	return trimmed, nil
}

// URLPath is the URL of the page at a clean path: every segment escaped, under Root.
func URLPath(clean string) string {
	if clean == "" {
		return Root
	}
	segs := strings.Split(clean, "/")
	for i, s := range segs {
		segs[i] = escape(s)
	}
	return Root + strings.Join(segs, "/")
}

// escape is a segment with every byte outside A-Z a-z 0-9 _ . - ~ as %XX: the path travels in
// the query part of the URL, where '+' and ',' would not survive as themselves.
func escape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.IndexByte("_.-~", c) >= 0 {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// IsURLPath reports whether urlPath is one URLPath builds: the only URLs a spy read may GET.
func IsURLPath(urlPath string) bool {
	rest, ok := strings.CutPrefix(urlPath, Root)
	if !ok {
		return false
	}
	plain, err := url.PathUnescape(rest)
	if err != nil {
		return false
	}
	clean, err := CleanPath(plain)
	return err == nil && URLPath(clean) == urlPath
}

// underRoot is what an href names below the spy root, unescaped; ok is false when the href
// leaves the root.
func underRoot(href string) (string, bool) {
	for _, root := range linkRoots {
		if rest, found := strings.CutPrefix(href, root); found {
			plain, err := url.PathUnescape(rest)
			return strings.Trim(plain, "/"), err == nil
		}
	}
	return "", false
}

// linkPath is the clean spy path a link leads to; ok is false when it leaves the spy root or
// names what CleanPath refuses.
func linkPath(href string) (string, bool) {
	plain, ok := underRoot(href)
	if !ok {
		return "", false
	}
	clean, err := CleanPath(plain)
	return clean, err == nil
}

// actionPath is the clean spy path a form posts to, without the query or fragment its action
// carries (a CSRF value, for example); ok is false when the action leaves the spy root or does
// not clean to a path. An action page is named here: naming it is not fetching it.
func actionPath(action string) (string, bool) {
	for _, root := range linkRoots {
		rest, found := strings.CutPrefix(action, root)
		if !found {
			continue
		}
		if cut := strings.IndexAny(rest, "?#"); cut >= 0 {
			rest = rest[:cut]
		}
		plain, err := url.PathUnescape(rest)
		if err != nil {
			return "", false
		}
		clean, err := cleanNames(plain)
		return clean, err == nil && clean != ""
	}
	return "", false
}
