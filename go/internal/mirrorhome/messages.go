// Package mirrorhome builds the mirror of a Niagara install that toolbelt/mirror-niagara-home.sh makes: a
// directory of symlinks to everything in the install, with a real, writable modules/ that links every module
// jar but the ones being built. A build against it never writes to the install a station is running from.
// The header of that script has the why and the two guards.
package mirrorhome

// Exit codes of mirror-niagara-home.sh (unchanged from the bash tool; its header lists them).
const (
	ExitOK      = 0  // mirror ready
	ExitUsage   = 1  // usage, not a niagara_home, a relative source, or a step that failed
	ExitRefused = 20 // a guard refused: nothing was removed
)

// Marker is the file a mirror made by this tool holds; only a directory holding it is ever wiped.
const Marker = ".niagara-mirror"

// The layout of a niagara_home the tool reads.
const (
	repository = "etc/m2/repository" // what makes a directory a niagara_home
	modules    = "modules"           // the one directory that is not linked: the mirror gets a real one
	jarSuffix  = ".jar"
)

// markerFormat is the marker's content: the resolved source and when the mirror was made (markerTime).
const (
	markerFormat = "source=%s\ncreated=%s\n"
	markerTime   = "2006-01-02T15:04:05"
)

// Messages (fmt formats). All go to stderr but msgReady.
const (
	msgUsage       = "usage: mirror-niagara-home.sh <source_niagara_home> <mirror_dir> [exclude-jar ...]\n"
	msgNotHome     = "not a niagara_home (no " + repository + "): %s\n"
	msgInsideSrc   = "refusing: mirror dir is the source install or inside it: %s\n"
	msgContainsSrc = "refusing: mirror dir contains the source install: %s\n"
	msgNotDir      = "refusing: %s exists and is not a directory\n"
	msgLink        = "refusing: %s is a symlink, not a directory made by this script\n"
	msgUnresolved  = "refusing: mirror dir cannot be resolved (a link loop or a directory that cannot be searched): %s\n"
	msgNotMirror   = "refusing to wipe %s: non-empty and not a mirror made by this script (no " + Marker + ")\n"
	msgRelative    = "refusing: source_niagara_home must be an absolute path (the mirror's links would dangle): %s\n"
	msgFault       = "mirror-niagara-home: %s\n"
	msgReady       = "mirror ready: %s  (%d jars linked, modules/ writable, excluded: %s)\n"
	noneExcluded   = "none"
)
