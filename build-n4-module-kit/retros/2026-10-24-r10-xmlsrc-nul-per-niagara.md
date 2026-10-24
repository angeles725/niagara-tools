<!-- review-status: pending -->
<!-- module: kit -->
<!-- deltas: 1 -->
# 2026-10-24 · kit · r10-xmlsrc-nul-per-niagara

**Session**: niagara-research quality round 10, branch `fix/xmlsrc-nul-per-niagara` from origin/main 76f587d1
**Delta count**: 1

## What happened
A local station bog held `&#x0;` eight times in one String slot value. `internal/xmlsrc` refused U+0000 (XML 1.0,
encoding/xml), so every xmlsrc-based `.bog` tool (bog-audit, lint-bformat-call, lint-write-path `--bog`, and through
bog-audit commissioning-verify) failed the whole file with exit 3. The decision was taken from Niagara's own reader, not
from the XML spec: a `bajaObjectGraph` now reads U+0000 as Niagara does, with one warning; any other document still
refuses it.

## Evidence
- Niagara writes it: `XWriter.safe` escapes every char below 0x20 as `&#x<hex>;` (docSource nre
  `javax/baja/xml/XWriter.java:331-339`), so a String slot holding U+0000 is saved as `&#x0;`
- Niagara reads it: `XParser.toCharData` refuses only `<` and returns `(char)x` for any `&#...;`
  (`XParser.java:958-1030`); `read()` passes a literal NUL byte through (`:916-947`) into text, attribute values,
  comments and CDATA (`parseCDATA`, `:737-756`); only a name refuses it (`isName`, `:1265`). The station decoder is that
  parser (docSource baja `javax/baja/io/ValueDocDecoder.java:1145`, `BogDecoderPlugin`: `XParser.make(in)`)
- RED: 9/11 `TestNUL` subtests (`m.xml:2:22: illegal character code U+0000`), the bog-audit golden `truth-nul-char`
  (no `XML  WARN` row), commissioning-verify `truth-edge-bog-nul-char` (no `bog-xml` row), `TestXMLWarnings`
  `[ev: go/internal/xmlsrc/nul_test.go]`
- Real station bog (read-only): before, `bog-audit` exit 3 `:22787:119: illegal character code U+0000`; after, one
  `XML  WARN  <bog>:22787:78  U+0000 in 8 places` row and the checks run (exit 0); lint-bformat-call on a copy prints
  the one WARN row, exit 0
- Positions stay the written ones: U+0000 is masked by a 3-byte noncharacter (U+FDD0) while encoding/xml reads, and
  every offset is mapped back; a document that also holds U+FDD0 is refused rather than guessed

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | **A well-formedness rule the kit enforces is checked against Niagara's reader first.** Where XML 1.0 and `javax.baja.xml.XParser` disagree on a `.bog`, the station's reader is the truth: read what it reads, with one named WARN, never exit 3 for the whole file and never a silent pass. | `BUILD-LOOP.md` § 5 kit tooling doctrine (XML lints) | `[ev: go/internal/xmlsrc/nul.go]` |

## Lessons
- The failing input was a character reference (`&#x0;`), not a NUL byte: grep the bytes before choosing the fix, a
  byte count of `\0` finds nothing.
- A tolerance needs a home other than the error path: `Doc.Warnings` gives each consumer one line to print it.
