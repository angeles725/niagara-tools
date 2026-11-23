<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-11-10 · kit · delivery-guards-and-ui-contrast

**Session**: client A dashboard module 2.0 + mcp-n4 history tools delivery, and a training-simulator UI fix done in the same session
**Delta count**: 6

## What happened
Three late surprises cost rework at delivery time instead of at write time. (1) The push guard (client denylist) refused the branch because an evidence tag named the client; the commit was already made, so an unpushed commit had to be rewritten. (2) The pre-push hook then required a `ledger.d/` fragment and a pending retro that nothing asked for earlier. (3) Two working repositories (the client module checkout and a side project) have no remote at all, so their commits are on one disk only and nothing warned. Separately, an operator reported grey text that could not be read: a light/white theme had re-mapped every legacy colour token except the status-bar background, so instruction text in `--wb-text-secondary` sat on a dark `#3a3e43` bar (contrast about 1.3:1); a Playwright WCAG check found it in 10 of 10 screens.

## Evidence
- client-guard refused 3 lines in `mcp-n4-kit/skill/SKILL.md` (rule #10) on the first push of `feat/mcpn4-set-slot-types`; fixed by rewriting the unpushed commit to "client A" `[ev: 67df67c7]`
- pre-push: "build-relevant kit files changed without a ledger.d/ fragment ... + a pending retro" on the second push; satisfied by the retro and ledger commits `[ev: 3d37c6d2]`
- the client module checkout and the simulator checkout both answer "no upstream configured" `[ev: 3d37c6d2]`
- WCAG contrast test RED 10/10 screens, tokens fixed, one screen GREEN, the rest owned by per-screen code `[ev: 3d37c6d2]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Run the client denylist guard at pre-COMMIT (staged diff) as well as pre-push, so a client name in a retro or evidence tag is refused before a commit exists | `build-n4-module-kit/BUILD-LOOP.md` § `0.b Preflight (before the first build)` | `[ev: 67df67c7]` |
| Δ2 | Preflight prints the delivery requirements of a kit change up front (changelog.d fragment, ledger.d fragment committed alone, pending retro) so they are written with the change, not discovered by the pre-push refusal | `build-n4-module-kit/BUILD-LOOP.md` § `0.b Preflight (before the first build)` | `[ev: 3d37c6d2]` |
| Δ3 | APPLIED 2026-11-17 (`go/internal/preflight/remote.go`, row remote-backup). Preflight WARNs when the module checkout has no remote or no upstream (commits exist on one disk only) and names the backup decision as the operator's | `build-n4-module-kit/BUILD-LOOP.md` § `0.a Orient from BUILD-STATE (before touching anything)` | `[ev: 3d37c6d2]` |
| Δ4 | A dashboard SPA ships a WCAG AA contrast check (Playwright: visible text vs the first opaque background, ancestor opacity blended, disabled controls exempt; 4.5:1, 3:1 large) next to its node tests | `build-n4-module-kit/types/frontend-standard.md` § `Rule set` | `[ev: 3d37c6d2]` |
| Δ5 | A theme change re-maps EVERY colour token the screens consume (list consumers with a grep before the change) and pairs each text token with the backgrounds it may sit on; a token left on the old theme is a contrast defect | `build-n4-module-kit/types/frontend-standard.md` § `Rule set` | `[ev: 3d37c6d2]` |
| Δ6 | Faded state (pending / done items) uses a muted colour that still meets 4.5:1, never `opacity` on text, which drops contrast below the floor | `build-n4-module-kit/types/frontend-standard.md` § `Rule set` | `[ev: 3d37c6d2]` |

## Lessons
- A guard that only runs at push time turns a one-word fix into a history rewrite; run the cheap guards at commit time.
- Delivery requirements that a hook enforces belong in the preflight output, so the change is born complete.
- Contrast is a property of a token PAIR (text on background), so a theme migration must be checked pairwise, mechanically.
