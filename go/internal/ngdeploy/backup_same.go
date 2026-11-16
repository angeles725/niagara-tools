package ngdeploy

import "strings"

// sameDirOptions are the tar options of a backup when the station is <NIAGARA_HOME>/modules, whose jars the
// build has replaced by the time the backup runs (just before the copy). For each jar of the module that
// asideHome moved aside, the file aside is stored under the jar's name, and, in a full backup, the jar the
// build put in its place is left out; a jar the build installed where the directory had none is left out
// too. Only this module's own jars are named: another module's files, a file aside of its own included, are
// archived as they are. prefix is "" when the members are the files of the directory (a lightweight
// backup) and "<directory>/" when the member is the directory itself (a full backup).
func (d *deploy) sameDirOptions(prefix string) (options []string) {
	full := prefix != ""
	var exclude []string
	for _, jar := range d.homeJars {
		name := prefix + jar.name
		switch {
		case jar.moved:
			options = append(options, "--transform=s,^"+breQuote(name+asideSuffix)+"$,"+sedReplacement(name)+",")
			if full {
				exclude = append(exclude, "--exclude="+name)
			}
		case !jar.had && full:
			exclude = append(exclude, "--exclude="+name)
		}
	}
	if len(exclude) > 0 { // each pattern is the one member path, matched whole and literally
		options = append(append(options, "--anchored", "--no-wildcards"), exclude...)
	}
	return options
}

// breQuote is s as a literal in the sed basic regular expression of tar --transform, with "," the delimiter.
func breQuote(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`\.*[]^$,`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// sedReplacement is s as the literal replacement of a sed s command with "," the delimiter.
func sedReplacement(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`\&,`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
