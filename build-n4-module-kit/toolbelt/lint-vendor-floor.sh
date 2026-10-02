#!/usr/bin/env bash
# lint-vendor-floor.sh — parses every vendored browser library under <rc-dir>/vendor/**/*.js at the
# Chromium 83 panel floor (acorn, ecmaVersion 2020, classic script) and scans its tokens for APIs
# above that floor. A library that does not parse at the floor breaks the whole page on the HMI
# panel; an API above the floor throws only where it runs (it may be feature-guarded, so WARN).
# Run on every library bump and record the verdict next to the pin in rc/vendor/THIRD-PARTY.md.
# [ev: retro dashboard-frontend-standard Δ16]
#
# Checks (row id `vendor-floor`):
#   syntax   FAIL  the file does not parse as an ES2020 classic script (e.g. `??=` `||=` `&&=`
#                  ES2021, private methods / static blocks / class fields ES2022, top-level await).
#                  An ES2020 file that parses only as an ES module is WARN (load it with
#                  type="module" or vendor the UMD/global build — the panel standard is classic scripts).
#   api      WARN  a call to an API above the floor: .replaceAll( Promise.any( WeakRef
#                  FinalizationRegistry (C84/85), .at( (C92), Object.hasOwn( (C93), .findLast(
#                  .findLastIndex( (C97), structuredClone( (C98), AbortSignal.timeout( (C103),
#                  Object.groupBy( (C117), .toSorted( .toReversed( .toSpliced( (C110). Tokens, not
#                  text: a name inside a comment or a string is never flagged. Tokenized as a classic
#                  (sloppy) script first, as a module only when that fails, so a sloppy-only token
#                  (legacy octal, `with`) does not end the scan early.
#   --strict promotes every WARN row to FAIL.
#
# Tools: node + acorn. Resolution: KIT_NODE (node binary) else `node` on PATH; KIT_ACORN (acorn
# package dir) else toolbelt/eslint/node_modules/acorn (`npm install --prefix toolbelt/eslint`)
# else node's own require resolution (NODE_PATH). A missing tool is never a pass: one SKIP row
# names it and the exit is 4. An rc dir with no vendor/ needs no tool and exits 0.
#
# Usage:  lint-vendor-floor.sh <rc-dir> [--strict]
#   Row:   FAIL|WARN  vendor-floor  <path under rc-dir>:<line>  syntax|api: <reason>
#          SKIP  vendor-floor  <rc-dir>  unavailable: <tool> (<hint>)
# Exit:   0 no FAIL (WARN-only is still 0) · 1 any FAIL · 3 usage/env · 4 tool unavailable (K20)
#
# Mutation: VF3 -- treat a missing node as a clean pass (exit 0) so VF3 no longer sees exit 4
# Mutation: VF5 -- parse at ecmaVersion latest so the ??= fixture parses and VF5 exits 0
# Mutation: VF6 -- drop the API table so the replaceAll/hasOwn/findLast rows vanish
# Mutation: VF9 -- tokenize as a module only so a sloppy script's API calls after a legacy octal vanish
set -u
LC_ALL=C
export LC_ALL

usage_exit() {
  printf 'usage: lint-vendor-floor.sh <rc-dir> [--strict]\n' >&2
  exit 3
}

RC=""
STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    -*) usage_exit ;;
    *) [ -z "$RC" ] || usage_exit; RC="$1"; shift ;;
  esac
done
[ -n "$RC" ] || usage_exit
[ -d "$RC" ] || { printf 'lint-vendor-floor: not a directory: %s\n' "$RC" >&2; exit 3; }

TOOLBELT="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"

if [ ! -d "$RC/vendor" ]; then
  printf 'lint-vendor-floor: no vendor/ under %s — nothing to check\n' "$RC" >&2
  exit 0
fi

FILES=()
while IFS= read -r _f; do
  FILES+=("$_f")
done < <(find "$RC/vendor" -type f \( -name '*.js' -o -name '*.mjs' -o -name '*.cjs' \) | sort)
if [ "${#FILES[@]}" -eq 0 ]; then
  printf 'lint-vendor-floor: no *.js under %s/vendor — nothing to check\n' "$RC" >&2
  exit 0
fi

NODE_BIN="${KIT_NODE:-}"
if [ -z "$NODE_BIN" ]; then
  NODE_BIN="$(command -v node 2>/dev/null || true)"
fi
if [ -z "$NODE_BIN" ] || [ ! -x "$NODE_BIN" ]; then
  printf 'SKIP  vendor-floor  %s  unavailable: node not found (install Node, or set KIT_NODE)\n' "$RC"
  exit 4
fi

ACORN_SPEC="${KIT_ACORN:-}"
if [ -z "$ACORN_SPEC" ] && [ -d "$TOOLBELT/eslint/node_modules/acorn" ]; then
  ACORN_SPEC="$TOOLBELT/eslint/node_modules/acorn"
fi
[ -n "$ACORN_SPEC" ] || ACORN_SPEC="acorn"

"$NODE_BIN" - "$RC" "$ACORN_SPEC" "$STRICT" "${FILES[@]}" <<'JS'
'use strict';
const fs = require('fs');
const path = require('path');
const [rc, acornSpec, strictArg, ...files] = process.argv.slice(2);
const strict = strictArg === '1';
process.on('uncaughtException', (e) => {
  console.error(`lint-vendor-floor: env fault: ${e.message}`);
  process.exit(3);
});

let acorn;
try {
  acorn = require(acornSpec);
} catch (e) {
  console.log(`SKIP  vendor-floor  ${rc}  unavailable: acorn not resolvable from '${acornSpec}' ` +
    '(npm install --prefix toolbelt/eslint, or set KIT_ACORN)');
  process.exit(4);
}

const FLOOR = 2020; // Chromium 83 = ES2020 syntax
// name -> Chrome version that first shipped it
const MEMBER = { replaceAll: 85, at: 92, findLast: 97, findLastIndex: 97, toSorted: 110,
  toReversed: 110, toSpliced: 110 };
const QUALIFIED = { 'Object.hasOwn': 93, 'AbortSignal.timeout': 103, 'Promise.any': 85,
  'Object.groupBy': 117 };
const GLOBAL_CALL = { structuredClone: 98 };
const GLOBAL_NEW = { WeakRef: 84, FinalizationRegistry: 84 };

let fails = 0;
function row(sev, file, line, reason) {
  const s = (sev === 'WARN' && strict) ? 'FAIL' : sev;
  if (s === 'FAIL') fails++;
  console.log(`${s}  vendor-floor  ${path.relative(rc, file)}:${line}  ${reason}`);
}

// The parse error of src at ecmaVersion/sourceType, or null when it parses.
function parseError(src, ecmaVersion, sourceType) {
  try {
    acorn.parse(src, { ecmaVersion, sourceType, allowHashBang: true, locations: true });
    return null;
  } catch (e) {
    return e;
  }
}

// Tokens at 'latest', as a classic (sloppy) script first — the floor standard for vendored libs —
// and as a module only when the script tokenizer stops (an ES-module-only library). Each pass keeps
// the tokens read before an error; the longer run wins. A file neither tokenizer finishes already
// carries a syntax row from the parse at the floor.
function tokenRun(src, sourceType) {
  const out = [];
  try {
    for (const t of acorn.tokenizer(src, { ecmaVersion: 'latest', locations: true,
      allowHashBang: true, sourceType })) out.push(t);
    return { out, done: true };
  } catch (e) {
    return { out, done: false };
  }
}

function tokens(src) {
  const script = tokenRun(src, 'script');
  if (script.done) return script.out;
  const mod = tokenRun(src, 'module');
  return (mod.done || mod.out.length > script.out.length) ? mod.out : script.out;
}

function tokText(t) {
  if (t.type.label === 'name') return t.value;
  if (t.type.keyword) return t.type.keyword;
  return t.type.label;
}

for (const file of files) {
  const src = fs.readFileSync(file, 'utf8');
  const err = parseError(src, FLOOR, 'script');
  if (err) {
    const moduleErr = parseError(src, FLOOR, 'module');
    const line = err.loc ? err.loc.line : 1;
    if (!moduleErr) {
      row('WARN', file, line, `syntax: parses only as an ES module (${err.message}) — load with ` +
        'type="module" or vendor the UMD/global build');
    } else {
      row('FAIL', file, line, `syntax: does not parse at ecmaVersion ${FLOOR} (Chromium 83 floor): ` +
        err.message.replace(/ \(\d+:\d+\)$/, ''));
    }
  }
  const tk = tokens(src);
  for (let i = 0; i < tk.length; i++) {
    const t = tk[i];
    if (t.type.label !== 'name') continue;
    const prev = i > 0 ? tokText(tk[i - 1]) : '';
    const next = i + 1 < tk.length ? tokText(tk[i + 1]) : '';
    const line = t.loc.start.line;
    // MEMBER and QUALIFIED share no method name, so a member call never doubles a qualified row.
    if (prev === '.' && next === '(' && Object.prototype.hasOwnProperty.call(MEMBER, t.value)) {
      row('WARN', file, line, `api: .${t.value}( needs Chrome ${MEMBER[t.value]} (floor 83) — ` +
        'feature-guarded? record it in THIRD-PARTY.md');
    }
    if (prev === '.' && next === '(' && i > 1) {
      const q = tokText(tk[i - 2]) + '.' + t.value;
      if (Object.prototype.hasOwnProperty.call(QUALIFIED, q)) {
        row('WARN', file, line, `api: ${q}( needs Chrome ${QUALIFIED[q]} (floor 83) — ` +
          'feature-guarded? record it in THIRD-PARTY.md');
      }
    }
    if (prev !== '.' && next === '(' && Object.prototype.hasOwnProperty.call(GLOBAL_CALL, t.value)) {
      row('WARN', file, line, `api: ${t.value}( needs Chrome ${GLOBAL_CALL[t.value]} (floor 83) — ` +
        'feature-guarded? record it in THIRD-PARTY.md');
    }
    if (prev === 'new' && Object.prototype.hasOwnProperty.call(GLOBAL_NEW, t.value)) {
      row('WARN', file, line, `api: new ${t.value} needs Chrome ${GLOBAL_NEW[t.value]} (floor 83) — ` +
        'feature-guarded? record it in THIRD-PARTY.md');
    }
  }
}
process.exit(fails > 0 ? 1 : 0);
JS
