package buildstate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/gen"
)

// TestGoldenRealLedger generates the real build-n4-module-kit/BUILD-STATE.md from the committed
// BUILD-STATE.src.md and ledger.d/: the source is in its committed form (no generated line pasted
// into it), every fragment parses, and the result carries both the source's hand-kept fields and the
// newest fragment. BUILD-STATE.md itself is not committed, so there is no file to compare with.
func TestGoldenRealLedger(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "build-n4-module-kit")
	var diag bytes.Buffer
	got, code := Generate(root, &diag)
	if code != ExitOK {
		t.Fatalf("Generate: exit %d: %s", code, diag.String())
	}
	frags, problems, err := Scan(filepath.Join(root, "ledger.d"))
	if err != nil || len(problems) > 0 {
		t.Fatalf("scan: %v %+v", err, problems)
	}
	src, err := os.ReadFile(filepath.Join(root, SourceName))
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := Render(string(src), frags); got != gen.GeneratedLine+"\n"+want {
		t.Error("Generate differs from rendering BUILD-STATE.src.md with ledger.d/")
	}
	if !strings.Contains(got, "\n"+LastChange(frags)+"\n") || strings.Count(got, "\nmodule: ") < 2 {
		t.Error("the generated BUILD-STATE.md lacks the newest last_change or a hand-kept module envelope")
	}
}
