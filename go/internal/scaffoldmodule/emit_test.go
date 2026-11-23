package scaffoldmodule

import (
	"os"
	"path/filepath"
	"testing"
)

const signingLine = "scaffold-module: signing: set niagaraSigningAlias + niagaraSigningProfile (gradle properties) before the first build (a throwaway profile to copy: build-n4-module-kit/templates/signing-profile.xml)\n"

// The default (logic) run renames paths, rewrites text, copies binaries verbatim and reports on stdout.
func TestEmitLogicDefaults(t *testing.T) {
	tb := newKit(t)
	out := filepath.Join(t.TempDir(), "new", "parent") // created like mkdir -p
	r := run(t, tb, "DemoRoomPan", out, "--vendor", "Angeles")
	dest := out + "/DemoRoomPan"
	if r.code != 0 || r.err != "" || r.out != "scaffold-module: emitted DemoRoomPan -> "+dest+"\n"+signingLine {
		t.Fatalf("code=%d err=%q out=%q", r.code, r.err, r.out)
	}
	got := tree(t, dest)
	wantFile(t, got, "gradle/wrapper/gradle-wrapper.jar", string(jarBytes))
	wantFile(t, got, "gradlew", "#!/bin/sh\n# MinimalPan Angeles 4.14\n")
	wantFile(t, got, "gradlew.bat", "@rem MinimalPan angeles 7.6.17\r\n")
	wantFile(t, got, "settings.gradle.kts", "plugin 7.6.17 target 4.14 root DemoRoomPan\nno final newline 7x6y17 4x14")
	wantFile(t, got, "DemoRoomPan/DemoRoomPan-rt/src/com/angeles/DemoRoomPan/BDemoRoomPan.java",
		"package com.angeles.DemoRoomPan;\n// Copyright Angeles. DemoRoomPan-rt demoRoomPanTimer drp=a drp:b\n")
	if len(got) != 5+2+1 { // fixture files + station-test overlay + ledger
		t.Errorf("emitted %d files: %v", len(got), keys(got))
	}
	for rel, exec := range map[string]bool{"gradlew": true, "settings.gradle.kts": false, "gradlew.bat": false} {
		st, err := os.Stat(filepath.Join(dest, rel))
		if err != nil {
			t.Fatal(err)
		}
		if (st.Mode().Perm()&0o100 != 0) != exec {
			t.Errorf("%s mode %v, executable want %v", rel, st.Mode().Perm(), exec)
		}
	}
}

// --vendor, --target-version and --plugin-version reach paths (vendor, ASCII-lowercased) and text.
func TestEmitOptions(t *testing.T) {
	tb := newKit(t)
	out := t.TempDir()
	r := run(t, tb, "Foo", out, "--vendor", "Acme_2", "--target-version", "4.15", "--plugin-version", "8.0.1")
	if r.code != 0 || r.err != "" {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	got := tree(t, filepath.Join(out, "Foo"))
	wantFile(t, got, "settings.gradle.kts", "plugin 8.0.1 target 4.15 root Foo\nno final newline 7x6y17 4x14")
	wantFile(t, got, "Foo/Foo-rt/src/com/acme_2/Foo/BFoo.java",
		"package com.acme_2.Foo;\n// Copyright Acme_2. Foo-rt fooTimer f=a f:b\n")
	wantFile(t, got, "gradlew", "#!/bin/sh\n# MinimalPan Angeles 4.14\n") // wrapper scripts are never rewritten
}

// Each type reads its own fixture; the second profile directory and its references are renamed.
func TestEmitDashboardAndDriver(t *testing.T) {
	tb := newKit(t)
	for typ, second := range map[string]string{"dashboard": "ux", "driver": "wb"} {
		out := t.TempDir()
		r := run(t, tb, "AcmeLink", out, "--type", typ, "--vendor", "Example")
		if r.code != 0 || r.err != "" {
			t.Fatalf("%s: code=%d err=%q", typ, r.code, r.err)
		}
		got := tree(t, filepath.Join(out, "AcmeLink"))
		wantFile(t, got, "AcmeLink/AcmeLink-"+second+"/AcmeLink-"+second+".gradle.kts",
			"api(project(\":AcmeLink-rt\")) // AcmeLink-"+second+"\n")
		wantFile(t, got, "AcmeLink/AcmeLink-rt/src/com/example/AcmeLink/BAcmeLink.java",
			"package com.example.AcmeLink;\n// Copyright Example. AcmeLink-rt acmeLinkTimer al=a al:b\n")
	}
}

// Environment faults are exit 3 with one stderr line; an existing destination is left as it was.
func TestEnvFaults(t *testing.T) {
	t.Run("fixture missing", func(t *testing.T) {
		tb := newKit(t, "fixtures/MinimalDash")
		out := filepath.Join(t.TempDir(), "out")
		r := run(t, tb, "Foo", out, "--type", "dashboard", "--vendor", "Angeles")
		want := "scaffold-module: fixture skeleton not found: " + tb + "/../fixtures/MinimalDash\n"
		if r.code != 3 || r.out != "" || r.err != want {
			t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Errorf("out-dir was created before the fixture check: %v", err)
		}
	})
	t.Run("out-dir not creatable", func(t *testing.T) {
		tb := newKit(t)
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		for _, out := range []string{file, file + "/sub", ""} {
			r := run(t, tb, "Foo", out, "--vendor", "Angeles")
			if r.code != 3 || r.out != "" || r.err != "scaffold-module: cannot create out-dir: "+out+"\n" {
				t.Errorf("%q: code=%d out=%q err=%q", out, r.code, r.out, r.err)
			}
		}
	})
	t.Run("destination exists", func(t *testing.T) {
		tb := newKit(t)
		out := t.TempDir()
		if err := os.WriteFile(filepath.Join(out, "Foo"), []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
		r := run(t, tb, "Foo", out, "--vendor", "Angeles")
		if r.code != 3 || r.out != "" || r.err != "scaffold-module: destination already exists: "+out+"/Foo\n" {
			t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
		}
		if data, _ := os.ReadFile(filepath.Join(out, "Foo")); string(data) != "mine" {
			t.Errorf("existing destination changed: %q", data)
		}
	})
}
