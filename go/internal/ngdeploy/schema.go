package ngdeploy

// precheckSchema runs schema-risk on the deployed -rt source baseline (SCHEMA_BASELINE_DIR) against the
// -rt module directory, for a mode with Java. Without a baseline, --first-deploy (or FIRST_DEPLOY=1 in the
// env file) says there is nothing to diff; otherwise the deploy stops with exit 12. A LOSSY verdict stops
// it with exit 53 unless --accept-lossy; OUTAGE and a schema-risk that could not run always do.
func (d *deploy) precheckSchema() int {
	if !d.hasProfile("rt") {
		return proceed
	}
	baseline := d.env("SCHEMA_BASELINE_DIR")
	if baseline == "" {
		if d.isOn(optFirstDeploy) {
			d.say(msgSchemaFirstDeploy)
			return proceed
		}
		return d.fail(ExitNoBaseline, msgSchemaNeeds)
	}
	if !isDir(baseline) {
		return d.fail(ExitNoBaseline, msgSchemaNotDir, baseline)
	}
	tool := d.raw[optSchemaBin]
	if !isExecutable(tool) {
		return d.fail(ExitSchema, msgSchemaUnavailable, tool)
	}
	switch status := d.show(tool, baseline, d.profileDir("rt")); {
	case status == 0:
		d.say(msgSchemaOK)
	case status == schemaLossy && d.isOn(optAcceptLossy):
		d.warn(warnSchemaLossy)
	case status == schemaLossy:
		return d.fail(ExitSchema, msgSchemaLossy)
	case status == schemaOutage:
		return d.fail(ExitSchema, msgSchemaOutage)
	default:
		return d.fail(ExitSchema, msgSchemaFault, status)
	}
	return proceed
}
