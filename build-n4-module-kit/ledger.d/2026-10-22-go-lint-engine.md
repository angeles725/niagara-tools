<!-- ledger: kit -->
<!-- date: 2026-10-22 -->
<!-- last-change-note: quality round 8 T8k-2a: Go lint engine + lint-null-context-write port -->
log: 2026-10-22 · quality round 8 T8k-2a → modular Go lint engine (`go/internal/lint`: Rule, Finding/Row, Parse modes One/First/Each/Raw, fail-closed Walk, Run; `javasrc.File`; `cmd/n4lint`), `toolbelt/lint-null-context-write.sh` now a wrapper over rule package `rules/nullcontext`, byte-identical on fixtures + client trees except the issue #293(1) fix (null Context bounded to the get/resolve call); `types/logic-authoring.md` ORD form per case + `BFoxProxySession` release; closes #293. Retro retros/2026-10-22-go-lint-engine.md 2Δ pending. [ev: retro go-lint-engine]
open_issue: Go lint: go/** is not build-relevant in .githooks/pre-push (a Go change to a lint passes the gate without a BUILD-STATE line or retro) [ev: retro go-javasrc-scanner Δ1]
open_issue: Go lint: method-boundary awk quirks (a ')' inside a parameter list hides the method; `new X() {` field named as a method) — fix awk + javasrc together [ev: retro go-javasrc-scanner Δ2]
open_issue: Go lint: port lint-service-types (#286) onto internal/lint (lint-driver-scaffold #282 done) [ev: retro go-lint-engine Δ1]
