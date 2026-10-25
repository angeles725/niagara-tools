package stationsnapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
)

// snapFile copies <station>/<rel> to <out>/<rel> as `cp -p` then `chmod 0644` did and records its manifest
// entry, or prints a FAIL row (unreadable, or the copy failed with cp's message).
func (r *run) snapFile(rel string) {
	src, dst := r.station+"/"+rel, r.out+"/"+rel
	if syscall.Access(src, 4) != nil {
		r.row("FAIL", rel, "unreadable")
		r.failed = true
		return
	}
	data, err := cpP(src, dst)
	if err != nil {
		fmt.Fprintf(r.s.Err, "cp: %s\n", err)
		r.row("FAIL", rel, "copy failed")
		r.failed = true
		return
	}
	_ = os.Chmod(dst, 0o644) // outputs are never +x, whatever mode the source had (D10)
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	fmt.Fprintf(r.s.Out, "%-4s  %-14s  %s  sha256=%s bytes=%d\n", "PASS", "station-snap", rel, sha, len(data))
	r.files = append(r.files, fmt.Sprintf(`{"relpath": "%s", "sha256": "%s", "bytes": %d}`, rel, sha, len(data)))
}

// cpP is GNU `cp -p src dst` for one regular file (dst not a directory): the bytes copied, or an error worded
// as cp's message after "cp: ". The source mode and access/modification times are kept.
func cpP(src, dst string) ([]byte, error) {
	st, err := os.Stat(src)
	if err != nil {
		return nil, fmt.Errorf("cannot stat %s: %s", cpQuote(src), errText(err))
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s for reading: %s", cpQuote(src), errText(err))
	}
	flags := os.O_WRONLY | os.O_TRUNC
	if _, err := os.Stat(dst); errors.Is(err, fs.ErrNotExist) {
		flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	} else if err != nil {
		return nil, fmt.Errorf("cannot stat %s: %s", cpQuote(dst), errText(err))
	}
	f, err := os.OpenFile(dst, flags, st.Mode().Perm())
	if err != nil {
		return nil, fmt.Errorf("cannot create regular file %s: %s", cpQuote(dst), errText(err))
	}
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, fmt.Errorf("error writing %s: %s", cpQuote(dst), errText(err))
	}
	_ = os.Chmod(dst, st.Mode().Perm())
	atime := st.ModTime()
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		atime = time.Unix(sys.Atim.Sec, sys.Atim.Nsec)
	}
	_ = os.Chtimes(dst, atime, st.ModTime())
	return data, nil
}

// consoles are the regular files (symlinks followed) the bash glob <station>/console*.txt named, in C order.
func consoles(station string) []string {
	entries, _ := os.ReadDir(station) // an unreadable station dir globs nothing
	var out []string
	for _, e := range entries {
		n := e.Name()
		if len(n) >= len("console.txt") && strings.HasPrefix(n, "console") && strings.HasSuffix(n, ".txt") &&
			isRegular(station+"/"+n) {
			out = append(out, n)
		}
	}
	return out
}

// historyPointers records every regular file under <station>/history as a size pointer, as the bash
// `find history -type d -name '.*' -prune -o -type f -print | sort` loop did: symlinks are neither followed
// nor listed, dot-dirs are pruned unread, an unreadable dir prints find's message, an unreadable file the
// shell's redirection message and the size 0.
func (r *run) historyPointers() {
	root := r.station + "/history"
	if !isDir(root) {
		return
	}
	var paths []string
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := readDirNames(dir)
		if err != nil {
			fmt.Fprintf(r.s.Err, "find: %s: %s\n", findQuote(dir), errText(err))
			return
		}
		for _, n := range entries {
			p := dir + "/" + n
			st, err := os.Lstat(p)
			switch {
			case err != nil:
				fmt.Fprintf(r.s.Err, "find: %s: %s\n", findQuote(p), errText(err))
			case st.IsDir() && !strings.HasPrefix(n, "."):
				walk(p)
			case st.Mode().IsRegular():
				paths = append(paths, p)
			}
		}
	}
	if st, err := os.Lstat(root); err == nil && st.IsDir() { // find -P: a symlinked history is not entered
		walk(root)
	}
	sort.Strings(paths)
	for _, p := range paths {
		size := int64(0)
		if f, err := os.Open(p); err != nil {
			r.shellErr(lineHistWc, p+": "+errText(err))
		} else {
			if st, err := f.Stat(); err == nil {
				size = st.Size()
			}
			f.Close()
		}
		r.pointers = append(r.pointers, fmt.Sprintf(`{"relpath": "%s", "bytes": %d}`, strings.TrimPrefix(p, r.station+"/"), size))
	}
}

// readDirNames lists dir's entry names (unsorted order does not matter: the paths are sorted after).
func readDirNames(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

// manifest is manifest.json as the bash printf/awk assembly wrote it (relpaths unescaped, as the bash did).
func manifest(station, stamp string, files, pointers []string) string {
	join := func(entries []string) string {
		if len(entries) == 0 {
			return ""
		}
		return "\n    " + strings.Join(entries, ",\n    ") + "\n"
	}
	return fmt.Sprintf("{\n  \"station\": \"%s\",\n  \"ts\": \"%s\",\n  \"files\": [%s  ],\n  \"pointers\": [%s  ]\n}\n",
		station, stamp, join(files), join(pointers))
}

// writeFile is the bash `{ ...; } > p`: the open error, or the content written.
func writeFile(p, content string) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, content)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
