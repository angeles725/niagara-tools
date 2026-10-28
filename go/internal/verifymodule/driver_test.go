package verifymodule

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureDriver is the merged DR2 scaffold every driver check is pinned on: it must pass clean, and each
// negative case is a copy of it with one doctrine rule broken.
const fixtureDriver = "../../../build-n4-module-kit/fixtures/MinimalDriver/MinimalDriver"

const (
	fxRt  = "mod/MinimalDriver-rt/src/com/angeles/MinimalDriver/"
	fxWb  = "mod/MinimalDriver-wb/src/com/angeles/MinimalDriver/wb/"
	fxJar = "MinimalDriver-rt.jar=META-INF/module.xml=<module/>"
)

// copyFixture copies the MinimalDriver module dir to <dir>/mod and returns dir.
func copyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	err := filepath.WalkDir(fixtureDriver, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(fixtureDriver, p)
		dst := filepath.Join(dir, "mod", rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// edit replaces old with new in <dir>/<rel>; the case is void when old is absent, so that is fatal.
func edit(t *testing.T, dir, rel, old, new string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q: the fixture changed, so this negative case no longer breaks anything", rel, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), old, new, -1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// driverRows runs the gate with --src on the MinimalDriver-rt stub jar and returns its driver-* rows.
func driverRows(t *testing.T, dir string) []string {
	t.Helper()
	write(t, dir, map[string]string{strings.SplitN(fxJar, "=", 2)[0]: strings.SplitN(fxJar, "=", 2)[1]})
	_, out, _ := runMain(dir, "--src", "@DIR@/mod", "@DIR@/MinimalDriver-rt.jar")
	var rows []string
	for _, ln := range strings.Split(out, "\n") {
		if f := strings.Fields(ln); len(f) >= 2 && strings.HasPrefix(f[1], "driver-") {
			rows = append(rows, ln)
		}
	}
	return rows
}

// wantRow fails unless one row has the status and check and contains every part.
func wantRow(t *testing.T, rows []string, status, check string, parts ...string) {
	t.Helper()
	for _, r := range rows {
		f := strings.Fields(r)
		if f[0] != status || f[1] != check {
			continue
		}
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(r, p)
		}
		if ok {
			return
		}
	}
	t.Errorf("no %s %s row containing %q in\n%s", status, check, parts, strings.Join(rows, "\n"))
}

// TestDriverFixtureClean: the DR2 scaffold passes every driver check with no FAIL or WARN row, and every
// check prints a row.
func TestDriverFixtureClean(t *testing.T) {
	rows := driverRows(t, copyFixture(t))
	for _, r := range rows {
		if s := strings.Fields(r)[0]; s == "FAIL" || s == "WARN" {
			t.Errorf("fixture row %s", r)
		}
	}
	for _, c := range driverChecks {
		found := false
		for _, r := range rows {
			found = found || strings.Fields(r)[1] == c
		}
		if !found {
			t.Errorf("no %s row on the fixture in\n%s", c, strings.Join(rows, "\n"))
		}
	}
}

// TestDriverNotDriver: a module with no driver class gets no driver row, so every recorded golden (none
// holds a driver) stays byte-identical; a driver base named only in a comment or a string is no driver.
func TestDriverNotDriver(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{
		"Foo-rt.jar":                  "META-INF/module.xml=<module/>",
		"mod/Foo-rt/src/com/x/A.java": "// class BA extends BNNetwork {}\nclass BB extends BComponent { String s = \"class BC extends BNDevice {\"; }\n",
	})
	_, out, _ := runMain(dir, "--src", "@DIR@/mod", "@DIR@/Foo-rt.jar")
	if strings.Contains(out, "  driver-") {
		t.Errorf("non-driver module got driver rows:\n%s", out)
	}
	if _, out, _ = runMain(dir, "@DIR@/Foo-rt.jar"); strings.Contains(out, "  driver-") {
		t.Errorf("no --src run got driver rows:\n%s", out)
	}
}

// TestDriverScaffold: lint-driver-scaffold runs on every driver module; its FAIL rows fail the gate and
// its WARN rows warn.
func TestDriverScaffold(t *testing.T) {
	dir := copyFixture(t)
	edit(t, dir, fxRt+"BMinimalDriverDevice.java", "      if (answer.isError()) pingFail(\"device error \" + answer.status);\n      else pingOk();",
		"      pingOk();")
	edit(t, dir, fxRt+"BMinimalDriverDevice.java", "    if (!hasAddress())\n    {\n      pingFail(\"ipAddress and udpPort are not set\");\n      return;\n    }\n    try\n    {\n      Frame answer = minimalDriverNetwork().send(peer(), MinimalDriverCodec.CMD_PING, 0, 0);\n      pingOk();\n    }\n    catch (Exception e)\n    {\n      pingFail(String.valueOf(e.getMessage() != null ? e.getMessage() : e));\n    }",
		"    pingOk();")
	edit(t, dir, fxRt+"model/MinimalDriverCodec.java", "package com.angeles.MinimalDriver.model;", "package com.angeles.MinimalDriver.model;\n// TODO add poll support")
	rows := driverRows(t, dir)
	wantRow(t, rows, "FAIL", "driver-scaffold", "BMinimalDriverDevice.java:", "only calls pingOk()")
	wantRow(t, rows, "WARN", "driver-scaffold", "MinimalDriverCodec.java:", "wizard stub")
}

// TestDriverCommConfig: a network or device slot that holds a bare framework comm config FAILs (§ 1.9);
// the same slot holding the driver's subclass passes, and a bare config on a non-driver class is not looked at.
func TestDriverCommConfig(t *testing.T) {
	dir := copyFixture(t)
	edit(t, dir, fxRt+"BMinimalDriverNetwork.java", "  type = \"BMinimalDriverUdpCommConfig\",\n  defaultValue = \"new BMinimalDriverUdpCommConfig()\"",
		"  type = \"BUdpCommConfig\",\n  defaultValue = \"new BUdpCommConfig()\"")
	edit(t, dir, fxRt+"BMinimalDriverDevice.java", "@NiagaraProperty(\n  name = \"points\",",
		"@NiagaraProperty(name = \"tcpConfig\", type = \"com.tridium.ndriver.datatypes.BTcpCommConfig\")\n@NiagaraProperty(\n  name = \"points\",")
	edit(t, dir, fxRt+"BMinimalDriverPointFolder.java", "@NiagaraType", "@NiagaraType\n@NiagaraProperty(name = \"c\", type = \"BUdpCommConfig\", defaultValue = \"new BUdpCommConfig()\")")
	rows := driverRows(t, dir)
	wantRow(t, rows, "FAIL", "driver-comm-config", "BMinimalDriverNetwork.java:", "udpConfig", "BUdpCommConfig")
	wantRow(t, rows, "FAIL", "driver-comm-config", "BMinimalDriverDevice.java:", "tcpConfig", "BTcpCommConfig")
	for _, r := range rows {
		if strings.Contains(r, "BMinimalDriverPointFolder.java") {
			t.Errorf("non-network class flagged: %s", r)
		}
	}
}

// TestDriverDiscoveryAlive: a getDiscoveryObjects that never asks job.isAlive() FAILs (§ 1.11 step 1),
// in each discovery host; an isAlive() left only in a comment or a string does not count.
func TestDriverDiscoveryAlive(t *testing.T) {
	dir := copyFixture(t)
	edit(t, dir, fxRt+"BMinimalDriverNetwork.java", "if (job == null || !job.isAlive()) return null;", "if (job == null) return null; // !job.isAlive()")
	edit(t, dir, fxRt+"BMinimalDriverNetwork.java", "      if (!job.isAlive()) return null;\n", "      String s = \"isAlive()\";\n")
	edit(t, dir, fxRt+"BMinimalDriverPointDeviceExt.java", "if (job == null || !job.isAlive()) return null;", "if (job == null) return null;")
	edit(t, dir, fxRt+"BMinimalDriverPointDeviceExt.java", "      if (!job.isAlive()) return null;\n", "")
	rows := driverRows(t, dir)
	wantRow(t, rows, "FAIL", "driver-discovery-alive", "BMinimalDriverNetwork.java:", "getDiscoveryObjects")
	wantRow(t, rows, "FAIL", "driver-discovery-alive", "BMinimalDriverPointDeviceExt.java:", "getDiscoveryObjects")
}

const tcpConfigSrc = "package com.angeles.MinimalDriver;\nimport com.tridium.ndriver.datatypes.BTcpCommConfig;\n@NiagaraType\n%s\npublic class %s extends BTcpCommConfig\n{\n  public %s() { %s }\n}\n"

func tcpConfig(name, ann, ctor string) string {
	return fmt.Sprintf(tcpConfigSrc, ann, name, name, ctor)
}

// TestDriverServerEnabled: a TCP comm config subclass that defaults serverEnabled to true, by a property
// override or a setServerEnabled(true) call, FAILs (§ 1.6); one that keeps the false default passes.
func TestDriverServerEnabled(t *testing.T) {
	dir := copyFixture(t)
	write(t, dir, map[string]string{
		fxRt + "BAnnTcpCommConfig.java":  tcpConfig("BAnnTcpCommConfig", "@NiagaraProperty(name = \"serverEnabled\", type = \"boolean\", defaultValue = \"true\", override = true)", ""),
		fxRt + "BCtorTcpCommConfig.java": tcpConfig("BCtorTcpCommConfig", "", "setServerEnabled(true);"),
	})
	rows := driverRows(t, dir)
	wantRow(t, rows, "FAIL", "driver-server-enabled", "BAnnTcpCommConfig.java:", "serverEnabled")
	wantRow(t, rows, "FAIL", "driver-server-enabled", "BCtorTcpCommConfig.java:", "setServerEnabled(true)")

	dir = copyFixture(t)
	write(t, dir, map[string]string{
		fxRt + "BOkTcpCommConfig.java": tcpConfig("BOkTcpCommConfig", "@NiagaraProperty(name = \"serverEnabled\", type = \"boolean\", defaultValue = \"false\", override = true)", "// setServerEnabled(true);"),
	})
	wantRow(t, driverRows(t, dir), "PASS", "driver-server-enabled", "1 TCP comm config")
}

const pointMgrAgent = `@NiagaraType(agent = @AgentOn(types = { "MinimalDriver:MinimalDriverPointDeviceExt", "MinimalDriver:MinimalDriverPointFolder" }))`

// TestDriverManager: a point manager must be @AgentOn this driver's point device ext (§ 1.6), and a
// BBasicNetwork driver must ship a manager @AgentOn its network or device (PD-01, § 1.3).
func TestDriverManager(t *testing.T) {
	t.Run("point-manager-wrong-target", func(t *testing.T) {
		dir := copyFixture(t)
		edit(t, dir, fxWb+"BMinimalDriverPointManager.java", `"MinimalDriver:MinimalDriverPointDeviceExt"`, `"MinimalDriver:Nope"`)
		wantRow(t, driverRows(t, dir), "FAIL", "driver-manager", "BMinimalDriverPointManager.java:", "MinimalDriver:MinimalDriverPointDeviceExt")
	})
	t.Run("point-manager-no-agent", func(t *testing.T) {
		dir := copyFixture(t)
		edit(t, dir, fxWb+"BMinimalDriverPointManager.java", pointMgrAgent, "@NiagaraType")
		wantRow(t, driverRows(t, dir), "FAIL", "driver-manager", "BMinimalDriverPointManager.java:", "MinimalDriver:MinimalDriverPointDeviceExt")
	})
	t.Run("pd01-basic-network-no-manager", func(t *testing.T) {
		dir := copyFixture(t)
		edit(t, dir, fxRt+"BMinimalDriverNetwork.java", "  extends BNNetwork\n", "  extends BBasicNetwork\n")
		wantRow(t, driverRows(t, dir), "FAIL", "driver-manager", "PD-01", "MinimalDriver:MinimalDriverNetwork")
	})
	t.Run("pd01-basic-network-with-manager", func(t *testing.T) {
		dir := copyFixture(t)
		edit(t, dir, fxRt+"BMinimalDriverNetwork.java", "  extends BNNetwork\n", "  extends BBasicNetwork\n")
		write(t, dir, map[string]string{fxWb + "BMinimalDriverDeviceManager.java": "package com.angeles.MinimalDriver.wb;\n" +
			"@NiagaraType(agent = @AgentOn(types = \"MinimalDriver:MinimalDriverNetwork\"))\npublic class BMinimalDriverDeviceManager extends BAbstractManager {}\n"})
		rows := driverRows(t, dir)
		wantRow(t, rows, "PASS", "driver-manager", "PD-01", "BMinimalDriverDeviceManager")
		for _, r := range rows {
			if strings.HasPrefix(r, "FAIL") && strings.Contains(r, "driver-manager") {
				t.Errorf("unexpected %s", r)
			}
		}
	})
}
