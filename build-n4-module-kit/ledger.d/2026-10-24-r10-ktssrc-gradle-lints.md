<!-- ledger: kit -->
<!-- date: 2026-10-24 -->
<!-- last-change-note: quality round 10: lint-structure L14/L15 read the gradle root scripts with the shared Kotlin scanner ktssrc -->
log: 2026-10-24 · quality round 10 → lint-structure L15 reads the root `build.gradle.kts` with `go/internal/ktssrc` (new token offsets + `Code` line view) instead of the Java scanner: nested comments, template strings and raw strings are read as Kotlin, a split `id(` call applies the plugin, a non-Kotlin script is the visible L15 WARN; L14 matches settings.gradle.kts on the same code lines (a pin in a comment or raw string is no pin, a non-Kotlin script is exit 3); byte-identical on 336 roots (fixtures + modulos_niagara_n4, Cliente included). Retro: retros/2026-10-24-r10-ktssrc-gradle-lints.md.
