package modulemap

import (
	"errors"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// abs is rel under the root as given.
func (s *scan) abs(rel string) string {
	if s.root == "/" {
		return "/" + rel
	}
	return s.root + "/" + rel
}

// parts lists the root's -rt/-ux/-wb/-se sub-directories, in name order.
func (s *scan) parts() ([]string, error) {
	if !lint.IsDir(s.root) { // a wrong path: the empty-root ERROR row, never a clean pass
		return nil, nil
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, errors.New("cannot list " + s.root + ": " + err.Error())
	}
	var parts []string
	for _, e := range entries {
		n := e.Name()
		if !lint.IsDir(s.abs(n)) {
			continue
		}
		for _, suf := range []string{"-rt", "-ux", "-wb", "-se"} {
			if strings.HasSuffix(n, suf) && len(n) > len(suf) {
				parts = append(parts, n)
				break
			}
		}
	}
	return parts, nil
}

// walk adds every file under the part to s.files (relative to the root), skipping dot directories and the
// part's build/ output. A symlinked directory is not descended (its files are listed under their real path, and
// a link back up the tree cannot loop); a symlinked file is listed. An unreadable directory fails the scan closed.
func (s *scan) walk(part string) error {
	var visit func(rel string) error
	visit = func(rel string) error {
		entries, err := os.ReadDir(s.abs(rel))
		if err != nil {
			return errors.New("cannot list every file under " + s.abs(part) + ": " + err.Error())
		}
		for _, e := range entries {
			n, child := e.Name(), rel+"/"+e.Name()
			if strings.HasPrefix(n, ".") || child == part+"/build" {
				continue
			}
			if e.Type()&os.ModeSymlink != 0 && lint.IsDir(s.abs(child)) {
				continue
			}
			if lint.IsDir(s.abs(child)) {
				if err := visit(child); err != nil {
					return err
				}
			} else if lint.IsFile(s.abs(child)) {
				s.files = append(s.files, child)
			}
		}
		return nil
	}
	return visit(part)
}

// walked is a scan of root with every part walked: lint.ErrEmptyRoot with no part, an error for an unreadable
// part directory.
func walked(root string) (*scan, error) {
	s := &scan{root: strings.TrimRight(root, "/")}
	if s.root == "" {
		s.root = "/"
	}
	parts, err := s.parts()
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	for _, p := range parts {
		if err := s.walk(p); err != nil {
			return nil, err
		}
	}
	return s, nil
}
