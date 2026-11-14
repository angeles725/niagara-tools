package ngdeploy

import "regexp"

// What the deploy looks for and builds, apart from the code that uses it.
var (
	// annotation is a Niagara annotation Slot-o-Matic generates code from.
	annotation = regexp.MustCompile(`@Niagara(Type|Property|Action|Topic|Singleton)`)
	// annotationChange is an added or removed line of a diff that carries one.
	annotationChange = regexp.MustCompile(`^[+-][[:space:]]*@Niagara(Type|Property|Action|Topic|Singleton)`)
	// objectName is a full git object name as `git rev-parse HEAD` prints it: 40 hex digits (SHA-1) or 64
	// (SHA-256), lower case. Nothing else is written to .last-deploy-sha or handed to git from it.
	objectName = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
	// wholeNumber is a count as the deploy accepts it: decimal digits, no sign, no leading zero.
	wholeNumber = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
	// digits is a switch value: bash compared it to 1 as a number.
	digits = regexp.MustCompile(`^[0-9]+$`)
)

const (
	// typeEntry marks a line of module.xml that declares a type. The trailing space is load-bearing: the
	// <types> wrapper element must not count.
	typeEntry = "<type "
	// javaSuffix is the name of a source file scanned for annotations.
	javaSuffix = ".java"
	// diffPathspec limits the annotation diff to Java sources.
	diffPathspec = "*/src/com/**/*.java"
	// fallbackBaseline is the diff baseline when .last-deploy-sha names no commit.
	fallbackBaseline = "HEAD~1"
	// lastDeployFile records the commit of the last deploy, in the working directory.
	lastDeployFile = ".last-deploy-sha"
	// backupDir holds the backups, in the working directory; backupStamp is the time layout of their names.
	backupDir    = "_backups"
	backupStamp  = "20060102-150405"
	backupSuffix = ".tar.gz"
	// moduleXML and indexHTML are the jar members the verify step reads.
	moduleXML = "META-INF/module.xml"
	indexHTML = "rc/index.html"
	// detectionOff and detectionStrict are the SLOTOMATIC_DETECTION values with an effect.
	detectionOff    = "off"
	detectionStrict = "strict"
)

// profilesOf lists the jar profiles each mode builds and deploys, in order.
var profilesOf = map[string][]string{"A": {"rt", "ux"}, "B": {"ux"}, "C": {"rt"}}

// backupProfiles are the jars of the module a lightweight backup archives when they are deployed.
var backupProfiles = []string{"rt", "ux", "wb"}

// buildVerbs are the gradle tasks of a build, each run for every profile of the mode before the next.
var buildVerbs = []string{"clean", "jar"}

// required are the settings every deploy needs, in the order they are checked.
var required = []string{"MODULE_NAME", "GRADLEW_PATH", "NIAGARA_HOME", "NIAGARA_USER_HOME", "JAVA_HOME", "STATION_MODULES_DIR"}

// expectedTypes names the setting holding the expected type count of a profile's jar.
var expectedTypes = map[string]string{"rt": "EXPECTED_RT_TYPES", "ux": "EXPECTED_UX_TYPES"}
