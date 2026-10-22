<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-22 · kit · go-lint-bundled-jar-class-version

**Session**: niagara-research quality round 8, branch `r8/lint-bundled-jar-class-version-go` from origin/main 942a4f4
**Delta count**: 1

## What happened
`toolbelt/lint-bundled-jar-class-version.sh` forked `unzip -p | od | awk` for every sampled class entry (the T27
profile: 18-25 s on the Honeywell tree) and became `go/cmd/lint-bundled-jar-class-version`
(`go/internal/jarversion`, archive/zip) behind a thin `go/go-tool.sh` `go_tool_bin` wrapper. The header (Usage,
Exit, `[ev:]` tag, both `# Mutation:` lines) is kept byte for byte, so `toolbelt/INDEX.md` does not change. The
port keeps the bash rules: the `find -P` walk with `build/` and dot-directories pruned (symlinks neither listed
nor followed), C byte order, GNU find's C-locale error wording and quoting for an unreadable sub-directory, at
most 20 `.class` entries per jar outside `META-INF/versions/`, bytes 6-7 as the major version (a 7-byte entry
gives its 7th byte, a shorter one none; the magic is not checked), bzip2 entries read and encrypted ones not, the
first record of a duplicated name, and a jar that is not a readable zip counted clean. Capturing the old output
first exposed a silent pass: `unzip -p <jar> <entry>` takes the entry name as a wildcard, so `Foo[1].class` or
`back\slash.class` read nothing and passed (and a control character in a name, as listed by `unzip -Z1`, did not
match either); the port reads every sampled entry literally.

## Evidence
- Old-vs-new over 50 edge shapes (usage with 0/2 args, `--strict`/`--` taken as the root, empty, file, missing,
  symlinked and mode-000 roots, `.`/`./`/`..`/relative/trailing- and double-slash roots, no jars, Java 8/9/mixed,
  major 258, no classes, not-a-zip, truncated, empty and zero-byte zips, `build/`/dot pruning, `x.jar/` dir,
  21-class limit, multi-release, 0/6/7/8-byte classes, duplicates, wildcard/backslash/UTF-8/tab/space/`$` names,
  `[`/`*` in jar names, prepended data, zip64, bzip2, encrypted, unreadable jar, one and two unreadable
  sub-directories, `--x` and `r--` dirs, a quote in a locked dir name, symlinked jar/dir/broken link): 45/50
  byte-identical (stdout, stderr, exit); 15/15 real-tree runs byte-identical (the 13 Cliente roots incl.
  Honeywell with and without a trailing slash and relative, plus `~/.m2/repository` and
  `~/.gradle/caches/modules-2`, 517 jars, 48 FAIL rows) `[ev: r8 jar-version differential harness]`
- The 5 edge differences, all deliberate: the wildcard-name and tab-name entries are now FAIL rows (3 cases
  including the whole-tree run), and bash leaked `od: cannot skip past end of combined input` on stderr for every
  entry shorter than 7 bytes or never extracted (encrypted, unmatched name), which the port does not print (all 5)
- `go/internal/jarversion` table tests (RED: package missing, build failed → GREEN) cover the version read,
  sampling, row format, the walk, find's quoting and every exit path `[ev: go/internal/jarversion/*_test.go]`
- Mutations: BJCV2 (`> 52` broken) fails `TestAboveJava8`/`TestCheckJar`/`TestMain` and bats BJCV2;
  BJCV-finderr (walk error dropped) fails `TestListJarsUnreadable`/`TestMain` and bats BJCV-finderr
- Honeywell tree at load ~2.5: bash 8.1-13.1 s (3 runs), wrapper 77-86 ms (5 runs); `~/.m2/repository` (41 FAIL
  rows) bash 33-39 s, wrapper ~0.28 s `[ev: r8 jar-version timing]`

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A lint must never pass an external tool a data value it parses as a pattern; a name read from an archive or tree is matched literally.** Why: `unzip -p <jar> <entry>` globbed `Foo[1].class` to nothing and the jar passed. | `CONTRIBUTING.md` § 2.2 Go module | `[ev: retro go-lint-bundled-jar-class-version]` |

Δ1 is applied in this unit for lint-bundled-jar-class-version only; the retro stays pending until a fold review
decides the rule.

## Lessons
- Bash's fail-open paths are kept, not fixed, in a port unless named: an unreadable or corrupt `.jar`, and a
  symlinked root (`find -P` lists nothing), still pass clean; whether they should be exit 3 is a contract decision.
- An unredirected stderr of a helper in a pipeline (`od`) is noise the old output carries; a byte-identity harness
  separates it from a real behaviour change.
- GNU find's C-locale message is `find: '<path>': <strerror>` with backslash escapes and octal for non-printable
  bytes; `zsh` may alias `find` to `bfs`, so probe through `bash -c`.

---
**Status**: PENDING (Δ1 applied to lint-bundled-jar-class-version; fold review pending) — INDEX row generated from the header: `| 2026-10-22-go-lint-bundled-jar-class-version.md | kit | 2026-10-22 | pending | 1 |`
