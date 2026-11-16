package niagaratest

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// pluginVersion is the one com.tridium.niagara gradle plugin version the copy's etc/m2 ships.
func pluginVersion(home string) (string, error) {
	d := home + "/etc/m2/repository/com/tridium/niagara/com.tridium.niagara.gradle.plugin"
	es, err := os.ReadDir(d)
	if err != nil {
		return "", fmt.Errorf("no niagara gradle plugin in etc/m2: %v", err)
	}
	var vs []string
	for _, e := range es {
		if e.IsDir() && versionRe.MatchString(e.Name()) {
			vs = append(vs, e.Name())
		}
	}
	if len(vs) != 1 {
		return "", fmt.Errorf("etc/m2 ships %d niagara gradle plugin versions %q, want exactly 1", len(vs), vs)
	}
	return vs[0], nil
}

// build runs gradle in WSL against the copy: :jar and :moduleTestJar install into the copy's modules/, and
// every home the signing plugin writes (niagara_user_home, user.home) is scratch. Gradle's own cache is the
// caller's (GRADLE_USER_HOME: no 7.6 download per run), and its JDK the WSL JDK 8 (-P overrides a client
// gradle.properties that names a Windows JDK path, which WSL gradle cannot parse).
func (h host) build(ctx context.Context, p *plan, s cli.Stdio) (int, error) {
	v, err := pluginVersion(p.home().wsl)
	if err != nil {
		return -1, err
	}
	for _, d := range []string{"nuh", "gradle-uh", "out/testng"} {
		if err := os.MkdirAll(p.sub(d), 0o755); err != nil {
			return -1, err
		}
	}
	log, err := os.Create(p.sub("out/build.log"))
	if err != nil {
		return -1, err
	}
	defer log.Close()
	code, err := h.run.Run(ctx, Cmd{Name: "bash", Dir: p.gradleRoot, Stdout: log, Stderr: log,
		Args: []string{"./gradlew", "--no-daemon", "--max-workers=4", "-Pniagara_home=" + p.home().wsl,
			"-Pniagara_user_home=" + p.sub("nuh"), "-PniagaraPluginVersion=" + v, "-Porg.gradle.java.installations.paths=" + p.jdk8,
			":" + p.part + ":jar", ":" + p.part + ":moduleTestJar"},
		Env: []string{"JAVA_HOME=" + p.jdk8, "GRADLE_USER_HOME=" + p.gradleUserHome, "JAVA_TOOL_OPTIONS=-Duser.home=" + p.sub("gradle-uh")}})
	if err != nil || code == 0 {
		return -1, err
	}
	row(s, "FAIL", fmt.Sprintf("build: gradle exit %d (last lines of the build log on stderr)", code))
	fmt.Fprint(s.Err, tail(p.sub("out/build.log"), 4096))
	return exitEnv, nil
}

// tail is the last n bytes of a file, read without loading the rest.
func tail(path string, n int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > n {
		_, _ = f.Seek(-n, io.SeekEnd)
	}
	b, _ := io.ReadAll(io.LimitReader(f, n))
	return string(b)
}
