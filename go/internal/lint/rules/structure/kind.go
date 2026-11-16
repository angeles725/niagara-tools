package structure

// The declared module kinds (types/structure.md § Module map, the `type:` header line). A kind is read from
// the module map only: the lint never infers one from the files a part lacks, so a part that is empty by
// accident still FAILs.
const (
	kindKey     = "type:"   // the MODULE-MAP.md header line that declares the module's kind
	kindTheme   = "theme"   // types/theme.md: a zero-Java -ux part of CSS, NSS, fonts and icons
	kindLibrary = "library" // types/utility-lib.md: an -rt part others depend on, with no palette
	themeSheet  = "*.nss"   // what a declared theme part must hold: the picker lists nss/*.nss files
)

// The L5 and L9 row texts. l5Missing and l5Empty are the bash lint's, byte for byte.
const (
	l5Missing = "L5: module.palette missing for -rt profile"
	l5Empty   = "L5: module.palette has no named component entries (empty palette — see B788)"
	l9Empty   = "L9: empty skeleton artifact (0 Java classes, empty palette — delete or populate the profile"
	// l9ThemeHint closes l9Empty on a -ux part, the only part a theme has; a -wb part keeps the bash text.
	l9ThemeHint = "; a theme module declares `type: theme` in its MODULE-MAP.md header, types/theme.md § 7)"
	l9Theme     = "L9: declared theme (type: theme in MODULE-MAP.md) holds no .nss style sheet (a theme without nss/*.nss is never listed — add it or delete the profile)"
)
