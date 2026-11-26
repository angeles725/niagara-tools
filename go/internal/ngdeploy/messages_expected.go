package ngdeploy

// Messages of the recorded expected signer (expected_signer.go).
const (
	// msgSignerFromRecord (stdout): %[1]s the alias, %[2]s the module, %[3]s the file.
	msgSignerFromRecord = "expected signer: %[1]s (expected_signer of %[2]s in %[3]s)"
	// msgSignerMismatch: %[1]s the configured alias, %[2]s the module, %[3]s the recorded alias, %[4]s the file.
	msgSignerMismatch = "niagaraSigningAlias=%[1]s is configured but the build-state envelope of %[2]s " +
		"records expected_signer: %[3]s (%[4]s).\n" +
		"  Deploy with that alias (--gradle-prop niagaraSigningAlias=%[3]s) or, if the signer changed, " +
		"update expected_signer there."
	// msgSignerInvalid: %[1]s the reason (the sweep's text), %[2]s the file.
	msgSignerInvalid = "%[1]s in %[2]s"
	// msgStateUnreadable: %[1]s the file, %[2]s the reason.
	msgStateUnreadable = "--build-state %[1]s cannot be read (%[2]s)"
	// msgBuildStateEmpty: %s the flag.
	msgBuildStateEmpty = "%s requires a file name, not an empty value"
)
