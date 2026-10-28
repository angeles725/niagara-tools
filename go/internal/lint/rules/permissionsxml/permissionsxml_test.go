package permissionsxml

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G21 contract (ledger.d/2026-10-23-r8-p1-sign.md), one hand-written case per testdata/*.txt: the
// kit template is clean; a root other than <permissions> and a group type other than station|workbench|all FAIL; a
// req-permission or opt-permission without <name> or <purposeKey> WARNs; new File("/…") and Paths.get("/…") in -rt
// src/ WARN while new File(Sys.getStationHome(), …), a comment, a string and srcTest/ are clean; --strict; a
// malformed file is exit 3; an empty, declaration-only or comment-only file (no root element) FAILs; empty roots and
// usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 12) }

// TestLatin1Decoded (run-3 review R4-001): an ISO-8859-1 (or US-ASCII) declared file is decoded, so a non-ASCII
// byte is read as its character (printed as UTF-8) and never fails the parse as invalid UTF-8; an encoding the
// security manager's files never use is an error (exit 3), not a silent pass-through.
func TestLatin1Decoded(t *testing.T) {
	t.Parallel()
	doc := func(enc string) []string {
		return []string{`<?xml version="1.0" encoding="` + enc + `"?>`,
			"<permissions><niagara-permission-groups type=\"station\">",
			"<req-permission><name>CAF\xc9</name></req-permission>", // \xc9 is É in ISO-8859-1, invalid UTF-8
			"</niagara-permission-groups></permissions>"}
	}
	for _, enc := range []string{"ISO-8859-1", "iso-8859-1", "latin1"} {
		fs, err := checkXML(javasrc.NewFile("M-rt/module-permissions.xml", doc(enc)))
		if err != nil || len(fs) != 1 || !strings.HasPrefix(fs[0].Msg, "<req-permission> CAFÉ has no <purposeKey>") {
			t.Errorf("%s: findings %+v, err %v", enc, fs, err)
		}
	}
	ascii := doc("US-ASCII")
	ascii[2] = "<req-permission><name>CAFE</name></req-permission>"
	if fs, err := checkXML(javasrc.NewFile("M-rt/module-permissions.xml", ascii)); err != nil || len(fs) != 1 ||
		!strings.HasPrefix(fs[0].Msg, "<req-permission> CAFE has no") {
		t.Errorf("US-ASCII: findings %+v, err %v", fs, err)
	}
	if _, err := checkXML(javasrc.NewFile("M-rt/module-permissions.xml", doc("Shift_JIS"))); err == nil ||
		!strings.Contains(err.Error(), "Shift_JIS") {
		t.Errorf("an unsupported encoding is an error naming it, got %v", err)
	}
}
