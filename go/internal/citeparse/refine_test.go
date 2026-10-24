package citeparse

import (
	"reflect"
	"strings"
	"testing"
)

// kindNote is the part of a Pointer a refinement case pins: its kind and its notes.
type kindNote struct {
	Kind  Kind
	Notes string
}

func TestCardRefinesTemplatesAndUnverifiableKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       kindNote
	}{
		{"retro placeholder", "[ev: retro <stem> Δn]", kindNote{Template, ""}},
		{"corpus placeholder", "[ev: corpus B<n>]", kindNote{Template, ""}},
		{"path placeholder", "[ev: retros/<slug>.md:3]", kindNote{Template, ""}},
		{"ellipsis alone", "[ev: …]", kindNote{Template, ""}},
		{"engram observation", "[ev: engram obs 11479]", kindNote{Unverifiable, "memory-tool observation"}},
		{"live observation", "[ev: live oBIX PANCCADIA JACE 2026-09-07]", kindNote{Unverifiable, "live observation"}},
		{"CERT-live observation", "[ev: CERT-live local build 2026-10-22]", kindNote{Unverifiable, "live observation"}},
		{"QA observation", "[ev: QA harness run]", kindNote{Unverifiable, "live observation"}},
		{"commit word", "[ev: commit 35f84ca]", kindNote{Unverifiable, "VCS or tracker reference"}},
		{"issue", "[ev: issue #259]", kindNote{Unverifiable, "VCS or tracker reference"}},
		{"pull request", "[ev: PR #40 86aa533]", kindNote{Unverifiable, "VCS or tracker reference"}},
		{"bare short SHA", "[ev: f12f0fe]", kindNote{Unverifiable, "VCS or tracker reference"}},
		{"campaign unit", "[ev: r8 S3 apply]", kindNote{Unverifiable, "VCS or tracker reference"}},
		{"self-marked", "[deferred: unverifiable — no 4.15.3.28 install on this host]", kindNote{Unverifiable, "self-marked unverifiable"}},
		{"command run", "[ev: dmesg 2026-10-22 07:59]", kindNote{Unverifiable, "command run"}},
		{"bats run", "[ev: bats tests/lint-write-path.bats WP-stale-action]", kindNote{Unverifiable, "command run"}},
		{"decision ID", "[ev: D1b]", kindNote{Unverifiable, "decision or plan ID"}},
		{"plan task ID", "[ev: T8j-1 harness]", kindNote{Unverifiable, "decision or plan ID"}},
		{"design decision", "[ev: design D1c]", kindNote{Unverifiable, "decision or plan ID"}},
		{"user directive", "[ev: user directive 2026-09-20]", kindNote{Unverifiable, "decision or plan ID"}},
		{"all-letter hex word stays unparsed", "[ev: defaced thing]", kindNote{Unparsed, "kind not covered"}},
		{"a glob stays unparsed", "[ev: tests/lint-*.bats]", kindNote{Unparsed, "glob, not one file"}},
		{"web URL", "[ev: https://docs.gradle.org/current/userguide/build_environment.html, read 2026-10-24]", kindNote{Unverifiable, "web page"}},
		{"plain http URL", "[ev: http://example.com/a.txt]", kindNote{Unverifiable, "web page"}},
		{"a real retro is not a template", "[ev: retro go-ci-steps Δ1]", kindNote{Retro, ""}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ps := Card(tt.text)
			if len(ps) != 1 {
				t.Fatalf("%d pointers, want 1: %+v", len(ps), ps)
			}
			if got := (kindNote{ps[0].Kind, strings.Join(ps[0].Notes, " ")}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
