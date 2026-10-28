package preflight

import (
	"os"
	"strings"
	"testing"
)

// gradle76 is `gradlew -q javaToolchains` as Gradle 7.6 prints it: an Options block, then one block per toolchain.
const gradle76 = ` + Options
     | Auto-detection:     Enabled
     | Auto-download:      Disabled

 + Eclipse Temurin JRE 1.8.0_412-b08
     | Location:           /usr/lib/jvm/temurin-8-jre
     | Language Version:   8
     | Vendor:             Eclipse Temurin
     | Is JDK:             false
     | Detected by:        Common Linux Locations

 + Eclipse Temurin JDK 17.0.11+9
     | Location:           /usr/lib/jvm/temurin-17
     | Language Version:   17
     | Vendor:             Eclipse Temurin
     | Is JDK:             true
     | Detected by:        Current JVM
`

const fullJDK8 = `
 + Zulu JDK 1.8.0_402-b06
     | Location:           /opt/zulu8
     | Language Version:   8
     | Vendor:             Azul Zulu
     | Is JDK:             true
     | Detected by:        Gradle property 'org.gradle.java.installations.paths'
`

// TestToolchains: the opt-in --toolchains row (K6): a full JDK 8 listed by gradlew javaToolchains passes; a JRE 8
// or a JDK 17 alone fails; gradle.properties paths pinned to a full JDK 8 passes without running gradle; no
// executable gradlew, or a listing with no toolchain, is SKIP with its reason; without the flag there is no row.
func TestToolchains(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/", "gr/gradlew", "pin/gradlew", "pin/gradle.properties=org.gradle.java.installations.paths="+d+"/jdk8\n",
		"jdk8/release=JAVA_VERSION=\"1.8.0_402\"\n", "jdk8/bin/javac", "nogw/")
	for _, g := range []string{d + "/gr/gradlew", d + "/pin/gradlew"} {
		if err := os.Chmod(g, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	const pre = "toolchains    "
	for _, c := range []struct {
		name, gr, listing, want string
		flag, ranGradle         bool
	}{
		{"jre8 and jdk17 only", "gr", gradle76, "FAIL  " + pre + "gradlew javaToolchains lists no full JDK 8 (a 4.15 build requires one, a JRE is not enough): install a JDK 8 or set org.gradle.java.installations.paths in gradle.properties", true, true},
		{"full jdk 8 listed", "gr", gradle76 + fullJDK8, "PASS  " + pre + "full JDK 8 listed by gradlew javaToolchains: /opt/zulu8", true, true},
		{"nothing listed", "gr", "FAILURE: Build failed with an exception.\n", "SKIP  " + pre + "gradlew javaToolchains listed no toolchain (gradle failed or printed nothing)", true, true},
		{"paths pinned", "pin", "", "PASS  " + pre + "org.gradle.java.installations.paths pins a full JDK 8: " + d + "/jdk8 (gradle.properties)", true, false},
		{"no gradlew", "nogw", "", "SKIP  " + pre + "no executable gradlew in " + d + "/nogw", true, false},
	} {
		f := &fakeHost{out: map[string]string{"/gradlew": c.listing}}
		args := []string{"--jvm-dir", d + "/none", d + "/nh", d + "/" + c.gr}
		if c.flag {
			args = append([]string{"--toolchains"}, args...)
		}
		_, out, _ := pf(f.host(), nil, args...)
		if !strings.Contains(out, c.want+"\n") {
			t.Errorf("%s:\n%s\nwant row %q", c.name, out, c.want)
		}
		ran := false
		for _, call := range f.calls {
			if strings.HasPrefix(call, "gradlew") {
				ran = true
				if want := "gradlew -q -p " + d + "/" + c.gr + " javaToolchains -Porg.gradle.java.installations.auto-detect=true"; call != want {
					t.Errorf("%s: ran %q, want %q", c.name, call, want)
				}
			}
		}
		if ran != c.ranGradle {
			t.Errorf("%s: gradle ran %v, want %v", c.name, ran, c.ranGradle)
		}
	}
	f := &fakeHost{out: map[string]string{"/gradlew": gradle76}}
	if _, out, _ := pf(f.host(), nil, "--jvm-dir", d+"/none", d+"/nh", d+"/gr"); strings.Contains(out, "toolchains") {
		t.Errorf("no --toolchains, got a toolchains row:\n%s", out)
	}
}
