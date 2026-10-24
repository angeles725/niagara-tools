package preflight

import (
	"os"
	"strings"
	"testing"
)

func TestJdk8(t *testing.T) {
	d := t.TempDir()
	mk(t, d, "nh/", "gr/", "jvm/a-11/release=JAVA_VERSION=\"11.0.2\"\n", "jvm/b-8/release=JAVA_VERSION=\"1.8.0_1\"\n",
		"jvm/c-8/release=JAVA_VERSION=\"1.8.0_2\"\n", "wsl/.h/release=JAVA_VERSION=\"1.8.0\"\n", "wsl/w/bin/java", "jh11/release=JAVA_VERSION=\"11\"\n",
		"jh11/bin/java", "empty/")
	for _, java := range []string{d + "/wsl/w/bin/java", d + "/jh11/bin/java"} {
		if err := os.Chmod(java, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f := &fakeHost{out: map[string]string{"/bin/java": `openjdk version "1.8.0_412"`}}
	for _, c := range []struct{ jvm, javaHome, want string }{
		{d + "/jvm", "", "PASS  jdk8          JDK 8 found: " + d + "/jvm/b-8/"},
		{d + "/jvm/", "", "PASS  jdk8          JDK 8 found: " + d + "/jvm//b-8/"},
		{d + "/wsl", "", "PASS  jdk8          JDK 8 found: " + d + "/wsl/w/"}, // .h is not globbed; bin/java says 1.8
		{d + "/empty", d + "/jvm/c-8", "PASS  jdk8          JDK 8 found: " + d + "/jvm/c-8"},
		{d + "/empty", d + "/wsl/w", "PASS  jdk8          JDK 8 found: " + d + "/wsl/w"},
		// a JAVA_HOME release file that is not 1.8 decides: its bin/java is not asked
		{d + "/empty", d + "/jh11", "FAIL  jdk8          JDK 8 not found under " + d + "/empty (or JAVA_HOME); use --jvm-dir to override"},
		{d + "/none", "", "FAIL  jdk8          JDK 8 not found under " + d + "/none (or JAVA_HOME); use --jvm-dir to override"},
	} {
		env := map[string]string{"JAVA_HOME": c.javaHome, "HOME": d + "/jvm"} // $HOME is never searched
		if _, out, _ := pf(f.host(), env, "--jvm-dir", c.jvm, d+"/nh", d+"/gr"); !strings.Contains(out, c.want+"\n") {
			t.Errorf("--jvm-dir %s JAVA_HOME=%q:\n%s\nwant row %q", c.jvm, c.javaHome, out, c.want)
		}
	}
	for _, call := range f.calls {
		if strings.HasPrefix(call, "java") && call != "java -version" {
			t.Errorf("ran %q, want java -version", call)
		}
	}
}
