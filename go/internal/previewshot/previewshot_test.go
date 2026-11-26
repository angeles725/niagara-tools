package previewshot

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const url = "http://127.0.0.1:8123/hmi"

func TestURLModePassesTheFlagsExactly(t *testing.T) {
	f := newFixture(t)
	shot := filepath.Join(f.dir, "out", "hmi.png")
	code, out, errs := f.run(t, url, shot)
	if code != 0 || errs != "" {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	want := []string{"--use-gl=angle", "--use-angle=swiftshader", "--enable-unsafe-swiftshader", "--virtual-time-budget=25000", "--window-size=1280,800", "--screenshot=" + shot, url}
	if got := f.browserArgs(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("browser args\n got %q\nwant %q", got, want)
	}
	if !strings.Contains(out, shot) || !strings.Contains(out, "7 bytes") {
		t.Fatalf("stdout should name the file and its size: %q", out)
	}
	if f.exists("server.args") {
		t.Fatal("URL mode must not start a server")
	}
}

func TestSizeAndBudgetOverrideTheDefaults(t *testing.T) {
	f := newFixture(t)
	shot := filepath.Join(f.dir, "a.png")
	if code, _, errs := f.run(t, url, shot, "--size", "1024x768", "--budget", "5000"); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	got := f.browserArgs(t)
	if got[3] != "--virtual-time-budget=5000" || got[4] != "--window-size=1024,768" {
		t.Fatalf("args %q", got)
	}
}

func TestPreviewShotGLReplacesTheGLFlags(t *testing.T) {
	f := newFixture(t)
	f.env["PREVIEW_SHOT_GL"] = "--use-gl=swiftshader"
	if code, _, errs := f.run(t, url, filepath.Join(f.dir, "a.png")); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if got := f.browserArgs(t); got[0] != "--use-gl=swiftshader" || got[1] != "--virtual-time-budget=25000" {
		t.Fatalf("args %q", got)
	}
}

func TestUsageErrorsExit3AndRunNothing(t *testing.T) {
	for _, args := range [][]string{
		{}, {url}, {url, "a.png", "extra"}, {url, "a.png", "--size", "1280"}, {url, "a.png", "--size", "0x800"},
		{url, "a.png", "--budget", "soon"}, {url, "a.png", "--budget", "0"}, {url, "a.png", "--nope"},
		{"/tmp/x.html", "a.png"}, {url, "a.png", "--path", "/x"}, {"--serve", "a.png"},
		{"--serve", "rc", "a.png", "--path", "hmi"}, {url, "a.png", "--size"},
	} {
		f := newFixture(t)
		code, out, errs := f.run(t, args...)
		if code != 3 || out != "" || !strings.Contains(errs, "usage: preview-shot") {
			t.Errorf("%q: exit %d, stdout %q, stderr %q", args, code, out, errs)
		}
		if f.exists("browser.args") || f.exists("server.args") {
			t.Errorf("%q: something ran", args)
		}
	}
}

func TestMissingBrowserExit3NamesTheInstallCommand(t *testing.T) {
	f := newFixture(t)
	delete(f.env, "PREVIEW_SHELL")
	code, _, errs := f.run(t, url, filepath.Join(f.dir, "a.png"))
	if code != 3 || !strings.Contains(errs, "npx playwright install chromium-headless-shell") {
		t.Fatalf("exit %d: %s", code, errs)
	}
	f.env["PREVIEW_SHELL"] = filepath.Join(f.dir, "gone")
	code, _, errs = f.run(t, url, filepath.Join(f.dir, "a.png"))
	if code != 3 || !strings.Contains(errs, "PREVIEW_SHELL") || !strings.Contains(errs, "gone") {
		t.Fatalf("a PREVIEW_SHELL that is not executable: exit %d: %s", code, errs)
	}
}

// marker makes a fake shell that records which one ran.
func marker(name string) string {
	return "#!/bin/sh\necho " + name + " > \"$FAKE_DIR/ran\"\nfor a in \"$@\"; do case \"$a\" in --screenshot=*) printf x > \"${a#--screenshot=}\";; esac; done\n"
}

func TestBrowserDiscoveryOrder(t *testing.T) {
	f := newFixture(t)
	shot := filepath.Join(f.dir, "a.png")
	pw := filepath.Join(f.home, ".cache", "ms-playwright")
	writeExec(t, filepath.Join(pw, "chromium_headless_shell-999", "chrome-linux", "headless_shell"), "#!/bin/sh\necho old > \"$FAKE_DIR/ran\"\n")
	writeExec(t, filepath.Join(pw, "chromium_headless_shell-1255", "chrome-headless-shell-linux64", "chrome-headless-shell"),
		"#!/bin/sh\necho new > \"$FAKE_DIR/ran\"\nfor a in \"$@\"; do case \"$a\" in --screenshot=*) printf x > \"${a#--screenshot=}\";; esac; done\n")
	writeExec(t, filepath.Join(f.dir, "bin", "chrome-headless-shell"), "#!/bin/sh\necho path > \"$FAKE_DIR/ran\"\n")
	f.env["PATH"] = filepath.Join(f.dir, "bin")
	delete(f.env, "PREVIEW_SHELL")
	if code, _, errs := f.run(t, url, shot); code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if got := strings.TrimSpace(f.read(t, "ran")); got != "new" {
		t.Fatalf("want the newest Playwright revision by number (1255 over 999), ran %q", got)
	}
	f.env["PREVIEW_SHELL"] = f.shell
	if code, _, _ := f.run(t, url, shot); code != 0 || !f.exists("browser.args") {
		t.Fatal("PREVIEW_SHELL must win over the Playwright cache")
	}
}

func TestBrowserDiscoveryFallbacks(t *testing.T) {
	f := newFixture(t)
	delete(f.env, "PREVIEW_SHELL")
	shot := filepath.Join(f.dir, "a.png")
	// only the old layout, in a custom PLAYWRIGHT_BROWSERS_PATH
	custom := filepath.Join(f.dir, "pw")
	writeExec(t, filepath.Join(custom, "chromium_headless_shell-1000", "chrome-linux64", "headless_shell"), marker("old-layout"))
	f.env["PLAYWRIGHT_BROWSERS_PATH"] = custom
	if code, _, errs := f.run(t, url, shot); code != 0 || strings.TrimSpace(f.read(t, "ran")) != "old-layout" {
		t.Fatalf("old layout via PLAYWRIGHT_BROWSERS_PATH: exit %d: %s", code, errs)
	}
	// nothing in the cache: chrome-headless-shell on PATH
	delete(f.env, "PLAYWRIGHT_BROWSERS_PATH")
	writeExec(t, filepath.Join(f.dir, "bin", "chrome-headless-shell"), marker("on-path"))
	f.env["PATH"] = "/nonexistent:" + filepath.Join(f.dir, "bin")
	if code, _, errs := f.run(t, url, shot); code != 0 || strings.TrimSpace(f.read(t, "ran")) != "on-path" {
		t.Fatalf("PATH: exit %d: %s", code, errs)
	}
}

func TestNoScreenshotIsExit1(t *testing.T) {
	for _, tc := range []struct{ shot, exit, why string }{
		{"none", "0", "no file"}, {"empty", "0", "empty"}, {"data", "2", "browser failed"},
	} {
		f := newFixture(t)
		t.Setenv("FAKE_SHOT", tc.shot)
		t.Setenv("FAKE_EXIT", tc.exit)
		shot := filepath.Join(f.dir, "a.png")
		if err := os.WriteFile(shot, []byte("STALE"), 0o644); err != nil {
			t.Fatal(err)
		}
		code, out, errs := f.run(t, url, shot)
		if code != 1 || out != "" || !strings.Contains(errs, "preview-shot:") {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", tc.why, code, out, errs)
		}
		if b, _ := os.ReadFile(shot); string(b) == "STALE" {
			t.Errorf("%s: a stale file from an earlier run must not pass as the new shot", tc.why)
		}
	}
}
