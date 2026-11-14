package ngdeploy

// copyJars copies the built jars of the mode into the station modules directory with cp, one by one: it
// runs only after every precheck, verify and the gate passed. A cp that fails is exit 40 and stops there.
func (d *deploy) copyJars() int {
	for _, profile := range profilesOf[d.raw[optMode]] {
		if d.show("cp", d.builtJar(profile), d.station+"/") != 0 {
			return d.fail(ExitCopy, msgCopyFailed, d.jarName(profile))
		}
	}
	d.say(msgCopyOK)
	return proceed
}
