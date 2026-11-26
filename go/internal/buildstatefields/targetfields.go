package buildstatefields

import (
	"errors"
	"regexp"
)

// The optional envelope fields that record the Niagara version a module is built and deployed for
// (BUILD-STATE.src.md § How to read): niagara_target is the station's version, plugin_version the niagara-module
// gradle plugin version of that line. Both are 2 to 4 dotted numbers, or `unknown`, which records nothing.
const (
	TargetField = "niagara_target"
	PluginField = "plugin_version"
	// TargetPattern and PluginPattern are the shapes of a valid value, as the messages word them.
	TargetPattern = "2 to 4 dotted numbers such as 4.15 or 4.15.6.20, or unknown"
	PluginPattern = "2 to 4 dotted numbers such as 7.6.22, or unknown"
)

var dotted = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}$`)

// ValidVersion reports whether v is 2 to 4 dotted decimal numbers ("4.15", "4.15.6.20").
func ValidVersion(v string) bool { return dotted.MatchString(v) }

var targetRules = []fieldRule{
	{TargetField, TargetPattern, ValidVersion},
	{PluginField, PluginPattern, ValidVersion},
}

func targetEnvelopes(text string) []fieldEnv { return fieldEnvelopes(text, TargetField, PluginField) }

// CheckTargets returns the first error of the niagara_target and plugin_version fields of every envelope.
func CheckTargets(text string) string {
	envs := targetEnvelopes(text)
	for _, r := range targetRules {
		if msg := r.problem(envs); msg != "" {
			return msg
		}
	}
	return ""
}

// Target is what an envelope records about the Niagara version of a module: the station's version and the
// plugin version, each "" when the module has none (no envelope, no field, or `unknown`).
type Target struct{ Version, Plugin string }

// ModuleTarget is the Target the envelope naming module records. Only the envelopes of module count: the first
// error among them (the messages of fieldRule) is the error; another module's malformed envelope is the sweep's.
func ModuleTarget(text, module string) (Target, error) {
	var envs []fieldEnv
	for _, e := range targetEnvelopes(text) {
		if e.module == module {
			envs = append(envs, e)
		}
	}
	for _, r := range targetRules {
		if msg := r.problem(envs); msg != "" {
			return Target{}, errors.New(msg)
		}
	}
	if len(envs) == 0 {
		return Target{}, nil
	}
	pick := func(name string) string {
		if v := envs[0].values[name]; len(v) == 1 && v[0] != SignerUnknown {
			return v[0]
		}
		return ""
	}
	return Target{Version: pick(TargetField), Plugin: pick(PluginField)}, nil
}
