package ngdeploy

import (
	"regexp"
	"syscall"
)

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
	// digestAttribute is a digest attribute of a manifest section: SHA-256-Digest, SHA-256-Digest-Manifest...
	digestAttribute = regexp.MustCompile(`^[A-Za-z0-9_-]+-Digest(-[A-Za-z0-9_-]+)?:`)
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
	// stagedSuffix ends the name a new jar has beside its target until it is renamed. It does not end in
	// .jar: the station must not read a staged file as a module.
	stagedSuffix = ".ng-deploy-new"
	// asideSuffix ends the name the station's jar has beside its place while the new one is renamed in.
	asideSuffix = ".ng-deploy-old"
	// moduleXML and indexHTML are the jar members the verify step reads.
	moduleXML = "META-INF/module.xml"
	indexHTML = "rc/index.html"
	// detectionOff and detectionStrict are the SLOTOMATIC_DETECTION values with an effect.
	detectionOff    = "off"
	detectionStrict = "strict"
)

// What two builds of the same sources differ in, and the version-bump precheck therefore does not compare
// (jarcontent.go, descriptor.go), beside the digests above: the signature files, which are the entries
// directly under metaInf whose name ends, in any case, in one of signatureExts; and buildStamps, the
// attributes the vendor's plugin stamps on the root element of META-INF/module.xml at every build: when it
// ran and on which machine.
var (
	signatureExts = []string{".SF", ".RSA", ".DSA", ".EC"}
	buildStamps   = map[string]bool{"buildMillis": true, "buildHost": true}
)

// The limits of that comparison, which keep its memory bounded whatever a jar holds.
const (
	maxJarContent = 1 << 31 // bytes of content a jar may declare, all entries together: over it, exit 52
	maxDescriptor = 1 << 23 // bytes of the manifest and of module.xml, which are read whole: over it, exit 52
	contentChunk  = 1 << 16 // bytes of an entry hashed at a time
)

const (
	metaInf           = "META-INF/"
	manifestName      = "META-INF/MANIFEST.MF"
	manifestEntryName = "Name:"
)

// profilesOf lists the jar profiles each mode builds and deploys, in order.
var profilesOf = map[string][]string{"A": {"rt", "ux"}, "B": {"ux"}, "C": {"rt"}}

// deployable are the profiles whose jars the tool can put in a station, whatever the mode.
var deployable = []string{"rt", "ux"}

// signalNames are the signals the copy watches for, as a message names them.
var signalNames = map[syscall.Signal]string{syscall.SIGINT: "SIGINT", syscall.SIGTERM: "SIGTERM"}

// backupProfiles are the jars of the module a lightweight backup archives when they are deployed.
var backupProfiles = []string{"rt", "ux", "wb"}

// buildVerbs are the gradle tasks of a build, each run for every profile of the mode before the next.
var buildVerbs = []string{"clean", "jar"}

// required are the settings every deploy needs, in the order they are checked.
var required = []string{"MODULE_NAME", "GRADLEW_PATH", "NIAGARA_HOME", "NIAGARA_USER_HOME", "JAVA_HOME", "STATION_MODULES_DIR"}

// expectedTypes names the setting holding the expected type count of a profile's jar.
var expectedTypes = map[string]string{"rt": "EXPECTED_RT_TYPES", "ux": "EXPECTED_UX_TYPES"}
