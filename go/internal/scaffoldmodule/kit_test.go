package scaffoldmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// result is what one run of the command printed and returned.
type result struct {
	code     int
	out, err string
}

// run calls the public entry with only the toolbelt variable set, as the launcher does.
func run(t *testing.T, toolbelt string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(name string) string {
		if name == ToolbeltEnv {
			return toolbelt
		}
		return ""
	}})
	return result{code, out.String(), errb.String()}
}

const usageWant = "usage: scaffold-module.sh <ModuleName> <out-dir> --vendor <v> [--target-version <x.y>] [--plugin-version <x.y.z>] [--type <logic|dashboard|driver|library|service>] [--with-doc]\n" +
	"       scaffold-module.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]\n"

// jarBytes is a binary payload that carries every substituted token: a verbatim copy keeps them.
var jarBytes = []byte("PK\x00\x03MinimalPan Angeles angeles 4.14 7.6.17\x00\xff")

// ledgerTemplate has the sample row the scaffold drops and no final newline (grep adds one).
const ledgerTemplate = "# VERSION-LEDGER\n\n| Module | Version |\n|---|---|\n| <Module> | <x.y.z> | <YYYY-MM-DD> | built |\n\n## Classes\n| <Module> | not at line start: kept"

// kitFiles is a small kit: one fixture per type plus the templates, each file with every token its rules rewrite.
func kitFiles() map[string]string {
	files := map[string]string{
		"templates/VERSION-LEDGER.md":                                                   ledgerTemplate,
		"templates/station-test/moduleTest-include.xml":                                 "<type class=\"test.com.example.ModuleA.BModuleALinkTest\"/>\n",
		"templates/station-test/srcTest/test/com/example/ModuleA/BModuleALinkTest.java": "package test.com.example.ModuleA;\n// Example ModuleA\nString[] in = { @LINK_IN_SLOTS@ };\n",
	}
	for fixture, second := range map[string]string{"MinimalPan": "", "MinimalDash": "ux", "MinimalDriver": "wb"} {
		lc := "m" + fixture[1:]
		sym := map[string]string{"MinimalPan": "mp", "MinimalDash": "md", "MinimalDriver": "md"}[fixture]
		root := "fixtures/" + fixture + "/"
		files[root+"gradle/wrapper/gradle-wrapper.jar"] = string(jarBytes)
		files[root+"gradlew"] = "#!/bin/sh\n# " + fixture + " Angeles 4.14\n"
		files[root+"gradlew.bat"] = "@rem " + fixture + " angeles 7.6.17\r\n"
		files[root+"settings.gradle.kts"] = "plugin 7.6.17 target 4.14 root " + fixture + "\nno final newline 7x6y17 4x14"
		files[root+fixture+"/"+fixture+"-rt/src/com/angeles/"+fixture+"/B"+fixture+".java"] = "package com.angeles." + fixture + ";\n" +
			"// Copyright Angeles. " + fixture + "-rt " + lc + "Timer " + sym + "=a " + sym + ":b\n"
		if second != "" {
			files[root+fixture+"/"+fixture+"-"+second+"/"+fixture+"-"+second+".gradle.kts"] = "api(project(\":" + fixture + "-rt\")) // " + fixture + "-" + second + "\n"
		}
	}
	for _, more := range []map[string]string{loginKitFiles(), serviceKitFiles(), docKitFiles()} {
		for rel, content := range more {
			files[rel] = content
		}
	}
	return files
}

// newKit writes kitFiles (minus the paths in drop, a path or a directory prefix) and returns its toolbelt dir.
func newKit(t *testing.T, drop ...string) string {
	t.Helper()
	kit := t.TempDir()
	if err := os.MkdirAll(filepath.Join(kit, "toolbelt"), 0o755); err != nil {
		t.Fatal(err)
	}
files:
	for rel, content := range kitFiles() {
		for _, d := range drop {
			if rel == d || len(rel) > len(d) && rel[:len(d)+1] == d+"/" {
				continue files
			}
		}
		path := filepath.Join(kit, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if filepath.Base(rel) == "gradlew" {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(kit, "toolbelt")
}

// tree returns every regular file under root (slash-relative) with its content; a missing root is empty.
func tree(t *testing.T, root string) map[string]string {
	t.Helper()
	got := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == root {
				return filepath.SkipAll
			}
			return err
		}
		if d.Type().IsRegular() {
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			rel, _ := filepath.Rel(root, path)
			got[filepath.ToSlash(rel)] = string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// wantFile fails unless the emitted tree has rel with exactly this content.
func wantFile(t *testing.T, got map[string]string, rel, content string) {
	t.Helper()
	have, ok := got[rel]
	if !ok {
		t.Fatalf("missing %s; emitted: %v", rel, keys(got))
	}
	if have != content {
		t.Errorf("%s:\n got %q\nwant %q", rel, have, content)
	}
}
