<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 5 -->
# 2026-10-15 · kit · devguide415-lexicons-ux-i18n

**Session**: research-sdd focus `dev-capabilities-kit-gaps` (niagara-research B1304–B1315) mapped the official N4.15.6.20 Developer Guide (`docDeveloper`) + `javax.baja` source against this kit; this retro carries the lexicon / UI-i18n findings of corpus B1305. Complements (does not repeat) `2026-10-15-n4-course-lexicon-logging-testing.md` (**lex** Δ1–Δ5).
**Delta count**: 5

## What happened
The devguide `localization.txt` and the framework code resolve a slot's display name from the BARE slot name in the declaring type's module lexicon (`NSlot.getDefaultDisplayName` → `Lexicon.make(module, cx).get(this.name)`), and an enum's from the bare tag. The kit advises the opposite — prefix keys as `Type.slot` "to dodge the B759 collision" — and its own checkers (`slot-coverage.sh`, `generate-wiring-map.sh`) accept or prefer the dotted form, so a lexicon can pass the kit gate while Workbench shows `toFriendly` text. Corpus B759 §759.1 already said the dotted form is not used for slot resolution; the kit cites B759 for the reverse advice. The kit also never teaches the lookup chain, lexicon modules (`lexicons{}` translations), locale selection, `LocalizableRuntimeException`, the diagnostic lexicon, or the JS `lex!` plugin — and `web:LexiconRpc` is `unrestricted`, which makes `module.lexicon` content readable by any web caller.

## Evidence
- Slot key = bare slot name — `[ev: code organized/baja/baja/vineflower/com/tridium/sys/schema/NSlot.java:42]` `[ev: niagara-help 4.15.6.20 devguide-clean/localization.txt:106-108]` `[ev: corpus B1305 §1305.2 C1]` `[ev: corpus B759 §759.1]`
- Kit advice and checkers — `[ev: code types/dashboard.md:203]` `[ev: code toolbelt/slot-coverage.sh:23]` `[ev: code toolbelt/generate-wiring-map.sh:20]`
- Enum key = bare tag (kit says `BOpMode.cooling`) — `[ev: code organized/baja/baja/vineflower/com/tridium/sys/schema/EnumType.java:50]` `[ev: niagara-help 4.15.6.20 devguide-clean/localization.txt:135-136]` `[ev: code types/value-types.md:51-52]`
- Lookup chain lang → root → null; `getText` returns the key; lexicon modules; locale order — `[ev: code organized/docSource/docSource-doc/extracted/baja/javax/baja/util/Lexicon.java:313-334]` `[ev: niagara-help 4.15.6.20 devguide-clean/localization.txt:59-70,189-214]` `[ev: niagara-help 4.15.6.20 devguide-clean/build.txt:443-521]`
- `LocalizableRuntimeException`, diagnostics `niagara.lexicon.diagnostics*` — `[ev: corpus B1305 §1305.1 rows 12-13]`
- JS `lex!` + `BLexiconRpc` `@NiagaraRpc(permissions="unrestricted")` — `[ev: code organized/web/web-rt/decompiled/com/tridium/web/rpc/BLexiconRpc.java:44]` `[ev: niagara-help 4.15.6.20 devguide-clean/jsdoc/js-ux/lex_lex.js.txt:157-208]`
- Kit grep 2026-10-15 (non-retro): `lexicons {`, `!lexicon`, `Accept-Language`, `BNameMap`, `displayNames`, `LocalizableException`, `lexicon.diagnostics`, `lex!` → 0 hits

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Replace the `Type.slot` prefixing advice: slot display names resolve from the BARE slot name in the declaring type's module lexicon, enum tags from the bare tag; `Type.slot` keys are never read. For colliding slot names across types use a `displayNames` `BNameMap` or override `getDisplayName(Slot,Context)`; dynamic enums carry `lexicon=<module>` in their `BEnumRange` options. Fix `slot-coverage.sh` (WARN "dotted key ignored by the framework" when a slot is matched only through `X.slot`) and drop the dotted-first lookup in `generate-wiring-map.sh`. Lint: YES. Corrects lex Δ1's catalogue on the slot row. | `types/dashboard.md:203` + `types/value-types.md:51` + `toolbelt/slot-coverage.sh:23` + `toolbelt/generate-wiring-map.sh:20` | `[ev: corpus B1305 §1305.2 C1]` |
| Δ2 | Add the resolution + override chain under L4: `Lexicon.get` = locale file → root `module.lexicon` → null; `getText` returns the key; lexicon modules (`lexicons{}` gradle block, `<lexicons>` in the manifest) override other modules' text — the supported way to ship a translation; browser users resolve `User.language` → `Accept-Language` → VM default, Workbench uses the VM locale; always look up with the caller's Context. Lint: partial (check `<lexicons>` entries against resource files). | `types/structure.md` § lexicon (L4) | `[ev: corpus B1305 §1305.1 rows 3-5]` |
| Δ3 | Extend lex Δ2 with the exception idiom: throw `LocalizableRuntimeException("<module>","<key>", args)` and render it with `toString(cx)` — `getMessage()` was formatted with a null Context; `LexiconText.toLocalizable(args)` for deferred text. Lint: YES via lex Δ2's key-exists checker. | `types/observability.md` § Fault surfacing | `[ev: corpus B1305 §1305.1 rows 6,12]` |
| Δ4 | Add a triage note: to see which lexicon key a UI label came from, set `niagara.lexicon.diagnostics=true` and `.prefix` (tokens `lexiconModule`, `lexiconLang`, `lexiconKey`) in `system.properties`; a label equal to its key means the key is absent. Lint: no. | `types/issues-and-gotchas.md` (new entry) | `[ev: corpus B1305 §1305.1 row 13]` |
| Δ5 | Add a ux i18n rule: use `lex!<module>` + `lex.get(key, …args)`; JS substitutes `{n}` only (no MessageFormat `''`); `web:LexiconRpc` is `unrestricted` and returns the module's lexicon for the caller's language, so never put secrets or internal-only text in `module.lexicon`. Lint: YES — grep `lex!` targets exist; WARN on secret-like keys (`password`, `token`, `apikey`). | `types/frontend-standard.md` § i18n | `[ev: corpus B1305 §1305.3]` |

## Lessons
- A checker that accepts a key form the framework never reads is worse than no checker: it certifies a dead translation.
- When the kit cites a corpus block, the advice should be re-read against that block — here B759 already contradicted the kit text that cites it.
- The lexicon RPC is unauthenticated by design; lexicon content is public UI text, never configuration.
