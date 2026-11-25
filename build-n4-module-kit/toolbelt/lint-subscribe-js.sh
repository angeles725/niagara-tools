#!/usr/bin/env bash
# lint-subscribe-js.sh — WARNs a `new baja.Subscriber()` in module JavaScript that the file never unsubscribes (FAIL under --strict).
#
# The JavaScript twin of lint-subscribe-without-unsubscribe.sh. A BajaScript Subscriber holds every component it
# subscribes until `unsubscribeAll()` / `unsubscribe(comp)` releases it (an asynchronous network call); `detach()` only
# removes the event handlers, so a view that leaves the page with no unsubscribe keeps the subscription. Module .js
# files only (*.min.js, build/ and node_modules/ are skipped). Checks (WARN; each row names one):
#   new-no-teardown        a `new baja.Subscriber(` site in a file with no `.unsubscribeAll(` or `.unsubscribe(` call
#                          anywhere (a promise finally, a detach/close hook or a widget doDestroy all count); one row per site.
#   subscribe-no-teardown  a `.subscribe(` on a Subscriber-named receiver (sub, $sub, kidSubscriber, subA) in a file with no
#                          teardown call and no `new baja.Subscriber(` site; the first one only. `subject.subscribe(` and
#                          `store.subscribe(` (RxJS, Redux) are not Subscribers.
#   waiver-no-reason       FAIL: a `// lint-subscribe-js:` marker with nothing after the colon; the site is not waived.
# A file that loads `bajaux/mixin/subscriberMixIn` as a module id in code (a define/require dependency or an import; not a
# comment or another string) is paired too: the mixin's destroy detaches the widget's Subscriber and
# calls unsubscribeAll() on it (organized/bajaux/bajaux-ux/extracted/rc/mixin/subscriberMixIn.js:304-317) [CERT].
# Comments, strings, templates and regex literals are no code, and `function unsubscribeAll()` is no call (a call has a
# `.` before it). The pairing is per file, as the Java twin's is: one teardown call pairs every site of the file. Waive a
# site torn down elsewhere with `// lint-subscribe-js: <reason>` on its line or the comment-only line above (the reason
# is mandatory). Limit: only `new baja.Subscriber(` is a site, not a bare `new Subscriber(` imported from
# baja/comp/Subscriber. The reading is the Go package go/internal/lint/rules/subscribeleakjs (doc comment, goldens).
#
# Evidence ([CERT] unless marked; niagara-research, BS=organized/bajaScript/bajaScript-ux/extracted/rc, NH=niagara-help):
#   BS/baja/comp/Subscriber.js:30 constructor, :336 subscribe, :632 unsubscribe, :758 unsubscribeAll (every held component);
#   the detach doc at :186-205 removes handlers. NH/devguide-clean/jsdoc/bajaScript-ux/baja.Subscriber.txt:644 detach,
#   :915 unsubscribeAll; tutorial-gettingStarted.txt:538-541 (`.finally(... sub.unsubscribeAll(); sub.detach();)`) and
#   :566-568 [CERT-doc]; devguide-examples/examples/bajaScript/subscription.js:55-59 [CERT-doc];
#   devguide-clean/remoteProgramming.txt:92 (cleanup via unsubscribeAll()) [CERT-doc]. Widget teardown:
#   organized/bajaux/bajaux-ux/extracted/rc/Widget.js:905 destroy -> :978 doDestroy; the vendor pattern
#   NH/module-js/history-ux/rc/HistoryGroupManager.js:87 (`this.$sub.unsubscribeAll()` in doDestroy).
#   `niagara_help.py search Subscriber` finds only the Java class javax.baja.sys.Subscriber; the JS names above come
#   from the corpus sources. API survey 2026-11-24 transports row K6 (finding G76).
# Implementation: that rule package on the shared engine go/internal/lint (go/README.md). This wrapper is the stable
# entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this
# script execs `n4lint lint-subscribe-js <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-subscribe-js.sh [--strict] <src-root>
#   Rows: WARN  lint-subscribe-js  <file>:<line>  <check>: <detail> (types/dashboard.md, BajaScript section)   (FAIL under --strict)
#         FAIL  lint-subscribe-js  <file>:<line>  waiver-no-reason: lint-subscribe-js: needs a reason after the colon
#   Exits:  0  clean or WARN only · 1  any FAIL, or any WARN under --strict · 3  usage/env: no root, an unknown option,
#           a root that is not a directory, a root with no *.js (stdout gets `ERROR  lint-subscribe-js  <root>  no *.js
#           sources found ...`), a sub-directory the walk cannot enter or an unreadable file
# VCS-free by design (kit-links L2).
# Mutation: SJS-leak -- ignoring the new baja.Subscriber( site reports nothing for an un-torn-down Subscriber
# Mutation: SJS-paired -- reading only teardown calls above the site WARNs the doDestroy hook and the one-call file
# Mutation: SJS-mixin -- ignoring the mixin id WARNs the view whose widget destroy already unsubscribes
# Mutation: SJS-mixin-code -- matching the id as raw text pairs the block-comment, string, log and longer-id files
# Mutation: SJS-waiver-subscribe -- skipping the waiver check on the subscribe-no-teardown row WARNs the waived files
# Mutation: SJS-bare-paired -- returning at the teardown before the bare-marker scan hides the FAIL in a paired file
# Mutation: SJS-detach -- counting detach( as a teardown passes the Subscriber that is only detached
# Mutation: SJS-notcode -- matching raw text pairs a leak with a commented or quoted unsubscribe and flags a commented site
# Mutation: SJS-subscribe -- matching any receiver WARNs subject.subscribe( and store.subscribe(, or reports every subscribe
# Mutation: SJS-waiver -- ignoring the marker WARNs the waived sites of waived.js
# Mutation: SJS-waiver-bare -- accepting a bare marker waives the site without a stated reason
# Mutation: SJS-waiver-literals -- reading a .js line with the Java comment finder loses the marker after an apostrophe in a template
# Mutation: SJS-skip -- reading *.min.js, build/, node_modules/ or Java WARNs files the module does not own
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-subscribe-js: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-subscribe-js "$@"
