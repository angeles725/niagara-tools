package files

// Go twins of test_files.TestFilePaths (go_server.go_twin("files", "unit")).

import (
	"strings"
	"testing"
)

func refused(t *testing.T, given string) string {
	t.Helper()
	got, err := StationPath(given)
	if err == nil {
		t.Errorf("StationPath(%q) = %q, want a refusal", given, got)
		return ""
	}
	return err.Error()
}

func accepted(t *testing.T, given, want string) {
	t.Helper()
	if got, err := StationPath(given); err != nil || got != want {
		t.Errorf("StationPath(%q) = %q, %v; want %q", given, got, err, want)
	}
}

func TestAFileOrdAndARelativePathNameTheSameStationFile(t *testing.T) {
	for _, given := range []string{"file:^px/General.px", "px/General.px", "station:|file:^px/General.px"} {
		accepted(t, given, "px/General.px")
	}
}

func TestTheUrlPathEncodesEachSegment(t *testing.T) {
	for given, want := range map[string]string{"px/floor plan#1?.px": "/file/px/floor%20plan%231%3F.px",
		"nav/NavFile.nav": "/file/nav/NavFile.nav", "a/~b_c-d.e/é%": "/file/a/~b_c-d.e/%C3%A9%25"} {
		if got, err := URLPath(given); err != nil || got != want {
			t.Errorf("URLPath(%q) = %q, %v; want %q", given, got, err, want)
		}
	}
	if _, err := URLPath("px/a\xed\xa0\x80.px"); err == nil || err.Error() != "'utf-8' codec can't encode character "+
		"'\\ud800' in position 1: surrogates not allowed" {
		t.Errorf("a lone surrogate: %v", err)
	}
}

func TestUnsafePathsAreRefusedBeforeAnyRequest(t *testing.T) {
	for _, bad := range []string{"", "..", "px/../config.bog", "/etc/passwd", "file:/etc/passwd",
		"file:px/a.px", "file:^", "px|slot:/Drivers", "station:|slot:/x",
		"px/", "px//a.px", "./px/a.px", "C:\\px\\a.px", "C:/px/a.px", "~/a.px",
		"px/a\x00.px", "px/a\n.px", strings.Repeat("x", 1100)} {
		refused(t, bad)
	}
}

func TestBogFilesAreRefusedLikeTheFileServlet(t *testing.T) {
	for _, bad := range []string{"config.bog", "CONFIG.BOG", "backups/config.bog.gz", "px/a.Bog"} {
		if msg := refused(t, bad); !strings.Contains(msg, ".bog") {
			t.Errorf("%q: %q names no .bog", bad, msg)
		}
	}
}

func TestWindowsPathAliasesOfRefusedFilesAreRefused(t *testing.T) {
	for _, bad := range []string{"config.bog.", "config.bog ", "config.bog. .", "config.bog::$DATA",
		"px/.. /config.bog", "px/.../a.px", "px/... /a.px", "px/a.px:stream",
		"backups/config.bog.gz.", "px/ /a.px"} {
		refused(t, bad)
	}
	accepted(t, "px/a b.px", "px/a b.px")
	accepted(t, "px/v1.2/a.px", "px/v1.2/a.px")
}

func TestTextTypesAreDetectedByExtensionOrContentType(t *testing.T) {
	str := func(s string) *string { return &s }
	for _, path := range []string{"px/a.px", "a.xml", "nav/NavFile.nav", "c.chart", "d.json", "e.csv", "f.html", "g.txt"} {
		if !IsText(path, nil) {
			t.Errorf("%s is not text", path)
		}
	}
	for _, c := range []struct {
		path string
		kind *string
		want bool
	}{
		{"images/a.png", str("image/png"), false}, {"notes", str("text/plain; charset=UTF-8"), true},
		{"blob.bin", str("application/octet-stream"), false}, {"a.PX", nil, true}, {"v1.2/a", nil, false},
		{"a", str(" Application/Atom+XML ;q"), true}, {"a", str("\u0085text/x"), true}, {"a.propert\u0130es", nil, false},
	} {
		if got := IsText(c.path, c.kind); got != c.want {
			t.Errorf("IsText(%q, %v) = %v", c.path, c.kind, got)
		}
	}
}

// TestRefusalTable pins each refusal's words and its order: the first rule that fires names it.
func TestRefusalTable(t *testing.T) {
	for given, want := range map[string]string{
		"":                        "file_ord must be a file ORD such as file:^px/General.px or a relative path such as px/General.px",
		"file:px/a.px":            "only station-home file ORDs (file:^...) are read, got 'file:px/a.px'",
		"station:|file:^":         "file_ord names no file: 'station:|file:^'",
		"a\x7f|b":                 "file path must not contain control characters",
		"/a|b":                    "file path must not contain '|' (no ORD hop after the file)",
		"c:a/":                    "file path must be relative to the station home, got 'c:a/'",
		"~":                       "file path must be relative to the station home, got '~'",
		"a\\b":                    "file path must be relative to the station home, got 'a\\\\b'",
		"a/..:/":                  "file path names a directory ('a/..:/'): the station serves files only",
		"a/../b:":                 "file path must not contain empty, '.' or '..' segments: 'a/../b:'",
		"a/b:c.":                  "file path must not contain ':' (drive or NTFS stream): 'a/b:c.'",
		"a /b.bog":                "file path segments must not end in '.' or ' ' (a Windows station strips them, aliasing another name): 'a /b.bog'",
		"a/B.BoG.Gz":              "the station never serves .bog / .bog.gz files (FileServlet); use the offline config.bog route instead",
		"it's":                    "",
		strings.Repeat("é", 1024): "",
		strings.Repeat("é", 1025): "file path is longer than 1024 characters",
	} {
		got, err := StationPath(given)
		switch {
		case want == "" && err != nil:
			t.Errorf("%q refused: %v", given, err)
		case want == "" && got != given:
			t.Errorf("%q became %q", given, got)
		case want != "" && (err == nil || err.Error() != want):
			t.Errorf("%q: %v\nwant %s", given, err, want)
		}
	}
}
