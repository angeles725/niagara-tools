package ngdeploy

import "strings"

// gradleArgs are the arguments of every gradlew call: the three properties, then the tasks.
func (d *deploy) gradleArgs(tasks []string) []string {
	return append([]string{
		"-Pniagara_home=" + d.env("NIAGARA_HOME"),
		"-Pniagara_user_home=" + d.env("NIAGARA_USER_HOME"),
		"-Porg.gradle.java.installations.paths=" + d.env("JAVA_HOME"),
	}, tasks...)
}

// build runs gradlew clean and jar for the profiles of the mode; a gradlew that fails is exit 30.
// --no-deploy ends the run here with the jars in build/libs.
func (d *deploy) build() int {
	mode := d.raw[optMode]
	var tasks []string
	for _, verb := range buildVerbs {
		for _, profile := range profilesOf[mode] {
			tasks = append(tasks, ":"+d.module+"-"+profile+":"+verb)
		}
	}
	// The bash tool joined the list with its IFS, a newline: the line is kept as it printed it.
	d.say(msgBuild, mode, strings.Join(tasks, "\n"))
	if d.show(d.env("GRADLEW_PATH"), d.gradleArgs(tasks)...) != 0 {
		return d.fail(ExitBuild, msgBuildFailed)
	}
	d.say(msgBuildOK)
	if d.isOn(optNoDeploy) {
		d.say(msgNoDeploy)
		return ExitOK
	}
	return proceed
}
