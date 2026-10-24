package jarlist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestListParts pins the head/rest split: the manifest and the signature entries the bash put first stay in
// archive order, an orphan block or a nested .SF ends the head, and the rest is sorted by name.
func TestListParts(t *testing.T) {
	jar, err := golden.BuildJar("--- META-INF/MANIFEST.MF (stored) ---\nM\n--- META-INF/A.SF (stored) ---\ns\n" +
		"--- META-INF/A.RSA (stored) ---\nr\n--- META-INF/C.RSA (stored) ---\no\n--- META-INF/B.SF (stored) ---\ns\n" +
		"--- z (stored) ---\nz\n--- d/ (stored) ---\n--- a (noeol) ---\na\n--- b (base64) ---\nAAE=\n")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "x.jar")
	if err := os.WriteFile(p, []byte(jar), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := List(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `entries 9
head META-INF/MANIFEST.MF stored size=2 sha256=42097422722a4c0b +descriptor
head META-INF/A.SF stored size=2 sha256=cbc80bb5c0c0f894 +descriptor
head META-INF/A.RSA stored size=2 sha256=8e54b0ca18020275 +descriptor
rest META-INF/B.SF stored size=2 sha256=cbc80bb5c0c0f894 +descriptor
rest META-INF/C.RSA stored size=2 sha256=7427d152005f9ed0 +descriptor
rest a deflated size=1 sha256=ca978112ca1bbdca +descriptor
rest b deflated size=2 sha256=b413f47d13ee2fe6 +descriptor
rest d/ stored size=0 sha256=e3b0c44298fc1c14
rest z stored size=2 sha256=c865f6c5ab8d1b0b +descriptor
`
	if got != want {
		t.Errorf("got\n%swant\n%s", got, want)
	}
}
