package ngdeploy

// The lines about <NIAGARA_HOME>/modules, the directory gradle's :jar step installs the built jars into
// before any check of the deploy has run (home.go). The states of a jar and the outcomes are the copy's
// (messages_copy.go), said of homeSite.
const (
	// msgRealHome takes NIAGARA_HOME, the marker of a mirror and the flag that opts in.
	msgRealHome = "NIAGARA_HOME %s is a real niagara_home (no %s): gradle's :jar step would replace the jars of its modules/ before any check runs; " +
		"build against a mirror (toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>), or pass %s"
	// msgSameDir takes the directory both settings name.
	msgSameDir = "STATION_MODULES_DIR is <NIAGARA_HOME>/modules (%s): the build installs the jars there, so the copy step only verifies them"
	// msgSameDirNoDeploy takes that directory, when --no-deploy (or NO_DEPLOY=1) names it too.
	msgSameDirNoDeploy = "--no-deploy refused: STATION_MODULES_DIR is <NIAGARA_HOME>/modules (%s): gradle's :jar step installs the built jars there " +
		"and --no-deploy keeps them, so the station would get jars no check has passed; deploy without --no-deploy (the copy step then verifies them), " +
		"or point STATION_MODULES_DIR or NIAGARA_HOME elsewhere; nothing was built or changed"
	// msgHomeAsideFound is msgAsideFound for <NIAGARA_HOME>/modules, before anything is built.
	msgHomeAsideFound = "build refused: a deploy that did not finish left previous jars aside in %s; nothing was built or changed"
	// msgHomeNotAJar is msgCopyFailed for <NIAGARA_HOME>/modules, before anything is built.
	msgHomeNotAJar = "build refused for %s; %s"
	// msgHomeAsideFailed takes the jar, the directory, why and an outcome.
	msgHomeAsideFailed = "build not started: %s could not be moved aside in %s (%s); %s"
	// noticeHomeRestored takes the directory, after a deploy that stopped put its jars back.
	noticeHomeRestored = "NOTICE: the jars the build installed in %s were taken out again: it holds the module jars it held before the build"
	// noticeHomeForeignLeft is that NOTICE when a jar of another origin (warnHomeNotOurs) was left in place.
	noticeHomeForeignLeft = "NOTICE: the jars the build installed in %s were taken out again and the module jars moved aside for it were put back; " +
		"the jar named above was not there before the build and is not the jar this run built: it was left in place"
	// noticeHomeMovedBack is that NOTICE when the build installed no jar: only the move aside was undone.
	noticeHomeMovedBack = "NOTICE: the build installed no jar in %s; the module jars moved aside for it were moved back: it holds the module jars it held before the build"
	// warnHomeNotOurs takes a jar the directory did not have that is not the one this run built, and the directory.
	warnHomeNotOurs = "WARN %s in %s was not there before the build and is not the jar this run built (build/libs): it is left where it is"
	// The warnings of a deploy that went through and could not tidy the directory: the file aside, the
	// directory and why; then the jar, the directory, its file aside and why.
	warnHomeAsideLeft = "WARN the previous jar %s in %s could not be removed (%s): delete it, the next deploy refuses while it is there"
	warnHomeNotBack   = "WARN %s in %s could not be moved back from %s (%s): rename it back, the next deploy refuses while it is there"
	// msgInterrupted takes SIGINT or SIGTERM; the exit code is 128 + the signal.
	msgInterrupted = "interrupted (%s) before the copy; nothing was copied to the station"
	// msgCopySame takes the jars and the directory; msgCopyNotInstalled the jar.
	msgCopySame         = "copy: nothing to copy, the build installed %s in %s (the same bytes as build/libs)"
	msgCopyNotInstalled = "copy failed for %s: STATION_MODULES_DIR is <NIAGARA_HOME>/modules and the jar the build left there is missing or is not the one in build/libs"

	// guardRealHome is what INSTALL_INTO_HOME switches off (msgEnvFileGuarded).
	guardRealHome = "the refusal of a real niagara_home"
	// homeModules is the directory of a niagara_home that holds its module jars.
	homeModules = "/modules"
)

// homeSite is <NIAGARA_HOME>/modules while the build and the checks run.
func homeSite(dir string) site {
	return site{holder: dir, since: "the build", before: "before the next build or deploy"}
}
