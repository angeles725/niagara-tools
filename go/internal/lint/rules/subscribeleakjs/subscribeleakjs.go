// Package subscribeleakjs is the lint-subscribe-js rule, the JavaScript twin of lint-subscribe-without-unsubscribe
// (package subscribeleak): a module .js file that creates a `new baja.Subscriber()` and never calls
// `.unsubscribeAll(` or `.unsubscribe(` anywhere in the file leaks every component it subscribes. WARN rows (FAIL
// under --strict); *.min.js, build/ and node_modules/ are not read.
//
//	new-no-teardown        WARN  a `new baja.Subscriber(` site in a file with no `.unsubscribeAll(` or `.unsubscribe(`
//	                             call (one row per site). The call may sit in a promise finally, a detach/close hook
//	                             or a widget doDestroy; the pairing is per file, as the Java twin's is.
//	subscribe-no-teardown  WARN  a `.subscribe(` on a Subscriber-named receiver (sub, $sub, kidSubscriber, subA) in a
//	                             file with no teardown call and no `new baja.Subscriber(` site (a Subscriber made
//	                             elsewhere and handed in); the first one only, as a file with a site already has its row.
//	waiver-no-reason       FAIL  a `// lint-subscribe-js:` marker with nothing after the colon; the site is not waived.
//
// A file that loads bajaux/mixin/subscriberMixIn as a module id in code (mixin.go) is paired too: the mixin's destroy calls unsubscribeAll()
// (organized/bajaux/bajaux-ux/extracted/rc/mixin/subscriberMixIn.js:304-317). detach() alone does not pair a site: it removes event handlers only and the components stay subscribed
// (baja.Subscriber.txt:644-672 below). Reading: jssrc.Code, so a comment, a string, a template or a regex never
// matches, and `function unsubscribeAll()` is no call (a call has a `.` before it). Waiver:
// `// lint-subscribe-js: <reason>` on the site's line or the comment-only line above (a Subscriber torn down in
// another file, or handed to a manager that unsubscribes it).
//
// Evidence (niagara-research; BS=organized/bajaScript/bajaScript-ux/extracted/rc, NH=niagara-help):
//   - BS/baja/comp/Subscriber.js:30 constructor, :336 subscribe, :632 unsubscribe, :758 unsubscribeAll (releases every
//     component the subscriber holds, an asynchronous network call); the detach doc, :186-205, removes handlers.
//   - NH/devguide-clean/jsdoc/bajaScript-ux/baja.Subscriber.txt:644 detach, :915 unsubscribeAll;
//     tutorial-gettingStarted.txt:538-541 (`.finally(... sub.unsubscribeAll(); sub.detach();)`) and :566-568 ("detach
//     ... is best practice for avoiding memory leaks"; unsubscribeAll stops listening); devguide-examples/examples/
//     bajaScript/subscription.js:55-59 (the same pair); devguide-clean/remoteProgramming.txt:92 (cleanup via unsubscribeAll()).
//   - Widget teardown: organized/bajaux/bajaux-ux/extracted/rc/Widget.js:905 destroy -> :978 doDestroy; the vendor
//     pattern NH/module-js/history-ux/rc/HistoryGroupManager.js:87 (`this.$sub.unsubscribeAll()` in doDestroy).
//
// Limits: only `new baja.Subscriber(` is a site (a bare `new Subscriber(` imported from baja/comp/Subscriber is not);
// a regex literal that holds a quote can hide a waiver marker on its line; a teardown anywhere in the file pairs
// every site, so a file that tears down one of two Subscribers is clean.
package subscribeleakjs

import (
	"regexp"
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-subscribe-js"

// Spec registers the rule with the argument contract of toolbelt/lint-subscribe-js.sh. StrictFails prints the WARN
// rows as FAIL under --strict; OptionalInAll keeps n4lint --all quiet on a root with no .js (a -rt root).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.First,
		Usage:  "usage: lint-subscribe-js.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.js"}, Except: []string{"*.min.js"}, Prune: []string{"build", "node_modules"}},
		// Spelled out: rules_test.go reads a rule's empty-root row from Spec.EmptyError, else it expects "*.java".
		EmptyError:    lint.EmptyRow{Msg: "no *.js sources found (wrong path or empty source tree)"},
		StrictFails:   true,
		OptionalInAll: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	newSub   = regexp.MustCompile(`\bnew\s+baja\s*\.\s*Subscriber\s*\(`)
	teardown = regexp.MustCompile(`\.\s*unsubscribe(All)?\s*\(`)
	subCall  = regexp.MustCompile(`([A-Za-z_$][A-Za-z0-9_$]*)\s*\.\s*subscribe\s*\(`)
	// subName is a receiver named for a Subscriber; `subject`, `subscription` and `store` (RxJS, Redux) are not.
	subName = regexp.MustCompile(`^([$_]*(sub|subscriber)([A-Z0-9_$].*)?|.*(Sub|Subscriber))$`)
)

const tail = ": the subscriptions outlive the view; detach() only removes handlers, so call unsubscribeAll() in a finally " +
	"or a widget doDestroy (types/dashboard.md, BajaScript section)"

const (
	msgNew       = "new-no-teardown: new baja.Subscriber() is never unsubscribed in this file (no .unsubscribeAll( or .unsubscribe( call)" + tail
	msgSubscribe = "subscribe-no-teardown: .subscribe( on a Subscriber that this file never unsubscribes (no .unsubscribeAll( or .unsubscribe( call)" + tail
	msgBare      = "waiver-no-reason: " + marker + " needs a reason after the colon"
)

// Check reports the unwaived sites of a file with no teardown call, and every waiver marker with no reason.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for _, n := range bare(f.Raw) {
		out = append(out, lint.Finding{Severity: lint.Fail, Line: n, Msg: msgBare})
	}
	code := jssrc.Code(f.Raw)
	for _, ln := range code {
		if teardown.MatchString(ln) {
			return sort2(out), nil // one call pairs every site of the file, before or after it
		}
	}
	if loadsMixIn(f.Raw) {
		return sort2(out), nil
	}
	sites, first := 0, 0
	for i, ln := range code {
		switch {
		case newSub.MatchString(ln):
			sites++
			if !waived(f.Raw, i+1) {
				out = append(out, lint.Finding{Line: i + 1, Msg: msgNew})
			}
		case first == 0 && subscribesOnSubscriber(ln):
			first = i + 1
		}
	}
	if sites == 0 && first > 0 && !waived(f.Raw, first) {
		out = append(out, lint.Finding{Line: first, Msg: msgSubscribe})
	}
	return sort2(out), nil
}

// subscribesOnSubscriber: ln calls .subscribe( on a receiver named for a Subscriber.
func subscribesOnSubscriber(ln string) bool {
	for _, m := range subCall.FindAllStringSubmatch(ln, -1) {
		if subName.MatchString(m[1]) {
			return true
		}
	}
	return false
}

// sort2 orders the rows by line, the bare-marker rows among the site rows.
func sort2(out []lint.Finding) []lint.Finding {
	sort.SliceStable(out, func(a, b int) bool { return out[a].Line < out[b].Line })
	return out
}
