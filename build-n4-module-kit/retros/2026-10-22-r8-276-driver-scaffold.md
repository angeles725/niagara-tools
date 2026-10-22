<!-- review-status: folded -->
<!-- module: kit -->
<!-- deltas: 2 -->
# 2026-10-22 · kit · r8-276-driver-scaffold

**Session**: niagara-research quality round 8, issue #276 (non-blocking findings of the T9e unit B review), branch `fix/r8-276-driver-scaffold` from origin/main 784ec45
**Delta count**: 2

## What happened
The T9e unit B review left six findings on `toolbelt/lint-driver-scaffold.sh` and the `doPing()` recipe. The lint
named a method by the leftmost `identifier(` on a depth-1 line and never cleared it, so
`@SuppressWarnings("x") public void doPing()` was read as a method called `SuppressWarnings`, and a device class
nested one level deeper was never entered: both wizard stubs passed. Its path went through `awk -v`, which turns
`\n` in a path into a newline. It kept a private comment stripper next to the shared `mb_strip`. The recipe said the
2000 ms connect timeout bounds the `Ping:` thread, but the `InetSocketAddress` constructor resolves DNS first.
This unit moved the lint onto `mb_strip`, named methods from the member header, passed the path via `ENVIRON`,
built LDS8 from the `wizard/` fixtures, documented the DNS caveat, and corrected the T9e retro's test count.

## Evidence
- RED on 784ec45: `bats tests/lint-driver-scaffold.bats` 8/12 ok; LDS6 (text block: false "not trusted" WARN),
  LDS7 (file ending in a text block named as unbalanced braces), LDS9 (3 of 4 annotated/nested stubs missed) and
  LDS10 (backslash path mangled) fail; GREEN 12/12 after, under gawk, mawk and gawk --posix `[ev: tests/lint-driver-scaffold.bats]`
- Mutations observed: dropping the literal blanking flips LDS6, keeping annotations flips LDS9, `awk -v` flips LDS10,
  dropping the END row flips LDS7 `[ev: toolbelt/lint-driver-scaffold.sh]`
- Same rows before and after on 2603 extracted docSource `.java` files and 802 client `.java` files `[ev: toolbelt/lint-driver-scaffold.sh]`
- `mb_parse` was not used for names: its Case B scan stops at an `@` line, so an annotated header with the `{` on
  the next line has no name and no method; its Case A needs the `{` on the signature line, so a `throws`
  continuation is unnamed too; and it reports the brace line, not the header line `[ev: toolbelt/lib/method-boundary.sh]`
- InetSocketAddress(String, int): "An attempt will be made to resolve the hostname into an InetAddress"
  (https://docs.oracle.com/javase/8/docs/api/java/net/InetSocketAddress.html); `Socket.connect(SocketAddress, int)`
  documents no behaviour for an unresolved endpoint in Java 8 or 17 `[ev: types/driver-authoring.md]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | A reachability ping states what its timeout bounds: `connect(addr, ms)` bounds the connect, not the DNS lookup in `new InetSocketAddress(host, port)`; a host-name device resolves on the `Ping:` thread, caches the `InetAddress`, and drops it on a host change or failed connect | `types/driver-authoring.md` § 5 | `[ev: types/driver-authoring.md]` |
| Δ2 | A lint names a method from its member header (the code since the last `;`, `{` or `}`, annotations dropped), at any class depth, not from the leftmost `identifier(` of a depth-1 line | `toolbelt/lint-driver-scaffold.sh` header; `types/driver-authoring.md` § 5 | `[ev: toolbelt/lint-driver-scaffold.sh]` |

## Lessons
- A "leftmost identifier on the line" rule is a fail-open name: an annotation with arguments or a nested class
  hides exactly the method the lint looks for.
- `awk -v` is not a path channel; `ENVIRON` passes bytes as is.
- A timeout bounds only the call it is passed to; name the blocking steps around it.

---
**Status**: FOLDED (same unit) — Δ1 `types/driver-authoring.md` § 5, Δ2 `toolbelt/lint-driver-scaffold.sh` header + `types/driver-authoring.md` § 5.
