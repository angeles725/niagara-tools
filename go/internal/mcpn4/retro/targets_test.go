package retro

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A proposed delta names where to change the kit. The Python server was retired at S17, so a
// target is a kit document or a Go file of this repository that exists (second bench retro,
// delta 14).
func TestEveryDraftTargetIsAKitDocumentOrAnExistingGoFile(t *testing.T) {
	var targets []string
	for _, r := range refusals {
		targets = append(targets, r.target)
	}
	for _, v := range verdicts {
		targets = append(targets, v.target)
	}
	for _, target := range targets {
		file, _, _ := strings.Cut(target, " · ")
		switch {
		case strings.HasSuffix(file, ".py"):
			t.Errorf("target %q cites a Python file", target)
		case strings.HasPrefix(file, "go/"):
			if _, err := os.Stat(filepath.Join("..", "..", "..", "..", file)); err != nil {
				t.Errorf("target %q: %v", target, err)
			}
		}
	}
}
