package buildmodule

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// Names the chain reads or runs; one place, so a renamed file changes one line.
const (
	gateScript      = "verify-module.sh"
	preflightScript = "preflight.sh"
	reportScript    = "report-module.sh"
	werrorInit      = "lib/werror.init.gradle"
	mirrorMarker    = modjar.MirrorMarker
	m2Repository    = "etc/m2/repository"
	// aliasProperty is the gradle property the kit's root signing block takes the signing alias from
	// (templates/signing-root.gradle.kts); gradleEnvPrefix is how the environment sets a gradle property.
	aliasProperty   = modjar.AliasProperty
	gradleEnvPrefix = modjar.GradleEnvPrefix
	// devSigner is the name the signature files of the signing plugin's default development key carry
	// (META-INF/NIAGARA4.SF): a signer a deployed jar never makes the expected one.
	devSigner = modjar.DevSigner
	// m2Depth is how deep under etc/m2/repository a plugin version directory is looked for.
	m2Depth = 8
)

// autoProfiles are the profiles auto-detection tries, in build order.
var autoProfiles = []string{"rt", "ux", "wb"}

// uiProfiles are the values --ui-profile takes.
var uiProfiles = map[string]bool{"hmi": true, "lan": true, "both": true, "unknown": true}

// sourceSuffixes mark a profile with sources: a gradle file alone is a scaffold.
var sourceSuffixes = []string{".java", ".js", ".html"}

// prunedBelow are the directories never entered when listing the gradle roots below a repo root
// (besides every dot directory).
var prunedBelow = map[string]bool{"node_modules": true, "build": true}

// Gradle configuration readers. Each matches one line.
var (
	// settingsPluginVersion is the gradlePluginVersion declared in settings.gradle.kts.
	settingsPluginVersion = regexp.MustCompile(`gradlePluginVersion[[:space:]]*:[[:space:]]*String[[:space:]]*=[[:space:]]*"([0-9][^"]*)"`)
	// propertiesPlugin is a niagaraPluginVersion line of gradle.properties.
	propertiesPlugin = regexp.MustCompile(`^[[:space:]]*niagaraPluginVersion[[:space:]]*=`)
	// settingsPluginDefault is the niagaraPluginVersion default of settings.gradle.kts: the getOrElse("X")
	// argument (two or more digits and dots) after the name, the scaffold's
	// providers.gradleProperty("niagaraPluginVersion").getOrElse("X") form included.
	settingsPluginDefault = regexp.MustCompile(`niagaraPluginVersion(?:"\))?[^)]*getOrElse\("([0-9][0-9.]+)"\)`)
)

// Gradle log classifiers. Each matches one line.
var (
	// logCopyLocked: the plugin's post-jar copy into modules/ failed (either path separator: the Windows
	// gradlew.bat route prints backslashes). [ev: retro n4-course-signed-jars-gate-fix Δ5]
	logCopyLocked = regexp.MustCompile(`FileAlreadyExistsException.*modules[/\\].*\.jar`)
	// logCleanLocked: :clean could not delete a modules/<jar> a running station holds.
	logCleanLocked = regexp.MustCompile(`Unable to delete .*modules/.*\.jar`)
)

// logSlotErrors is what compileJava prints for slots Slot-o-Matic never generated.
const logSlotErrors = "have you run slot-o-matic"

// blanks is the bash [[:space:]] class.
const blanks = " \t\n\v\f\r"
