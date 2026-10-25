package niagaratest

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
)

// snapshot records the install entries a build or a run could write: every entry of modules/ by name, size
// and mtime, and the entries of etc/, security/ and the top level by name and mtime. A full walk of a 1.3 GB
// install over /mnt takes minutes; a write anywhere else would still have to add or touch a top-level entry.
type snapshot map[string]string

func takeSnapshot(root string) (snapshot, error) {
	s := snapshot{}
	for _, d := range []struct {
		dir  string
		size bool
	}{{".", false}, {"modules", true}, {"etc", false}, {"security", false}} {
		es, err := os.ReadDir(path.Join(root, d.dir))
		if os.IsNotExist(err) && d.dir != "." {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			info, err := e.Info()
			if err != nil {
				return nil, err
			}
			v := info.ModTime().UTC().String()
			if d.size {
				v += fmt.Sprintf(" %d bytes", info.Size())
			}
			s[path.Join(d.dir, e.Name())] = v
		}
	}
	return s, nil
}

// diff lists how now differs from s, sorted, at most 5 entries named.
func (s snapshot) diff(now snapshot) string {
	var ds []string
	for k, v := range s {
		if w, ok := now[k]; !ok {
			ds = append(ds, "removed "+k)
		} else if w != v {
			ds = append(ds, "changed "+k)
		}
	}
	for k := range now {
		if _, ok := s[k]; !ok {
			ds = append(ds, "added "+k)
		}
	}
	slices.Sort(ds)
	if len(ds) > 5 {
		ds = append(ds[:5], fmt.Sprintf("and %d more", len(ds)-5))
	}
	return strings.Join(ds, ", ")
}
