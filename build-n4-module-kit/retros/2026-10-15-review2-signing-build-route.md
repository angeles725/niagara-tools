<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 6 -->
# 2026-10-15 · kit · review2-signing-build-route

**Session**: N4 Developer Certification Course — Lab 10 build + Review Assignment 2 (`BPointSummaryService`) implemented, signed with the Lab 5 cert, built from WSL through Windows `gradlew.bat`, and verified live in station `Cristian_Angeles_Review`; plus the delta-count slip in the 2026-10-15 retro batch
**Delta count**: 6

## What happened
Lab 10 and Review Assignment 2 were built by the agent, not the user, by calling the course's Windows Gradle roots from WSL
(`cmd.exe /c "cd /d <win root> && gradlew.bat :<part>:slotomatic"` then `:<part>:jar`). Both builds were green and both
modules ran in a station on the first try. Review 2 needed the Lab 5 signing setup carried into a second Gradle root, and
the only way to prove the jar was signed by the intended cert (not the default self-signed one) was to compare the
certificate in `META-INF/CODESIGN.RSA` with the lab PEM by SHA-256 fingerprint. While locating that setup, the writer
agent printed the signing profile XML in full, which exposed the keystore and key passwords in its tool output. Separately,
the commit that backed up the day's 23 retros said "178 deltas"; the consolidation script counted 168, because the parent
had summed one group of retros by hand.

## Evidence
- Lab 10 build: `gradlew.bat :training-rt:slotomatic` then `:training-rt:jar` → BUILD SUCCESSFUL; jar copied to `C:\Niagara\Niagara-4.15.6.20\modules\training-rt.jar` (byte-identical to `build/libs`); job "Histories To Csv" Success live, 10 CSVs, regex filter → "Beginning export of 5 histories" `[ev: course Labs/Lab10 Jobs]`
- Review 2: only `api(":control-rt")` was needed beside `baja`. A `bql:select name, out from control:NumericWritable` ORD resolved against `Sys.getStation()` with the caller's Context, plus `BajaFileUtil.createFileToOverwrite(BOrd, Context)` → `BIFile.write(byte[])`, compiled with no `bql-rt`/`file-rt` declaration (contrast: Lab 10's `BHistoryToCsv`/`BHistoryDatabase` needed `history-rt` + `file-rt` + `bql-rt`) `[ev: course Review Assignments/Review Assignment 2.pdf]`
- Signing carried to a second root by copying two blocks from `N4_Labs/build.gradle.kts`: `signingServices { signingProfileFactory { allowDefaultProfile.set(false) } }` and `niagaraSigning { aliases.set(listOf("CodeSigning")); signingProfileFile.set(file("<…>/certificates/code_signing.xml")) }`; `--info` logged `Signing with CodeSigning` / `RestrictedSigningProfile@…code_signing.xml` `[ev: course Labs/Lab5 Code Signing]`
- Signer proof: `META-INF/CODESIGN.RSA` subject `CN=(CodeSigning)`, issuer Tridium University Root CA, SHA-256 fingerprint equal to `N4_Labs/certificates/code_signing.pem`; `META-INF/module.xml` deps `baja` + `control-rt`; class major 52 `[ev: course Review Assignment 2 build]`
- Secret exposure: the signing profile XML holds the keystore and key passwords in plain text; a `cat` of that file during setup discovery printed them into the agent transcript (no file or report repeated them) `[ev: course Labs/Lab5 Code Signing certificates/code_signing.xml]`
- Live result: `Files/Review.txt` lines `NumericWritable: 54,00 {ok} @ def` (decimal comma: Spanish locale, value from `BStatusNumeric.toString(cx)`) `[ev: course Review Assignment 2 station test]`
- Count slip: niagara-tools commit ffcddad message says 178 deltas; `kit-improvement-candidates-2026-10-15.check.sh` counts 168 (devguide retros = 119, hand sum said 129) `[ev: cab877b]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | Add a **signer-identity check** to the verify gate: `verify-module.sh --expect-signer <cert.pem>` extracts the certificate from `META-INF/*.RSA` (`openssl pkcs7 -inform DER -print_certs`), compares its SHA-256 fingerprint with the PEM, FAILs on mismatch and WARNs when the signer is the default self-signed `Niagara4Modules` dev cert. Today "signed" only proves that *a* signature exists. Lint-able: yes. Reinforces register C(bdps#Δ4) `allowDefaultProfile` WARN. | `build-verify.md` § Verify (the `# signed` row) + `toolbelt/verify-module.sh` | `[ev: course Review Assignment 2 build]` |
| Δ2 | **Signing-profile secrets discipline**: a Niagara signing profile XML (`signingProfileFile`) stores keystore/key passwords in clear text. Never `cat`/print it. Locate it by path and read only element names (e.g. `grep -o '<[a-zA-Z]*'`). Keep it outside every repo (add `*signing*.xml`, `*.p12`, `*.jks` to the client `.gitignore` template). Agent briefs that touch signing state "path only, never contents". Lint-able: yes (a `.gitignore`/tracked-file check for profile XML and keystores). | `types/security.md` § 3 · CSRF + secrets (new § 3.4 Signing profile files) | `[ev: course Labs/Lab5 Code Signing certificates/code_signing.xml]` |
| Δ3 | **Carrying signing to a new Gradle root**: document the two root blocks (`signingServices{signingProfileFactory{allowDefaultProfile.set(false)}}` + `niagaraSigning{aliases; signingProfileFile}`) as the portable unit, the `--info` lines that prove which profile signed, and that a part-level gradle file needs nothing. Lint-able: no (recipe). | `build-verify.md` § Signing per deploy target | `[ev: course Labs/Lab5 Code Signing]` |
| Δ4 | **Windows-native build route from WSL** (not only for `niagaraTest`): for a Gradle root on the Windows side (course labs, vendor roots, `C:\` clones), run `cmd.exe /c "cd /d <win root> && gradlew.bat :<part>:slotomatic"` then `:<part>:jar`. It uses the root's own `niagara_home` and plugin version and copies the jar into `<niagara_home>\modules`, so back up the installed jar first and close Workbench if it holds the module. Extends register C(tpb#Δ1), which covers only `niagaraTest`. Lint-able: no. | `build-verify.md` § Raw gradle (quick WSL iteration, no deploy) | `[ev: course Labs/Lab10 Jobs]` |
| Δ5 | **Worker threads keep the caller's Context**: when an action hands work to a dedicated thread/executor, capture the action's `Context` and pass it to every `BOrd.get/resolve` and `toString(cx)` inside the worker, so BQL and file access stay filtered by the invoking user (a `null` Context grants `BPermissions.all`) and value text follows the user's locale. Reinforces `types/security.md` § 2.4 (`post()` drops RBAC context) and register C(null-Context). Lint-able: partial (WARN `BOrd.make(...).get(` / `.resolve(` with a literal `null` Context inside a `Runnable`/lambda body). | `types/security.md` § 2.4 + `types/logic-authoring.md` (threading section) | `[ev: course Review Assignment 2 build]` |
| Δ6 | **Retro delta counts are computed, never summed by hand**: promote the register's no-loss check to `toolbelt/count-retro-deltas.sh <retro...>` (count `^\| *Δ?[0-9]+ *\|` rows per file, compare with each file's `**Delta count**` and its `retros/INDEX.md` row, print the total). Any commit message or summary that states a delta total quotes this script's output. Lint-able: yes (INDEX count column vs table rows). | `BUILD-LOOP.md` § 7. Retro + close + `toolbelt/` | `[ev: cab877b]` |

## Lessons
- "Signed" is not "signed by the right cert". Prove the signer by fingerprint, not by the presence of `META-INF/*.RSA`.
- A signing profile file is a secret store. Treat its path as data and its contents as never printable.
- The Windows Gradle root works from WSL through `cmd.exe`, but it writes into that root's `niagara_home\modules`. Back up before the first run.
- A worker thread inherits nothing from the action that started it. Carry the Context across explicitly, or the work runs with all permissions and the default locale.
- Totals that appear in commit messages or reports come from a script, never from a hand sum.
