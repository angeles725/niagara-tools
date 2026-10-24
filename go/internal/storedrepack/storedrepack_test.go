package storedrepack

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// signedSpec is a deflated jar whose "everything else" entries are not in name order, with directory entries,
// two signature aliases (one with all three blocks), an orphan block and a nested .SF.
const signedSpec = `--- META-INF/ ---
--- z/last.txt ---
zzz
--- META-INF/B.SF ---
b sig
--- a/ ---
--- a/first.txt ---
aaa
--- META-INF/MANIFEST.MF ---
Manifest-Version: 1.0
--- META-INF/C.RSA ---
orphan
--- META-INF/A.EC ---
a ec
--- META-INF/sub/N.SF ---
nested
--- META-INF/B.RSA ---
b rsa
--- META-INF/A.SF ---
a sig
--- m/mid.txt (stored) ---
mmm
--- META-INF/A.DSA ---
a dsa
--- META-INF/A.RSA ---
a rsa
`

// run builds the spec as dir/in.jar and runs Main on it with out (relative to dir).
func run(t *testing.T, spec, out string) (dir string, code int, stdout, stderr string) {
	t.Helper()
	dir = t.TempDir()
	jar, err := golden.BuildJar(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "in.jar"), []byte(jar), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code = Main([]string{filepath.Join(dir, "in.jar"), filepath.Join(dir, out)}, cli.Stdio{Out: &o, Err: &e})
	return dir, code, o.String(), e.String()
}

// entries reads a zip into name order and name -> bytes.
func entries(t *testing.T, path string) (*zip.ReadCloser, []string, map[string][]byte) {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { zr.Close() })
	var names []string
	content := map[string][]byte{}
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		content[f.Name] = b
	}
	return zr, names, content
}

// TestEntryOrder pins the order: the manifest, each top-level .SF in byte order with its .RSA, .DSA and .EC
// blocks, then every other file entry in input order (the bash wrote those in its extraction dir's walk order),
// with no directory entries.
func TestEntryOrder(t *testing.T) {
	dir, code, _, stderr := run(t, signedSpec, "out.jar")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	_, got, _ := entries(t, filepath.Join(dir, "out.jar"))
	want := []string{
		"META-INF/MANIFEST.MF",
		"META-INF/A.SF", "META-INF/A.RSA", "META-INF/A.DSA", "META-INF/A.EC",
		"META-INF/B.SF", "META-INF/B.RSA",
		"z/last.txt", "a/first.txt", "META-INF/C.RSA", "META-INF/sub/N.SF", "m/mid.txt",
	}
	if !slices.Equal(got, want) {
		t.Errorf("entries\n got %q\nwant %q", got, want)
	}
}

// TestAllStored pins the header shape Java readers need for STORED entries: method STORE, the sizes and CRC in
// the local header (no data descriptor, which java.util.zip.ZipInputStream rejects on a STORED entry), and no
// extra fields (zip -X).
func TestAllStored(t *testing.T) {
	dir, code, _, stderr := run(t, signedSpec, "out.jar")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	zr, _, _ := entries(t, filepath.Join(dir, "out.jar"))
	for _, f := range zr.File {
		if f.Method != zip.Store || f.Flags&0x8 != 0 || len(f.Extra) != 0 || f.CompressedSize64 != f.UncompressedSize64 {
			t.Errorf("%s: method %d flags %#x extra %d bytes, sizes %d/%d", f.Name, f.Method, f.Flags, len(f.Extra),
				f.CompressedSize64, f.UncompressedSize64)
		}
	}
}

// TestContentIdentity pins every output entry's bytes to the input entry of the same name, and that every
// input file entry is kept.
func TestContentIdentity(t *testing.T) {
	dir, code, _, stderr := run(t, signedSpec, "out.jar")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	_, inNames, in := entries(t, filepath.Join(dir, "in.jar"))
	_, outNames, out := entries(t, filepath.Join(dir, "out.jar"))
	files := slices.DeleteFunc(slices.Clone(inNames), func(n string) bool { return strings.HasSuffix(n, "/") })
	slices.Sort(files)
	slices.Sort(outNames)
	if !slices.Equal(files, outNames) {
		t.Fatalf("output names %q, want the input files %q", outNames, files)
	}
	for n, b := range out {
		if !bytes.Equal(b, in[n]) {
			t.Errorf("%s: output %q, input %q", n, b, in[n])
		}
	}
}

// TestSuccessLine pins the stdout line and that nothing else is written.
func TestSuccessLine(t *testing.T) {
	dir, code, stdout, stderr := run(t, signedSpec, "out.jar")
	want := "stored-repack: " + filepath.Join(dir, "out.jar") + "  (12 entries, all STORED, signature entries kept: " +
		"META-INF/A.SF META-INF/A.RSA META-INF/A.DSA META-INF/A.EC META-INF/B.SF META-INF/B.RSA)\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Errorf("exit %d\nstdout %q\nwant   %q\nstderr %q", code, stdout, want, stderr)
	}
	names, _ := os.ReadDir(dir)
	if len(names) != 2 {
		t.Errorf("dir holds %d entries after the run, want in.jar and out.jar only", len(names))
	}
}

// TestExits pins the exit codes and messages of the refusal paths; none of them leaves an output.
func TestExits(t *testing.T) {
	notJar := "--- a.txt ---\na\n"
	for _, tc := range []struct {
		name, spec, out string
		args            []string // nil: in.jar and out
		code            int
		stderr          string // "@IN@"/"@OUT@" stand for the paths
	}{
		{name: "no args", args: []string{}, code: 2, stderr: usage},
		{name: "one arg", args: []string{"x"}, code: 2, stderr: usage},
		{name: "three args", args: []string{"x", "y", "z"}, code: 2, stderr: usage},
		{name: "not a jar", spec: notJar, out: "out.jar", code: 1, stderr: "stored-repack: no META-INF/MANIFEST.MF in @IN@ (not a jar)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.args != nil {
				var o, e bytes.Buffer
				if code := Main(tc.args, cli.Stdio{Out: &o, Err: &e}); code != tc.code || e.String() != tc.stderr || o.Len() != 0 {
					t.Errorf("exit %d stdout %q stderr %q, want exit %d stderr %q", code, o.String(), e.String(), tc.code, tc.stderr)
				}
				return
			}
			dir, code, stdout, stderr := run(t, tc.spec, tc.out)
			want := strings.NewReplacer("@IN@", filepath.Join(dir, "in.jar"), "@OUT@", filepath.Join(dir, tc.out)).Replace(tc.stderr)
			if code != tc.code || stderr != want || stdout != "" {
				t.Errorf("exit %d stdout %q stderr %q, want exit %d stderr %q", code, stdout, stderr, tc.code, want)
			}
			if _, err := os.Lstat(filepath.Join(dir, tc.out)); err == nil {
				t.Errorf("%s exists after a refused run", tc.out)
			}
		})
	}
}

// TestRefuseExisting pins that an existing output is never touched.
func TestRefuseExisting(t *testing.T) {
	dir := t.TempDir()
	jar, _ := golden.BuildJar("--- META-INF/MANIFEST.MF ---\nM\n--- a ---\na\n")
	in, out := filepath.Join(dir, "in.jar"), filepath.Join(dir, "out.jar")
	if err := os.WriteFile(in, []byte(jar), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := Main([]string{in, out}, cli.Stdio{Out: &o, Err: &e})
	if b, _ := os.ReadFile(out); code != 1 || string(b) != "keep" || e.String() != "stored-repack: refusing to overwrite existing "+out+"\n" {
		t.Errorf("exit %d, out %q, stderr %q", code, b, e.String())
	}
}

// TestBadEntryLeavesNoOutput pins the streaming check: an entry whose bytes do not match its recorded CRC
// fails the run with exit 1, and neither the output nor a temporary file is left beside it.
func TestBadEntryLeavesNoOutput(t *testing.T) {
	dir := t.TempDir()
	spelled, err := golden.BuildJar("--- META-INF/MANIFEST.MF (stored) ---\nManifest-Version: 1.0\n--- a.txt (stored) ---\nhello\n")
	if err != nil {
		t.Fatal(err)
	}
	jar := []byte(spelled)
	i := bytes.LastIndex(jar, []byte("hello"))
	jar[i] = 'j' // the local data changes, the CRC in both headers does not
	in, out := filepath.Join(dir, "in.jar"), filepath.Join(dir, "out.jar")
	if err := os.WriteFile(in, jar, 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := Main([]string{in, out}, cli.Stdio{Out: &o, Err: &e})
	if code != 1 || !strings.HasSuffix(e.String(), "stored-repack: cannot unzip "+in+"\n") || o.Len() != 0 {
		t.Errorf("exit %d stdout %q stderr %q", code, o.String(), e.String())
	}
	if names, _ := os.ReadDir(dir); len(names) != 1 {
		t.Errorf("dir holds %v after a failed run, want in.jar only", names)
	}
}

// TestUsageIsTheScriptHeader pins usage to lines 2-15 of the wrapper with their "# " taken off: the bash
// printed `sed -n '2,15p' "$0"`, and the wrapper keeps that header.
func TestUsageIsTheScriptHeader(t *testing.T) {
	f, err := os.Open("../../../build-n4-module-kit/toolbelt/stored-repack.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var want strings.Builder
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan() && n <= 15; n++ {
		if n >= 2 {
			ln := sc.Text()
			if rest, ok := strings.CutPrefix(ln, "#"); ok {
				ln = strings.TrimPrefix(rest, " ")
			}
			want.WriteString(ln + "\n")
		}
	}
	if usage != want.String() {
		t.Errorf("usage\n%s\nwant (the script header)\n%s", usage, want.String())
	}
}

// TestRefuseDanglingSymlink pins that an output path holding a dangling symlink is an existing output, refused
// before the input is read (the bash's order: the overwrite check comes before unzip), and that the link and
// its missing target stay as they were.
func TestRefuseDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	in, out := filepath.Join(dir, "in.jar"), filepath.Join(dir, "link.jar")
	if err := os.WriteFile(in, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nowhere", out); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := Main([]string{in, out}, cli.Stdio{Out: &o, Err: &e})
	if want := "stored-repack: refusing to overwrite existing " + out + "\n"; code != 1 || e.String() != want {
		t.Errorf("exit %d stderr %q, want exit 1 stderr %q", code, e.String(), want)
	}
	if target, err := os.Readlink(out); err != nil || target != "nowhere" {
		t.Errorf("link.jar after the run: %q, %v", target, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "nowhere")); err == nil {
		t.Error("the run created the symlink's target")
	}
}
