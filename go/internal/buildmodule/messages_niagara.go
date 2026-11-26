package buildmodule

// Messages of the recorded Niagara target (niagara_target.go).
const (
	// msgNiagaraRecord (stdout): %[1]s the file, %[2]s the module, %[3]s what it records, %[4]s what the build uses.
	msgNiagaraRecord = "build.sh: INFO — %[1]s records %[3]s for %[2]s: building with %[4]s"
	// msgTargetHomeDiffers (stderr): %[1]s the module, %[2]s the recorded target, %[3]s its major.minor, %[4]s the
	// file, %[5]s the installation as worded, %[6]s its version.
	msgTargetHomeDiffers = "build.sh: FAIL — %[1]s is recorded for Niagara %[2]s (niagara_target in %[4]s) but %[5]s is Niagara %[6]s.\n" +
		"  Build against an installation of %[3]s (mirror it with toolbelt/mirror-niagara-home.sh) or, if the station " +
		"moved, update niagara_target there."
	// msgTargetHomeUnread (stderr): %[1]s the installation as worded, %[2]s the target, %[3]s the file, %[4]s the reason.
	msgTargetHomeUnread = "build.sh: WARN — niagara_target %[2]s (%[3]s) cannot be compared with %[1]s: %[4]v"
	// msgTargetFlagDiffers (stderr): %[1]s the flag value, %[2]s the recorded target, %[3]s its major.minor,
	// %[4]s the module, %[5]s the file.
	msgTargetFlagDiffers = "build.sh: FAIL — --target-version %[1]s differs from niagara_target %[2]s recorded for %[4]s in %[5]s " +
		"(want %[3]s).\n  Drop --target-version (it defaults to the record) or, if the station moved, update niagara_target there."
	// msgPluginFlagDiffers (stderr): %[1]s the flag or environment value, %[2]s the recorded plugin version,
	// %[3]s the module, %[4]s the file.
	msgPluginFlagDiffers = "build.sh: FAIL — --plugin-version (or NIAGARA_PLUGIN_VERSION) %[1]s differs from plugin_version %[2]s " +
		"recorded for %[3]s in %[4]s.\n  Drop it (it defaults to the record) or, if the plugin moved, update plugin_version there."
)
