package rcscan

import "testing"

// TestStaticNoVersion: the reference forms issue #1035 judges, the fingerprint at any query position, and the
// URLs that are no static reference.
func TestStaticNoVersion(t *testing.T) {
	t.Parallel()
	mounts := []string{"dash"}
	for u, want := range map[string]bool{
		"js/app.js": true, "./css/a.css": true, "img/a.png#x": true, "/module/m/a.js": true, "/file/x/a.png": true,
		"/dash/img/a.png": true, "js/a.js?x=1": true, "js/a.js?": true,
		"js/a.js?v=1": false, "js/a.js?x=1&v=2": false, "js/a.js?x=1&amp;v=2": false, "js/a.js?ver=2": false,
		"js/a.js?version=2": false, "js/a.js?v=": true, "js/a.js?v": true, "js/a.js?x=1&v=": true, "js/a.js?vv=1": true, "js/a.js?x=v": true,
		"": false, "#top": false, "//cdn/x.js": false, "https://x/js/a.js": false, "HTTP://x/img/a.png": false,
		"data:image/png;base64,AA": false, "blob:abc": false, "javascript:void(0)": false, "mailto:a@b": false,
		"page.html": false, "/dash/api/status": false, "/dash/module/x": false, "/other/img/a.png": false,
		"img/${n}.png": false, "img/{{n}}.png": false, "../img/a.png": false, "/api/x": false,
	} {
		if got := StaticNoVersion(u, mounts); got != want {
			t.Errorf("StaticNoVersion(%q) = %v, want %v", u, got, want)
		}
	}
}

// TestWaived: a marker needs a comment opener of the file's syntax and a reason before the comment closes.
func TestWaived(t *testing.T) {
	t.Parallel()
	for line, want := range map[string]bool{
		"x // rc-scan: allow static-url-nover why": true, "x /* rc-scan: allow static-url-nover why */": true,
		"x <!-- rc-scan: allow static-url-nover why -->": true, "x // rc-scan: allow static-url-nover": false,
		"x /* rc-scan: allow static-url-nover */": false, "x <!-- rc-scan: allow static-url-nover  -->": false,
		"x rc-scan: allow static-url-nover why": false, "x // rc-scan: allow static-url-nover-x why": false,
		"x // rc-scan: allow datauri-budget why": false,
	} {
		if got := waived(line); got != want {
			t.Errorf("waived(%q) = %v, want %v", line, got, want)
		}
	}
}
