// Package storedrepack repackages a signed N4 module jar with every entry STORED, so Workbench's re-signing
// never meets a deflate size mismatch (toolbelt/stored-repack.sh has the why). The entries stream from the
// input zip to a temporary file beside the output, which is linked into place only once it is complete and
// checked: no extraction to disk, no zip/unzip tools, and never a partial output jar.
package storedrepack

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// usage is lines 2-15 of toolbelt/stored-repack.sh without their "# " (the bash printed them with sed).
const usage = `stored-repack.sh — repackage a signed N4 module jar with every entry STORED (no compression).

WHY: Workbench's JarFileSigner re-deflates each entry while re-signing; when the JDK that built
the jar (WSL OpenJDK 8) and the Workbench JDK (Windows Zulu 8) deflate to different sizes it aborts
with "ZipException: invalid entry compressed size (expected N but got M)". A clean rebuild does NOT
fix it (same deflater, same size) and a local ` + "`jarsigner`" + ` cannot detect it (same deflater again).
STORED entries make the mismatch impossible by construction and keep the content bytes identical,
so the existing signature stays valid. Field-confirmed (operator re-signed STORED jars, 2026-09-01).

Entry order matters for a valid signed jar: META-INF/MANIFEST.MF first, then every
META-INF/<ALIAS>.SF with its .RSA/.DSA/.EC block (NIAGARA4, CODESIGN, SERVER1, ...), then everything else. Cost: a bigger jar (deflate ratio lost).

Usage: stored-repack.sh <in.jar> <out.jar>      (refuses to overwrite an existing out.jar)
Check: verify-module.sh --stored <out.jar>      (0 "Defl:" entries)
`

const manifest = "META-INF/MANIFEST.MF"

// Main runs `stored-repack <in.jar> <out.jar>`. Exits 0 with one summary line; 1 for an input that is not a
// readable jar, an existing output, or an output that cannot be written; 2 for usage.
func Main(args []string, s cli.Stdio) int {
	fail := func(format string, a ...any) int {
		fmt.Fprintf(s.Err, "stored-repack: "+format+"\n", a...)
		return 1
	}
	if len(args) != 2 {
		fmt.Fprint(s.Err, usage)
		return 2
	}
	in, out := args[0], args[1]
	if st, err := os.Stat(in); err != nil || !st.Mode().IsRegular() {
		return fail("input jar not found: %s", in)
	}
	// Lstat, not bash's [ -e ]: a dangling symlink is an existing output too (zip wrote through it).
	if _, err := os.Lstat(out); err == nil {
		return fail("refusing to overwrite existing %s", out)
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return fail("cannot resolve %s: %s", out, strerror.Text(err))
	}
	zr, err := zip.OpenReader(in)
	if err != nil {
		fmt.Fprintf(s.Err, "stored-repack: %s: %v\n", in, err)
		return fail("cannot unzip %s", in)
	}
	defer zr.Close()
	files, sigs, err := order(zr.File)
	if err != nil {
		fmt.Fprintf(s.Err, "stored-repack: %s: %v\n", in, err)
		return fail("cannot unzip %s", in)
	}
	if len(files) == 0 || files[0].Name != manifest {
		return fail("no %s in %s (not a jar)", manifest, in)
	}
	n, deflated, err := write(out, files)
	var bad *entryError
	switch {
	case deflated > 0:
		return fail("%d deflated entries remain in %s", deflated, abs)
	case errors.As(err, &bad):
		fmt.Fprintf(s.Err, "stored-repack: %s: %v\n", in, bad)
		return fail("cannot unzip %s", in)
	case errors.Is(err, fs.ErrExist):
		return fail("refusing to overwrite existing %s", out)
	case err != nil:
		return fail("cannot write %s: %s", out, strerror.Text(err))
	}
	kept := "none"
	if len(sigs) > 0 {
		kept = strings.Join(sigs, " ")
	}
	fmt.Fprintf(s.Out, "stored-repack: %s  (%d entries, all STORED, signature entries kept: %s)\n", abs, n, kept)
	return 0
}

// order is the file entries of a jar in repack order: the manifest, each META-INF/<ALIAS>.SF (byte order, as
// the bash glob sorted under LC_ALL=C) followed by its .RSA, .DSA and .EC blocks, then every other file entry in
// input order. Directory entries are dropped (zip -D). sigs names the signature entries in that order. An
// entry name the bash's unzip refused (absolute, a ".." component, or a duplicate) is an error.
func order(all []*zip.File) (files []*zip.File, sigs []string, err error) {
	byName := map[string]*zip.File{}
	for _, f := range all {
		if err := safeName(f.Name); err != nil {
			return nil, nil, err
		}
		if byName[f.Name] != nil {
			return nil, nil, fmt.Errorf("duplicate entry %s", f.Name)
		}
		byName[f.Name] = f
	}
	var sfs []string
	for name := range byName {
		if dir, base := path.Split(name); dir == "META-INF/" && strings.HasSuffix(base, ".SF") && !strings.HasPrefix(base, ".") {
			sfs = append(sfs, name)
		}
	}
	sort.Strings(sfs)
	first := map[string]bool{}
	if f := byName[manifest]; f != nil {
		files, first[manifest] = append(files, f), true
	}
	for _, sf := range sfs {
		for _, name := range []string{sf, strings.TrimSuffix(sf, "SF") + "RSA", strings.TrimSuffix(sf, "SF") + "DSA", strings.TrimSuffix(sf, "SF") + "EC"} {
			if f := byName[name]; f != nil {
				files, first[name], sigs = append(files, f), true, append(sigs, name)
			}
		}
	}
	for _, f := range all {
		if !first[f.Name] && !strings.HasSuffix(f.Name, "/") {
			files = append(files, f)
		}
	}
	return files, sigs, nil
}

// safeName refuses an entry name that would land outside an extraction dir; the bash's unzip failed on it.
func safeName(name string) error {
	if name == "" || strings.HasPrefix(name, "/") {
		return fmt.Errorf("unsafe entry name %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return fmt.Errorf("unsafe entry name %q", name)
		}
	}
	return nil
}

// entryError is an input entry whose bytes cannot be read back as its headers declare them.
type entryError struct {
	name string
	err  error
}

func (e *entryError) Error() string { return fmt.Sprintf("entry %s: %v", e.name, e.err) }

// write streams files into a temporary file beside out, checks the result and links it to out; it returns the
// number of entries written. On an error, or when deflated counts entries the check found not STORED, nothing
// is left at out and the temporary file is removed.
func write(out string, files []*zip.File) (n, deflated int, err error) {
	tmp, err := os.CreateTemp(filepath.Dir(out), "."+filepath.Base(out)+".tmp-*")
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err != nil {
			tmp.Close()
		}
		os.Remove(tmp.Name())
	}()
	zw := zip.NewWriter(tmp)
	for _, f := range files {
		if err := copyStored(zw, f); err != nil {
			return 0, 0, err
		}
	}
	if err := zw.Close(); err != nil {
		return 0, 0, err
	}
	if err := tmp.Sync(); err != nil {
		return 0, 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, 0, err
	}
	if n, deflated, err = check(tmp.Name()); err != nil || deflated > 0 {
		return 0, deflated, err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return 0, 0, err
	}
	// A link never replaces an existing out (fs.ErrExist); a filesystem without hard links gets a rename.
	if err := os.Link(tmp.Name(), out); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return 0, 0, err
		}
		if err := os.Rename(tmp.Name(), out); err != nil {
			return 0, 0, err
		}
	}
	return n, 0, nil
}

// copyStored writes f as a STORED entry: the CRC and size come from f's headers and go into the local header
// (no data descriptor), and the bytes stream through f.Open, which fails at EOF when they do not match those
// headers. No buffer is sized from a header.
func copyStored(zw *zip.Writer, f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return &entryError{f.Name, err}
	}
	defer rc.Close()
	w, err := zw.CreateRaw(&zip.FileHeader{
		Name:               f.Name,
		Comment:            f.Comment,
		Method:             zip.Store,
		Flags:              f.Flags & 0x800, // keep only the UTF-8 name flag; no data descriptor
		ModifiedTime:       f.ModifiedTime,
		ModifiedDate:       f.ModifiedDate,
		CreatorVersion:     f.CreatorVersion,
		ExternalAttrs:      f.ExternalAttrs,
		CRC32:              f.CRC32,
		CompressedSize64:   f.UncompressedSize64,
		UncompressedSize64: f.UncompressedSize64,
	})
	if err != nil {
		return err
	}
	src := &readSide{r: rc}
	if _, err := io.Copy(w, src); err != nil {
		if src.err != nil {
			return &entryError{f.Name, src.err} // a bad CRC, size or deflate stream: the input is at fault
		}
		return err
	}
	return nil
}

// readSide remembers the error its reader returned, so a failed copy tells a bad input from a failed write.
type readSide struct {
	r   io.Reader
	err error
}

func (s *readSide) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if err != nil && err != io.EOF {
		s.err = err
	}
	return n, err
}

// check reopens the written zip and counts its entries and those not STORED (the bash counted the "Defl:"
// lines of unzip -v).
func check(p string) (n, deflated int, err error) {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return 0, 0, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Method != zip.Store {
			deflated++
		}
	}
	return len(zr.File), deflated, nil
}
