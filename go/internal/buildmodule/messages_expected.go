package buildmodule

// Messages of the recorded expected signer (signing_expected.go).
const (
	// msgSignerFromRecord (stdout): %[1]s the alias, %[2]s the module, %[3]s the file.
	msgSignerFromRecord = "==> expected signer: %[1]s (expected_signer of %[2]s in %[3]s)"
	// msgSignerMismatch (stderr): %[1]s the configured alias, %[2]s the module, %[3]s the recorded alias, %[4]s the file.
	msgSignerMismatch = "build.sh: FAIL — niagaraSigningAlias=%[1]s is configured but the build-state envelope of %[2]s " +
		"records expected_signer: %[3]s (%[4]s).\n" +
		"  Build with that alias (--gradle-prop niagaraSigningAlias=%[3]s) or, if the signer changed, " +
		"update expected_signer there."
	// msgSignerInvalid (stderr): %[1]s the reason (the sweep's text), %[2]s the file.
	msgSignerInvalid = "build.sh: FAIL — %[1]s in %[2]s"
	// msgStateUnreadable (stderr): %[1]s the file, %[2]s the reason.
	msgStateUnreadable = "build.sh: FAIL — --build-state %[1]s cannot be read (%[2]s)"
)
