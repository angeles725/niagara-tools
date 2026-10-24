package reportmodule

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want options
		ok   bool
	}{
		{"root only", []string{"m"}, options{root: "m"}, true},
		{"every flag", []string{"--target-version", "4.14", "m", "--console-dir", "c", "--profile", "lan", "--legacy",
			"--wiring-map", "w", "--first-deploy", "--lint-only"},
			options{root: "m", targetVersion: "4.14", consoleDir: "c", profile: "lan", wiringMap: "w",
				legacy: true, firstDeploy: true, lintOnly: true}, true},
		{"-- ends the scan", []string{"m", "--", "--bogus", "n"}, options{root: "m"}, true},
		{"-- before the root", []string{"--", "m"}, options{}, false},
		{"no root", nil, options{}, false},
		{"missing value", []string{"m", "--target-version"}, options{root: "m"}, false},
		{"bad profile", []string{"m", "--profile", "kiosk"}, options{root: "m"}, false},
		{"empty profile", []string{"m", "--profile", ""}, options{root: "m"}, false},
		{"unknown flag", []string{"--bogus", "m"}, options{}, false},
		{"lone dash", []string{"-"}, options{}, false},
		{"second root", []string{"m", "n"}, options{root: "m"}, false},
		{"empty value is kept", []string{"--target-version", "", "m"}, options{root: "m"}, true},
		{"an empty root is replaced", []string{"", "m"}, options{root: "m"}, true},
	}
	for _, tt := range tests {
		got, ok := parseArgs(tt.args)
		if ok != tt.ok || (ok && !reflect.DeepEqual(got, tt.want)) {
			t.Errorf("%s: parseArgs(%q) = %+v, %v; want %+v, %v", tt.name, tt.args, got, ok, tt.want, tt.ok)
		}
	}
}
