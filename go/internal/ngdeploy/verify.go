package ngdeploy

import (
	"bytes"
	"io"
	"strconv"
	"strings"
)

// verify checks every built jar of the mode in build/libs, before anything is copied: its type count
// against EXPECTED_<RT|UX>_TYPES and, for the -ux jar when BUILD_ID is set, the cache-buster.
func (d *deploy) verify() int {
	for _, profile := range profilesOf[d.raw[optMode]] {
		jar := d.builtJar(profile)
		if code := d.verifyJar(jar, d.env(expectedTypes[profile])); code != proceed {
			return code
		}
		if profile == "ux" {
			if code := d.verifyCacheBuster(jar); code != proceed {
				return code
			}
		}
	}
	return proceed
}

// verifyJar counts the lines of the jar's META-INF/module.xml that declare a type and refuses a count that
// is not the expected one (exit 50). Kept from the bash tool, whose `unzip | grep -c` ran under set -e and
// pipefail: no such line at all ends the run with exit 1 and no message, and an unzip that fails after
// printing some ends it with unzip's own status.
func (d *deploy) verifyJar(jar, expected string) int {
	var xml bytes.Buffer
	status := d.run(&xml, d.s.Err, "unzip", "-p", jar, moduleXML)
	count := 0
	for _, line := range strings.SplitAfter(xml.String(), "\n") {
		if strings.Contains(line, typeEntry) {
			count++
		}
	}
	switch {
	case count == 0:
		return ExitFault
	case status != 0:
		return status
	case strconv.Itoa(count) != expected:
		return d.fail(ExitVerify, msgVerifyCount, expected, jar, count)
	}
	d.say(msgVerifyOK, basename(jar), count, expected)
	return proceed
}

// verifyCacheBuster refuses a -ux jar whose rc/index.html does not carry "?v=<BUILD_ID>" (exit 50); without
// a BUILD_ID there is nothing to check. An index.html that cannot be read does not carry it.
func (d *deploy) verifyCacheBuster(jar string) int {
	id := d.env("BUILD_ID")
	if id == "" {
		return proceed
	}
	var index bytes.Buffer
	d.run(&index, io.Discard, "unzip", "-p", jar, indexHTML)
	if !strings.Contains(index.String(), "?v="+id) {
		return d.fail(ExitVerify, msgCacheBuster, jar, id)
	}
	d.say(msgCacheBusterOK, id)
	return proceed
}
