package niagaratest

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The NRE-wrote-nuh proof. etc\ and security\ in the scratch niagara_user_home show that some boot used it; on a
// --reuse-home run boot 1 already made them, so this boot is proven to have used it by a file it wrote there: a
// regular file under nuh\ that is new or changed since just before test.exe started (that state holds everything
// the runner and its gradle build put there) and whose mtime lies in the boot window, test.exe's start to its
// exit, widened by nuhSkew. Which files the NRE rewrites on every boot is not recorded (the one listing, a
// non-booting `nre.exe -version` probe on an empty home, created etc\nre.properties, logging\ and security\
// keystores), so the proof names no file: any qualifying file passes, none fails. It runs on fresh runs too.

// nuhSkew is the clock-skew tolerance of the window: its ends are read on the WSL clock, the mtimes are stamped
// by the Windows side.
const nuhSkew = 2 * time.Second

// The walk of nuh\ is bounded: over nuhMaxEntries entries, or a dir deeper than nuhMaxDepth below nuh\, is an
// error (no proof), never a partial pass.
const (
	nuhMaxEntries = 20000
	nuhMaxDepth   = 16
	nuhListed     = 5 // qualifying files named in the row
)

// stamp is a regular file's mtime and size.
type stamp struct {
	mod  time.Time
	size int64
}

// walkNUH records every regular file under root by its slash path relative to root; symlinks are not followed
// and not recorded. An absent root is empty.
func walkNUH(root string) (map[string]stamp, error) {
	s := map[string]stamp{}
	n := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root && errors.Is(err, fs.ErrNotExist) {
				return fs.SkipAll
			}
			return err
		}
		if n++; n > nuhMaxEntries {
			return fmt.Errorf("over %d entries", nuhMaxEntries)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && strings.Count(rel, "/")+1 > nuhMaxDepth {
				return fmt.Errorf("%s is deeper than %d dirs", rel, nuhMaxDepth)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		s[rel] = stamp{info.ModTime(), info.Size()}
		return nil
	})
	return s, err
}

// nuhWritten compares nuh\ now with its state before the boot. It returns the PASS detail naming the files the
// boot wrote, or the FAIL detail (why) when none qualifies or nuh\ cannot be read.
func nuhWritten(root string, before map[string]stamp, start, end time.Time) (pass, why string) {
	window := fmt.Sprintf("boot window %s..%s, skew tolerance %s", utcStamp(start), utcStamp(end), nuhSkew)
	now, err := walkNUH(root)
	if err != nil {
		return "", fmt.Sprintf("scratch niagara_user_home not written by this boot: cannot read nuh\\: %v (%s)", err, window)
	}
	lo, hi := start.Add(-nuhSkew), end.Add(nuhSkew)
	var got []string
	for k, v := range now {
		if b, ok := before[k]; ok && b.mod.Equal(v.mod) && b.size == v.size {
			continue
		}
		if v.mod.Before(lo) || v.mod.After(hi) {
			continue
		}
		got = append(got, k)
	}
	if len(got) == 0 {
		return "", fmt.Sprintf("scratch niagara_user_home not written by this boot: none of its %s is new or changed with an mtime in the %s",
			plural(len(now), "file", "files"), window)
	}
	slices.Sort(got)
	named := make([]string, 0, min(len(got), nuhListed))
	for _, k := range got[:min(len(got), nuhListed)] {
		named = append(named, fmt.Sprintf("%s (mtime %s)", k, utcStamp(now[k].mod)))
	}
	list := strings.Join(named, ", ")
	if len(got) > nuhListed {
		list += fmt.Sprintf(" and %d more", len(got)-nuhListed)
	}
	return fmt.Sprintf("scratch niagara_user_home written by this boot: %s: %s (%s)", plural(len(got), "file", "files"), list, window), ""
}

func utcStamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }
