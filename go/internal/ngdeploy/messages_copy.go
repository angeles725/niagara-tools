package ngdeploy

// The lines about a modules directory whose jars are replaced all or none: the copy into the station, and
// the jars of the module in <NIAGARA_HOME>/modules while the build and the checks run (home.go). A site says
// how a line names the directory.
const (
	msgCopyOK = "copy ok"
	// msgCopyFailed takes the jar that could not be staged or renamed (with the reason after ": " when cp did
	// not print one) and what the station holds now: one of the outcomes below.
	msgCopyFailed = "copy failed for %s; %s"
	// The outcomes take the holder of a site and what its jars are compared with.
	outcomeUnchanged = "nothing was replaced: %s holds the jars it held before %s"
	outcomeUndone    = "what was already replaced was put back: %s holds the jars it held before %s"
	// msgCopyInterrupted takes SIGINT or SIGTERM and the same outcomes; the exit code is 128 + the signal.
	msgCopyInterrupted = "copy interrupted (%s); %s"
	// whyNotAJar takes what the station holds under the name of a jar (the kinds below).
	whyNotAJar     = "the station holds %s under that name, not a jar file"
	kindDirectory  = "a directory"
	kindOther      = "something that is not a regular file"
	kindLinkTo     = "a symbolic link to "
	kindDangling   = "a symbolic link to nothing"
	kindUnreadable = "something that cannot be examined"
	// outcomeMixed takes the holder, what its jars are compared with and the state of each jar (below),
	// joined by ", "; mixedPutBack follows it when a state names a file aside, and takes the suffix of those
	// files and by when they must be back.
	outcomeMixed  = "%s does NOT hold the jars it held before %s: %s"
	mixedPutBack  = "; put each named %s file back under its jar name %s"
	stateNew      = "%s is the NEW jar and the previous one is in %s (%s)" // the jar, the file aside, why it was not moved back
	stateMissing  = "%s is MISSING and the previous one is in %s (%s)"
	stateExtra    = "%s, which %s did not have, could not be removed (%s): remove it %s" // the jar, the holder, why, by when
	statePrevious = "%s is the previous jar"
	// warnAsideLeft takes the file aside that a finished copy could not remove, and why.
	warnAsideLeft = "WARN the jars are copied, but the previous jar %s could not be removed (%s): delete it, the next deploy refuses while it is in the station"
	// msgAsideFound takes the station directory; one state line per jar and the ways out follow it,
	// indented. The states take the jar, its file aside and the suffix; the ways out the suffix twice.
	msgAsideFound     = "copy refused: a deploy that did not finish left previous jars aside in %s; nothing was copied or changed"
	stateAsideWithJar = "%s present, %s present: the jar may be the new one of that deploy; the %s file is the previous jar"
	stateAsideNoJar   = "%s missing, %s present: the %s file is the previous jar"
	msgAsideWaysOut   = "to put the previous jars back, rename each %s file over its jar; to keep what is there, delete the %s files; then deploy again"
	indent            = "  "
)

// site is a modules directory whose jars are replaced all or none, as the lines about it name it.
type site struct {
	holder string // what holds the jars
	since  string // what its jars are compared with
	before string // by when a jar left in the wrong state must be put right
}

// stationSite is STATION_MODULES_DIR during the copy.
var stationSite = site{holder: "the station", since: "the deploy", before: "before the station restarts"}
