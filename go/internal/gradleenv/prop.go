package gradleenv

import "strings"

// PropFlag is the option both tools forward to gradle as a project property: --gradle-prop k=v is -Pk=v
// (niagaraSigningAlias=..., niagaraSigningProfile=...).
const PropFlag = "--gradle-prop"

// Prop is the gradle argument of a --gradle-prop value: -Pk=v, and ok false unless v is k=v with a
// non-empty key (the value may be empty).
func Prop(v string) (arg string, ok bool) {
	if v == "" || v[0] == '=' || !strings.Contains(v[1:], "=") {
		return "", false
	}
	return "-P" + v, true
}
