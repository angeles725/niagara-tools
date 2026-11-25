package scaffoldmodule

import (
	"path/filepath"
	"strings"
	"testing"
)

// serviceKitFiles is the service type's part of the small kit: the library fixture it starts from (the Gradle
// wrapper makes it a fixture, the plumbing files are what the overlay replaces) and templates/service-job.
func serviceKitFiles() map[string]string {
	fx := "fixtures/MinimalLibrary/"
	tp := "templates/service-job/"
	return map[string]string{
		fx + "gradle/wrapper/gradle-wrapper.jar":                             string(jarBytes),
		fx + "gradlew":                                                       "#!/bin/sh\n",
		fx + "MinimalLibrary/MODULE-MAP.md":                                  "# MODULE-MAP - MinimalLibrary\ntype: library\n",
		fx + "MinimalLibrary/MinimalLibrary-rt/module-include.xml":           "<types/>\n",
		fx + "MinimalLibrary/MinimalLibrary-rt/MinimalLibrary-rt.gradle.kts": "// MinimalLibrary-rt Angeles\n",
		tp + "MODULE-MAP.md":                                                 "# MODULE-MAP - ModuleA\n| service | rt | ModuleA-rt/src/com/example/ModuleA/BModuleAService.java |\n| plumbing | rt | y |\n",
		tp + "ModuleA-rt/module-include.xml":                                 "<type class=\"com.example.ModuleA.BModuleAService\" name=\"ModuleAService\"/>\n",
		tp + "ModuleA-rt/module.palette":                                     "<p n=\"ModuleAService\" m=\"@SYM@=ModuleA\" t=\"@SYM@:ModuleAService\"/>\n",
		tp + "ModuleA-rt/src/com/example/ModuleA/BModuleAService.java":       "package com.example.ModuleA; // Example ModuleA\n",
	}
}

// --type service starts from the library fixture and lays templates/service-job over it: the overlay's files
// carry the module's names and replace the fixture's plumbing (the map without its `type: library` line).
func TestServiceType(t *testing.T) {
	out := t.TempDir()
	r := run(t, newKit(t), "ColdBox", out, "--type", "service", "--vendor", "Acme")
	if r.code != 0 || r.err != "" {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	got, m := tree(t, filepath.Join(out, "ColdBox")), "ColdBox/"
	wantFile(t, got, m+"MODULE-MAP.md", "# MODULE-MAP - ColdBox\n| service | rt | ColdBox-rt/src/com/acme/ColdBox/BColdBoxService.java |\n| plumbing | rt | y |\n")
	wantFile(t, got, m+"ColdBox-rt/module-include.xml", "<type class=\"com.acme.ColdBox.BColdBoxService\" name=\"ColdBoxService\"/>\n")
	wantFile(t, got, m+"ColdBox-rt/module.palette", "<p n=\"ColdBoxService\" m=\"cb=ColdBox\" t=\"cb:ColdBoxService\"/>\n")
	wantFile(t, got, m+"ColdBox-rt/src/com/acme/ColdBox/BColdBoxService.java", "package com.acme.ColdBox; // Acme ColdBox\n")
	for rel := range got {
		if strings.Contains(rel, "StationTest") || strings.Contains(rel, "MinimalLibrary") {
			t.Errorf("unexpected %s", rel)
		}
	}
}

// A kit without templates/service-job is exit 3 naming the template, and the partial tree is removed; the
// other types never read it.
func TestServiceTemplateMissing(t *testing.T) {
	tb := newKit(t, "templates/service-job")
	out := t.TempDir()
	r := run(t, tb, "Foo", out, "--type", "service", "--vendor", "Angeles")
	want := "scaffold-module: service-job template not found: " + tb + "/../templates/service-job\n" +
		"scaffold-module: removed the partial " + out + "/Foo\n"
	if r.code != 3 || r.out != "" || r.err != want {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
	if r := run(t, tb, "Bar", out, "--type", "driver", "--vendor", "Angeles"); r.code != 0 {
		t.Errorf("driver: code=%d err=%q", r.code, r.err)
	}
}
