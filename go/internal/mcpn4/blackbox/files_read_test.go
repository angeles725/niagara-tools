package blackbox

// Station file reads and Px views: n4_read_file, n4_list_px_views, n4_px_summary (port of
// mcp-n4-kit/tests/test_files.py). The Px fixture (mcp-n4-kit/tests/fixtures/synthetic_view.px)
// is written for these tests; it is not a station file.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func sha(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func pngBytes(tail []byte) []byte { return append([]byte("\x89PNG\r\n\x1a\n"), tail...) }

func TestAPxFileIsReturnedAsTextOverTheFileServletPath(t *testing.T) {
	r, sv := fileRig(t, nil)
	px := syntheticPx(t)
	out := r.Srv.OK("n4_read_file", Args{"file_ord": "file:^px/General.px"})
	if out["status"] != "ok" || out["http_status"] != 200.0 || out["path"] != "px/General.px" {
		t.Errorf("status/http_status/path = %v %v %v", out["status"], out["http_status"], out["path"])
	}
	if out["text"] != string(px) || out["size_bytes"] != float64(len(px)) || out["sha256"] != sha(px) {
		t.Errorf("text/size/sha256 = %.80v %v %v", out["text"], out["size_bytes"], out["sha256"])
	}
	if out["truncated"] != false {
		t.Errorf("truncated = %v", out["truncated"])
	}
	wantStrings(t, "file requests", fileRequests(sv), []string{"px/General.px"})
}

func TestABinaryFileReturnsSizeAndDigestWithoutContent(t *testing.T) {
	r, sv := fileRig(t, nil)
	png := pngBytes(bytes.Repeat(func() []byte {
		b := make([]byte, 256)
		for i := range b {
			b[i] = byte(i)
		}
		return b
	}(), 4))
	serveFile(sv, "images/logo.png", png, "image/png")
	out := r.Srv.OK("n4_read_file", Args{"file_ord": "images/logo.png"})
	if _, has := out["text"]; has {
		t.Errorf("a binary read carries text: %v", out)
	}
	if out["binary"] != true || out["size_bytes"] != float64(len(png)) || out["content_type"] != "image/png" ||
		out["sha256"] != sha(png) {
		t.Errorf("out = %v", out)
	}
}

func TestTheSizeCapTruncatesAndSaysSo(t *testing.T) {
	r, sv := fileRig(t, nil)
	serveFile(sv, "px/Big.px", []byte("<px>"+strings.Repeat("a", 5000)+"</px>"), "text/xml")
	out := r.Srv.OK("n4_read_file", Args{"file_ord": "px/Big.px", "max_bytes": 1024})
	text, _ := out["text"].(string)
	digest, hasDigest := out["sha256"]
	if out["truncated"] != true || out["size_bytes"] != 1024.0 || len(text) != 1024 {
		t.Errorf("truncated/size/len(text) = %v %v %d", out["truncated"], out["size_bytes"], len(text))
	}
	if !hasDigest || digest != nil { // a digest of a prefix would mislead
		t.Errorf("sha256 = %v (present %v), want null", digest, hasDigest)
	}
	if out["content_length"] != 5009.0 {
		t.Errorf("content_length = %v", out["content_length"])
	}
}

func TestUnsafePathsAndBogFilesSendNothing(t *testing.T) {
	r, sv := fileRig(t, nil)
	for _, bad := range []string{"../config.bog", "/etc/passwd", "px/", "px|slot:/", "config.bog", "file:/etc/passwd"} {
		r.Srv.Err("n4_read_file", Args{"file_ord": bad})
	}
	wantStrings(t, "file requests", fileRequests(sv), []string{})
	r.Srv.OK("n4_read_file", Args{"file_ord": "px/General.px"}) // the recorder sees a GET, so [] above is not vacuous
	wantStrings(t, "file requests after a safe read", fileRequests(sv), []string{"px/General.px"})
}

func TestA401OnTheFilePathLatchesAndIsNotRetried(t *testing.T) {
	r, sv := fileRig(t, nil)
	sv.SetPassword("changed") // the station now rejects the session's credentials
	before := getCount(sv)
	text := r.Srv.Err("n4_read_file", Args{"file_ord": "px/General.px"})
	if !strings.Contains(text, "HTTPBasicScheme") || strings.Contains(text, Password) {
		t.Errorf("first refusal = %q", text)
	}
	if n := getCount(sv) - before; n != 1 {
		t.Errorf("%d GETs after the first read, want 1", n)
	}
	if text := r.Srv.Err("n4_read_file", Args{"file_ord": "px/General.px"}); !strings.Contains(text, "paused") {
		t.Errorf("second refusal = %q", text)
	}
	if n := getCount(sv) - before; n != 1 { // the latch sent nothing
		t.Errorf("%d GETs after the latched read, want 1", n)
	}
}

func TestTheFileToolsAreListedAsReadOnly(t *testing.T) {
	r, _ := fileRig(t, nil)
	tools := toolsByName(t, r.Srv)
	for _, name := range []string{"n4_read_file", "n4_list_px_views", "n4_px_summary"} {
		if got := Dig(tools[name], "annotations", "readOnlyHint"); got != true {
			t.Errorf("%s readOnlyHint = %v", name, got)
		}
	}
}

// outputRig is TestReadFileOutput's: a file rig whose server has a private state dir.
func outputRig(t *testing.T) (*Rig, *boxstation.Servlets, string) {
	t.Helper()
	state := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	r, sv := fileRig(t, func(o *Options) { o.StateDir = state })
	return r, sv, state
}

func TestOutputFileSavesTheCompleteBytesPrivately(t *testing.T) {
	r, sv, state := outputRig(t)
	png := pngBytes(make([]byte, 300))
	serveFile(sv, "images/logo.png", png, "image/png")
	out := r.Srv.OK("n4_read_file", Args{"file_ord": "images/logo.png", "output_file": "logo.png"})
	path := filepath.Join(state, "files", "logo.png")
	if out["output_file"] != path {
		t.Errorf("output_file = %v, want %s", out["output_file"], path)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, png) {
		t.Errorf("saved bytes differ (%v)", err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v (%v), want 0600", info, err)
	}
}

func TestATruncatedReadIsNotSavedAndAPathNameIsRefused(t *testing.T) {
	r, sv, state := outputRig(t)
	serveFile(sv, "px/Big.px", bytes.Repeat([]byte("x"), 4000), "text/xml")
	out := r.Srv.OK("n4_read_file", Args{"file_ord": "px/Big.px", "max_bytes": 1024, "output_file": "big.px"})
	if _, has := out["output_file"]; has {
		t.Errorf("a truncated read was saved: %v", out["output_file"])
	}
	if skipped, _ := out["output_skipped"].(string); !strings.Contains(skipped, "truncated") {
		t.Errorf("output_skipped = %v", out["output_skipped"])
	}
	if _, err := os.Stat(filepath.Join(state, "files", "big.px")); !os.IsNotExist(err) {
		t.Errorf("big.px exists (%v)", err)
	}
	r.Srv.Err("n4_read_file", Args{"file_ord": "px/General.px", "output_file": "../escape.px"})
	wantStrings(t, "file requests", fileRequests(sv), []string{"px/Big.px"}) // refused before the GET
}
