#!/usr/bin/env bash
# schema-risk.sh <before-dir> <after-dir>
# B795 §795.4 slot-diff survival classifier for Niagara N4 modules.
# Classifies each slot change between two module snapshots and returns the worst-cell verdict,
# so a saved-data-breaking change is caught BEFORE deploy (a .bog binds by name, ungated).
#
# Snapshot dir layout: module-include.xml + <pkg-path>/*.java  (B799 fixture shape).
# Annotation blocks are joined until parens balance (handles single-line fixtures AND
# multi-line real-module annotations identically — D4 requirement).
#
# Documented limits (D4 L1-L3):
#   L1  Multiple simultaneous renames (>=2 removed + >=2 added) are not paired; they surface
#       as separate remove_slot_unknown + add_slot rows. Verdict unchanged (LOSSY-or-worse).
#   L2  remove_slot_complex (SAFE) and retype_complex (LOSSY) are unreachable by design:
#       every unresolved removal -> remove_slot_unknown (LOSSY), every unresolved retype ->
#       retype_unknown (OUTAGE). Never downgrade on uncertainty (B795 §795.2).
#   L3  Unparseable/unbalanced annotations and slot-kind swaps (property<->action at same name)
#       emit UNKNOWN -> OUTAGE. No silent skips.
#
# Exit: 0=SAFE  1=LOSSY  2=OUTAGE  3=usage  4=env (unreadable snapshot / missing tool)
# Exit-code justification: usage (3) and env (4) sit ABOVE the verdict domain (0-2) so that a
# bad invocation never masquerades as OUTAGE in a deployment script.  This deviates from the
# kit's usual 2=usage/3=env convention by design (D4a).
set -u

# --- B795 §795.4 — embeddable classifier table (verbatim; diff-guarded by SR-CSV in schema-risk.bats) ---
CSV_TABLE=$(cat <<'CSV'
change_kind,verdict,evidence,note
add_slot,SAFE,B754-754.6-r1,new-default
reorder_slot,SAFE,B754-754.6-r2,index-free-byName
change_flags,SAFE,B754-754.6-r3,semantics-shift
change_default,SAFE,B754-754.6-r3,adopt-new-default
change_facets,SAFE,B754-754.6-r3,display-only
remove_slot_complex,SAFE,B754-754.6-r6,shunt-to-dynamic
remove_slot_simple,LOSSY,B754-754.6-r7,value-dropped
remove_action_or_topic,LOSSY,B754-754.6-r8,warn-skip
remove_slot_unknown,LOSSY,B754-754.6-r6r7-failsafe,use-when-subtype-unknown
rename_slot,LOSSY,B754-754.6-r9,orphaned-not-migrated
retype_complex,LOSSY,B754-754.6-r10,reverts-to-default
retype_simple,OUTAGE,B754-754.6-r11-B739,unparseable-v-propagates
retype_unknown,OUTAGE,B754-754.6-r10r11-failsafe,use-when-subtype-unknown
add_enum_tag,SAFE,B754-754.6,tag-string-stored
remove_or_rename_enum_tag,OUTAGE,B754-754.6-r13,InvalidEnumException
renumber_enum_ordinals,SAFE,B754-754.6-r5,fox-sync-unsafe
add_ext,SAFE,INFER-add_slot,complex-child-default
remove_ext,LOSSY,INFER-remove_slot_complex,dynamic-shunt-or-type-removed
class_rename,OUTAGE,B754-754.6-r12r13-failsafe,child-ref-LOSSY-else-classloader-OUTAGE
package_move,OUTAGE,INFER-r12-B631,classloader-unverifiable
swap_slot_kind,OUTAGE,INFER-795.2-failsafe,name-reused-diff-slot-kind
UNKNOWN,OUTAGE,B795-795.2-failsafe,default-fail-safe
CSV
)

usage() { printf 'usage: schema-risk.sh <before-dir> <after-dir>\n' >&2; exit 3; }

[ "$#" -ge 2 ] || usage
BEFORE_DIR="$1"
AFTER_DIR="$2"

# Env guard: module-include.xml must be readable in both snapshot dirs (R5.3 / D4)
[ -r "$BEFORE_DIR/module-include.xml" ] \
  || { printf 'error: cannot read %s/module-include.xml\n' "$BEFORE_DIR" >&2; exit 4; }
[ -r "$AFTER_DIR/module-include.xml"  ] \
  || { printf 'error: cannot read %s/module-include.xml\n' "$AFTER_DIR"  >&2; exit 4; }

# --- helpers ---

# type_name <snapshot-dir>: extract the first <type name="..."> value from module-include.xml
type_name() {
  awk 'match($0, /name="[^"]*"/) {
    n = substr($0, RSTART + 6, RLENGTH - 7)
    if (n != "") { print n; exit }
  }' "$1/module-include.xml"
}

# parse_slots <snapshot-dir>: emit one "kind TAB name TAB type" line per slot in
# declaration order.  Handles paren-balanced multi-line annotations (D4 L3).
parse_slots() {
  local dir="$1"
  local -a java_files
  mapfile -t java_files < <(find "$dir" -name "*.java" | sort)
  [ "${#java_files[@]}" -eq 0 ] && return
  awk '
    function count_depth(s,    i, n, ch, d) {
      d = 0; n = length(s)
      for (i = 1; i <= n; i++) {
        ch = substr(s, i, 1)
        if      (ch == "(") d++
        else if (ch == ")") d--
      }
      return d
    }
    function extract_attr(buf, attr,    patstr, matchstr, q1, q2) {
      # Accept optional whitespace around = (real modules write: name = "x")
      patstr = attr "[[:space:]]*=[[:space:]]*\"[^\"]*\""
      if (match(buf, patstr)) {
        matchstr = substr(buf, RSTART, RLENGTH)
        q1 = index(matchstr, "\"")
        if (q1 == 0) return ""
        matchstr = substr(matchstr, q1 + 1)
        q2 = index(matchstr, "\"")
        if (q2 == 0) return ""
        return substr(matchstr, 1, q2 - 1)
      }
      return ""
    }
    /^[[:space:]]*@NiagaraProperty/ ||
    /^[[:space:]]*@NiagaraAction/   ||
    /^[[:space:]]*@NiagaraTopic/ {
      buf   = $0
      depth = count_depth($0)
      while (depth > 0) {
        if ((getline line) <= 0) break
        buf   = buf " " line
        depth += count_depth(line)
      }
      n = extract_attr(buf, "name")
      if (n == "") next
      if (buf ~ /^[[:space:]]*@NiagaraProperty/) {
        t = extract_attr(buf, "type")
        printf "property\t%s\t%s\n", n, t
      } else {
        printf "action\t%s\taction\n", n
      }
    }
  ' "${java_files[@]}"
}

# csv_lookup <change_kind>: print "verdict|evidence|note" from CSV_TABLE
csv_lookup() {
  printf '%s\n' "$CSV_TABLE" | awk -F, -v k="$1" '
    NR == 1 { next }
    $1 == k { print $2 "|" $3 "|" $4; found = 1; exit }
    END     { if (!found) print "OUTAGE|B795-795.2-failsafe|default-fail-safe" }
  '
}

# verdict_score <verdict>: print numeric severity (SAFE=0, LOSSY=1, OUTAGE=2)
verdict_score() {
  case "$1" in
    SAFE)   printf '0' ;;
    LOSSY)  printf '1' ;;
    OUTAGE) printf '2' ;;
    *)      printf '2' ;;
  esac
}

# --- main ---

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

TYPE_NAME=$(type_name "$BEFORE_DIR")
[ -z "$TYPE_NAME" ] && TYPE_NAME="Module"

BEFORE_SLOTS="$WORK/before.tsv"
AFTER_SLOTS="$WORK/after.tsv"
CHANGES="$WORK/changes.tsv"

parse_slots "$BEFORE_DIR" > "$BEFORE_SLOTS"
parse_slots "$AFTER_DIR"  > "$AFTER_SLOTS"

# Diff the two slot sets and emit change records.
# Record format: change_kind TAB slot_label TAB before_info TAB after_info
awk -F'\t' '
  BEGIN {
    # Pre-declare as arrays to prevent scalar-vs-array conflicts in gawk
    # when a category has zero entries (never assigned any key).
    delete added; delete removed; delete kind_swap; delete retyped; delete unchanged
  }
  NR == FNR {
    b_kind[$2] = $1; b_type[$2] = $3
    b_seq[$2]  = FNR; b_ord[FNR] = $2; b_cnt = FNR
    next
  }
  {
    a_kind[$2] = $1; a_type[$2] = $3
    a_seq[$2]  = FNR; a_ord[FNR] = $2; a_cnt = FNR
  }
  END {
    # Categorise each slot; track counts to avoid length() on possibly-scalar vars
    n_rem = 0; n_add = 0; n_swap = 0; n_retype = 0
    for (nm in a_kind) {
      if (!(nm in b_kind)) {
        added[nm] = 1; n_add++
      } else if (b_kind[nm] != a_kind[nm]) {
        kind_swap[nm] = 1; n_swap++
      } else if (b_type[nm] != a_type[nm]) {
        retyped[nm] = 1; n_retype++
      } else {
        unchanged[nm] = 1
      }
    }
    for (nm in b_kind) {
      if (!(nm in a_kind)) { removed[nm] = 1; n_rem++ }
    }

    # Rename heuristic (D4 L1): exactly 1 removed + 1 added with same kind and type
    r_from = ""; r_to = ""
    is_rename = 0
    if (n_rem == 1 && n_add == 1) {
      for (nm in removed) r_from = nm
      for (nm in added)   r_to   = nm
      if (b_kind[r_from] == a_kind[r_to] && b_type[r_from] == a_type[r_to]) {
        is_rename = 1
        delete removed[r_from]; delete added[r_to]
        n_rem = 0; n_add = 0
      }
    }

    # Reorder detection: same name set, no type/kind changes, different declaration order
    is_reorder = 0
    if (!is_rename && n_rem == 0 && n_add == 0 && n_swap == 0 && n_retype == 0) {
      for (i = 1; i <= b_cnt; i++) {
        if (b_ord[i] != a_ord[i]) { is_reorder = 1; break }
      }
    }

    # Emit change records
    if (is_reorder) {
      for (i = 1; i <= b_cnt; i++) {
        nm = b_ord[i]
        print "reorder_slot\t" nm "\t" b_type[nm] "\t" b_type[nm]
      }
    } else {
      if (is_rename) {
        print "rename_slot\t" r_from "->" r_to "\t" b_type[r_from] "\t" a_type[r_to]
      }
      for (nm in removed) {
        if      (b_kind[nm] == "action") ck = "remove_action_or_topic"
        else if (b_type[nm] !~ /:/)      ck = "remove_slot_simple"
        else                             ck = "remove_slot_unknown"
        print ck "\t" nm "\t" b_type[nm] "\t-"
      }
      for (nm in added) {
        print "add_slot\t" nm "\t-\t" a_type[nm]
      }
      for (nm in kind_swap) {
        print "UNKNOWN\t" nm "\t" b_kind[nm] "\t" a_kind[nm]
      }
      for (nm in retyped) {
        if (b_type[nm] !~ /:/ && a_type[nm] !~ /:/) ck = "retype_simple"
        else                                         ck = "retype_unknown"
        print ck "\t" nm "\t" b_type[nm] "\t" a_type[nm]
      }
    }
  }
' "$BEFORE_SLOTS" "$AFTER_SLOTS" > "$CHANGES"

# Process change records: look up verdict, format output row, track worst cell
worst=0
worst_verdict="SAFE"

while IFS=$'\t' read -r ck slot_lbl _; do
  row=$(csv_lookup "$ck")
  verdict="${row%%|*}"
  rest="${row#*|}"
  evidence="${rest%%|*}"
  note="${rest#*|}"

  printf '%s  %s  %s.%s: %s (%s)\n' \
    "$verdict" "$ck" "$TYPE_NAME" "$slot_lbl" "$evidence" "$note"

  score=$(verdict_score "$verdict")
  if [ "$score" -gt "$worst" ]; then
    worst="$score"
    worst_verdict="$verdict"
  fi
done < "$CHANGES"

printf 'verdict=%s\n' "$worst_verdict"
exit "$worst"
