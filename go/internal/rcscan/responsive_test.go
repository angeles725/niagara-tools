package rcscan

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// respRun writes files (paths relative to the artifact dir) and runs rc-scan on it; it returns the exit code and
// only the responsive rows.
func respRun(t *testing.T, files map[string]string, args ...string) (int, string) {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var o, e bytes.Buffer
	code := Main(append([]string{root}, args...), cli.Stdio{Out: &o, Err: &e})
	if e.Len() != 0 {
		t.Fatalf("stderr %q", e.String())
	}
	var keep []string
	for _, l := range strings.SplitAfter(o.String(), "\n") {
		if strings.Contains(l, "responsive:") {
			keep = append(keep, l)
		}
	}
	return code, strings.Join(keep, "")
}

const (
	vpMeta   = `<meta name="viewport" content="width=device-width, initial-scale=1">`
	pageOK   = "<html><head>" + vpMeta + "</head><body></body></html>\n"
	mediaCSS = "@media (max-width: 768px) { .a { display: none; } }\n"

	wantNoViewport = "  rc-scan  rc/index.html: responsive: no <meta name=\"viewport\" content=\"width=device-width\"> in the HTML entry" +
		" page -- a phone renders it at desktop width (types/frontend-standard.md § Enforcement)\n"
	wantNoMedia = "  rc-scan  responsive: no @media width query (min-width/max-width/width) in any rc CSS or <style> block" +
		" -- the view cannot adapt to 360 / 768 / 1280 px (types/frontend-standard.md § Enforcement)\n"
)

func wantWide(path string, line int, decl string) string {
	return "  rc-scan  " + path + ":" + strconv.Itoa(line) + "  responsive: " + decl + " outside @media is wider than 360px --" +
		" the layout cannot shrink to a phone; use max-width, % or an @media rule (types/frontend-standard.md § Enforcement)\n"
}

func TestResponsive(t *testing.T) {
	t.Parallel()
	css := func(body string) map[string]string {
		return map[string]string{"rc/index.html": pageOK, "rc/css/app.css": body}
	}
	tests := []struct {
		name  string
		files map[string]string
		args  []string
		code  int
		want  string
	}{
		{"lan page without viewport", map[string]string{"rc/index.html": "<html><head></head></html>\n", "rc/a.css": mediaCSS},
			[]string{"--profile", "lan"}, 0, "WARN" + wantNoViewport},
		{"both page without viewport", map[string]string{"rc/index.html": "<html><head></head></html>\n", "rc/a.css": mediaCSS},
			[]string{"--profile", "both"}, 0, "WARN" + wantNoViewport},
		{"viewport present is clean", css(mediaCSS), []string{"--profile", "lan"}, 0, ""},
		{"viewport content before name", map[string]string{"rc/index.html": "<head><meta content=\"width=device-width\"\n name='viewport'></head>\n",
			"rc/a.css": mediaCSS}, []string{"--profile", "lan"}, 0, ""},
		{"viewport without device-width", map[string]string{"rc/index.html": "<head><meta name=\"viewport\" content=\"width=1024\"></head>\n",
			"rc/a.css": mediaCSS}, []string{"--profile", "lan"}, 0, "WARN" + wantNoViewport},
		{"html fragment is no entry page", map[string]string{"rc/frag.html": "<div>x</div>\n", "rc/a.css": mediaCSS},
			[]string{"--profile", "lan"}, 0, ""},
		{"no @media anywhere", css(".a { color: red; }\n"), []string{"--profile", "lan"}, 0, "WARN" + wantNoMedia},
		{"media without width feature is no width query", css("@media print { .a { color: red; } }\n"),
			[]string{"--profile", "lan"}, 0, "WARN" + wantNoMedia},
		{"range-form @media accepted", css("@media (width <= 768px) { .a { color: red; } }\n"), []string{"--profile", "lan"}, 0, ""},
		{"@media in an html style block accepted", map[string]string{"rc/index.html": "<html><head>" + vpMeta +
			"<style>\n@media (min-width: 600px) { .a { color: red; } }\n</style></head></html>\n"}, []string{"--profile", "lan"}, 0, ""},
		{"width outside @media", css(mediaCSS + ".panel {\n  width: 1200px;\n}\n"), []string{"--profile", "lan"}, 0,
			"WARN" + wantWide("rc/css/app.css", 3, "width: 1200px")},
		{"min-width outside @media", css(mediaCSS + ".panel { min-width: 480px; }\n"), []string{"--profile", "lan"}, 0,
			"WARN" + wantWide("rc/css/app.css", 2, "min-width: 480px")},
		{"width inside @media is clean", css("@media (min-width: 1000px) { .panel { width: 1200px; } }\n"), []string{"--profile", "lan"}, 0, ""},
		{"width after a closed @media is flagged", css("@media (min-width: 1000px) {\n.a { color: red; }\n}\n.b { width: 900px; }\n"),
			[]string{"--profile", "lan"}, 0, "WARN" + wantWide("rc/css/app.css", 4, "width: 900px")},
		{"max-width is fine", css(mediaCSS + ".panel { max-width: 1200px; }\n"), []string{"--profile", "lan"}, 0, ""},
		{"percent, vw, rem, em, calc and at-most-360 are fine", css(mediaCSS + ".a { width: 50%; }\n.b { width: 90vw; }\n" +
			".c { width: 40rem; }\n.d { min-width: 30em; }\n.e { width: calc(100% - 1200px); }\n.f { width: 360px; }\n"), []string{"--profile", "lan"}, 0, ""},
		{"strict turns WARN into FAIL", css(mediaCSS + ".panel { width: 1200px; }\n"), []string{"--profile", "lan", "--strict"}, 1,
			"FAIL" + wantWide("rc/css/app.css", 2, "width: 1200px")},
		{"strict FAILs the module-level row", css(".a{}\n"), []string{"--profile", "both", "--strict"}, 1, "FAIL" + wantNoMedia},
		{"allow marker on the declaration line", css(mediaCSS + ".panel { width: 1200px; } /* rc-scan: allow responsive */\n"),
			[]string{"--profile", "lan"}, 0, ""},
		{"allow marker in the page", map[string]string{"rc/index.html": "<!-- rc-scan: allow responsive -->\n<html><head></head></html>\n",
			"rc/a.css": mediaCSS}, []string{"--profile", "lan"}, 0, ""},
		{"allow marker silences the module-level row", css(".a { color: red; } /* rc-scan: allow responsive */\n"),
			[]string{"--profile", "lan"}, 0, ""},
		{"hmi does not run", map[string]string{"rc/index.html": "<html><head></head></html>\n", "rc/a.css": ".a { width: 1200px; }\n"},
			[]string{"--profile", "hmi"}, 0, ""},
		{"unknown (default) gives no responsive row", map[string]string{"rc/index.html": "<html><head></head></html>\n",
			"rc/a.css": ".a { width: 1200px; }\n"}, nil, 0, ""},
		{"vendored and ext assets are not scanned", map[string]string{"rc/index.html": pageOK, "rc/a.css": mediaCSS,
			"rc/vendor/x/demo.html": "<html><head></head></html>\n", "rc/vendor/x/x.css": ".a { width: 1200px; }\n",
			"rc/ext/y/y.css": ".a { width: 1200px; }\n"}, []string{"--profile", "lan"}, 0, ""},
		{"commented-out @media and declaration do not count", css("/* @media (max-width: 1px) { } */\n/* .a { width: 900px; } */\n"),
			[]string{"--profile", "lan"}, 0, "WARN" + wantNoMedia},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, rows := respRun(t, tt.files, tt.args...)
			if code != tt.code || rows != tt.want {
				t.Errorf("exit %d rows\n%q\nwant exit %d\n%q", code, rows, tt.code, tt.want)
			}
		})
	}
}
