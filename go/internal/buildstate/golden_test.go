package buildstate

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGoldenRealLedger regenerates the committed build-n4-module-kit/BUILD-STATE.md from the committed
// ledger.d/ and requires it byte for byte: the migration lost nothing and nobody hand-edited a
// generated part since.
func TestGoldenRealLedger(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "build-n4-module-kit")
	committed, err := os.ReadFile(filepath.Join(root, "BUILD-STATE.md"))
	if err != nil {
		t.Fatal(err)
	}
	frags, problems, err := Scan(filepath.Join(root, "ledger.d"))
	if err != nil || len(problems) > 0 {
		t.Fatalf("scan: %v %+v", err, problems)
	}
	got, err := Render(string(committed), frags)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(committed) {
		t.Error("BUILD-STATE.md differs from what ledger.d/ generates; run toolbelt/gen-build-state.sh")
	}
}
