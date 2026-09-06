#!/usr/bin/env bash
# bog-audit.sh — Niagara station config.bog auditor (Campaign 8 PR10, B810/B816)
# [ev: retro campaign8-bog-audit]
#
# Audits a station config.bog against the deployed module source.  The bog
# is a zip containing file.xml; the script also accepts a bare .xml path.
#
# Usage:
#   bog-audit.sh <config.bog|file.xml> --module <MOD> [--module <MOD>]...
#                [--source-dir <DIR>] [--strict]
#
# Row format: <CHECK_ID>  PASS|FAIL|WARN|SKIP  <component-path>  <detail>
# Exits:      0 clean · 1 any FAIL · 3 parse error / prefix not found /
#             python3 absent / usage
#
# Checks that run from the bog alone (no --source-dir needed):
#   CHECK1  inventory              INFO  component count per own module
#   CHECK8  hoa-leftover           WARN  writable with active manual override
#   CHECK9  orphan-handle          FAIL  link sourceOrd not in any component
#   CHECK10 duplicate-handle       FAIL  same handle on multiple components
#   CHECK11 proxy-link-safety      FAIL  own-module output -> writable, no fallback  [ev: corpus B810]
#   CHECK12 dashboard-write-link   WARN  servlet-written slot is also a link target  [ev: corpus B816]
#
# Checks that need --source-dir (emits SKIP rows without it):
#   CHECK2  action-flag-drift      WARN  (--strict -> FAIL)
#   CHECK3  out-of-facet           WARN  value below source MIN facet
#   CHECK4  transient-persisted    WARN  TRANSIENT slot has value in bog
#   CHECK5  schema-drift-bog-extra FAIL  slot in bog not in source class
#   CHECK6  src-missing            WARN  slot in source not in bog
#   CHECK7  link-dangling          FAIL  link targetSlotName not in source class
#
# Dot-directories are excluded when walking --source-dir (D9b).
# python3 stdlib is the single deliberate exception to the awk-only rule (D10).
# command -v python3 guard exits 3 when python3 is absent (K20).
# This script is VCS-free by design (kit-links.bats L2).
#
# [ev: corpus B810] [ev: corpus B816]
set -u
LC_ALL=C; export LC_ALL

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

# python3 guard (D10 — single deliberate exception to the awk-only rule)
command -v python3 >/dev/null 2>&1 || {
  printf 'bog-audit: python3 not found (required for bog XML parsing)\n' >&2
  exit 3
}

usage() {
  printf 'usage: bog-audit.sh <config.bog|file.xml> --module <MOD>... [--source-dir <DIR>] [--strict]\n' >&2
  exit 3
}

# --- argument parsing ---
BOG=""
MODULES=()
SOURCE_DIR=""
STRICT=0

while [ $# -gt 0 ]; do
  case "$1" in
    --module)
      shift; [ $# -gt 0 ] || usage
      MODULES+=("$1") ;;
    --source-dir)
      shift; [ $# -gt 0 ] || usage
      SOURCE_DIR="$1" ;;
    --strict) STRICT=1 ;;
    --*) usage ;;
    *)
      if [ -z "$BOG" ]; then BOG="$1"
      else usage; fi ;;
  esac
  shift
done

[ -n "$BOG" ]               || usage
[ "${#MODULES[@]}" -gt 0 ]  || usage
[ -f "$BOG" ]               || { printf 'bog-audit: not a file: %s\n' "$BOG" >&2; exit 3; }

# --- extract file.xml from bog zip (or accept bare .xml) ---
case "$BOG" in
  *.xml) XML="$BOG" ;;
  *)
    XML="$_TMP/file.xml"
    unzip -p "$BOG" file.xml > "$XML" 2>/dev/null || {
      printf 'bog-audit: cannot extract file.xml from %s\n' "$BOG" >&2
      exit 3
    } ;;
esac
[ -s "$XML" ] || { printf 'bog-audit: empty file.xml in %s\n' "$BOG" >&2; exit 3; }

# --- python3 audit engine (embedded, stdlib only) ---
cat > "$_TMP/audit.py" <<'PYEOF'
"""bog-audit: Niagara config.bog auditor — Campaign 8 PR10, D10 grammar."""
import sys, os, re, time
from collections import defaultdict

START = time.time()

BOG_XML   = os.environ.get('BOG_XML', '')
MODULES   = set(os.environ.get('BOG_MODULES', '').split())
SRC_DIR   = os.environ.get('BOG_SRC', '')
STRICT    = os.environ.get('BOG_STRICT', '0') == '1'
FAILED    = False

def emit(check, status, path, detail):
    global FAILED
    print(f'{check}  {status}  {path}  {detail}', flush=True)
    if status == 'FAIL':
        FAILED = True

def ga(text, name):
    """Get attribute value: single-quoted first, then double-quoted."""
    m = re.search(rf"\b{re.escape(name)}='([^']*)'", text)
    if m: return m.group(1)
    m = re.search(rf'\b{re.escape(name)}="([^"]*)"', text)
    return m.group(1) if m else None

# ================================================================
# 1. Source-dir scanning
# ================================================================
src_slots   = defaultdict(dict)  # class_name -> {slot: {type, min, flags}}
src_actions = defaultdict(dict)  # class_name -> {action: flags_char}
src_extends = {}                 # class_name -> superclass_name (direct extends clause)
servlet_writes = set()           # slot names written by servlet .set("SLOT", ...)

if SRC_DIR:
    for root, dirs, files in os.walk(SRC_DIR):
        dirs[:] = sorted(d for d in dirs if not d.startswith('.'))  # D9b
        for fname in sorted(files):
            if not fname.endswith('.java'):
                continue
            cls = fname[:-5]  # strip .java -> class name
            try:
                content = open(os.path.join(root, fname),
                               encoding='utf-8', errors='replace').read()
            except Exception:
                continue

            # Capture extends clause: `class BFoo extends BWebServlet`
            ext_m = re.search(
                r'\bclass\s+' + re.escape(cls) + r'\s+extends\s+(\w+)', content)
            if ext_m:
                src_extends[cls] = ext_m.group(1)

            # CHECK12: servlet writes — .set("SLOT", ...)
            for m in re.finditer(r'\.set\(\s*"([^"]+)"\s*,', content):
                servlet_writes.add(m.group(1))

            # @NiagaraProperty / @NiagaraAction blocks (paren-balanced join)
            lines = content.split('\n')
            i = 0
            while i < len(lines):
                ln = lines[i]
                if '@NiagaraProperty' not in ln and '@NiagaraAction' not in ln:
                    i += 1
                    continue
                buf = ln
                depth = ln.count('(') - ln.count(')')
                j = i + 1
                while depth > 0 and j < len(lines):
                    buf += ' ' + lines[j]
                    depth += lines[j].count('(') - lines[j].count(')')
                    j += 1
                i = j

                if '@NiagaraProperty' in buf:
                    nm = re.search(r'name\s*=\s*"([^"]+)"', buf)
                    if not nm:
                        continue
                    slot_name = nm.group(1)
                    tm = re.search(r'type\s*=\s*"([^"]+)"', buf)
                    slot_type = tm.group(1) if tm else ''
                    fm = re.search(r'flags\s*=\s*(Flags\.[A-Za-z_|]+|[A-Za-z_][A-Za-z0-9_.|]*)',
                                   buf)
                    slot_flags = fm.group(1) if fm else ''
                    # MIN facet: BDouble.make(N) or BRelTime.makeSeconds(N)
                    min_val = None
                    mm = re.search(
                        r'BFacets\.MIN[^,)]*,\s*BDouble\.make\(\s*(-?[0-9.]+)[dDfFlL]?\s*\)', buf)
                    if mm:
                        try: min_val = float(mm.group(1))
                        except ValueError: pass
                    mm2 = re.search(
                        r'BFacets\.MIN[^,)]*,\s*BRelTime\.makeSeconds\(\s*([0-9]+)[lL]?\s*\)', buf)
                    if mm2 and min_val is None:
                        try: min_val = float(mm2.group(1)) * 1000
                        except ValueError: pass
                    src_slots[cls][slot_name] = {
                        'type': slot_type, 'min': min_val, 'flags': slot_flags
                    }

                elif '@NiagaraAction' in buf:
                    nm = re.search(r'name\s*=\s*"([^"]+)"', buf)
                    if not nm:
                        continue
                    action_name = nm.group(1)
                    fm = re.search(r'flags\s*=\s*Flags\.(\w+)', buf)
                    if fm:
                        flag_word = fm.group(1).upper()
                        flag_map = {
                            'HIDDEN': 'h', 'OPERATOR': 'o', 'READONLY': 'r',
                            'SUMMARY': 's', 'TRANSIENT': 't',
                        }
                        flag_char = flag_map.get(flag_word,
                                                 flag_word[0].lower() if flag_word else 'o')
                    else:
                        flag_char = 'o'  # default: operator-visible
                    src_actions[cls][action_name] = flag_char

# ================================================================
# 2. XML parsing — line-scanner, one-element-per-line assumption
# ================================================================
# TAG_RE: matches <tag attrs> <tag attrs/> </tag>
TAG_RE = re.compile(r'<(/?)([A-Za-z]\w*)\b([^>]*?)(/?)>')

class Comp:
    __slots__ = ('name','handle','type_','pfx','module','path',
                 'slots','actions','has_fallback','is_writable')
    def __init__(self, name, handle, type_, pfx, module, path):
        self.name = name
        self.handle = handle
        self.type_ = type_
        self.pfx = pfx
        self.module = module
        self.path = path
        self.slots = {}          # slot_name -> {type, value, flags}
        self.actions = {}        # action_name -> flags_str (from bog <a> element)
        self.has_fallback = False
        _type_simple = type_.split(':')[-1] if ':' in type_ else type_
        self.is_writable = bool(
            re.match(r'(Boolean|Numeric|Float|Integer)Writable$', _type_simple))

    def class_name(self):
        t = self.type_.split(':')[-1] if ':' in self.type_ else self.type_
        return 'B' + t

prefix_map = {}   # pfx -> module_name
handle_map = {}   # handle -> Comp
handle_count = {} # handle -> int  (CHECK10)
link_list = []    # list of {container_h, src_h, src_slot, tgt_slot}
stack = []        # [{type, handle, path, comp}]
in_link = False
link_buf = {}

def nearest_comp():
    for fr in reversed(stack):
        if fr['type'] == 'comp':
            return fr['comp']
    return None

try:
    with open(BOG_XML, encoding='utf-8', errors='replace') as fh:
        for raw in fh:
            line = raw.rstrip()
            if not line.strip():
                continue

            for m in TAG_RE.finditer(line):
                is_closing   = bool(m.group(1))
                tag_name     = m.group(2)
                is_self_cls  = bool(m.group(4)) or m.group(0).endswith('/>')
                full         = m.group(0)

                # ---- closing tag (only pop for <p> and <a>) ----
                if is_closing:
                    if tag_name in ('p', 'a') and stack:
                        popped = stack.pop()
                        if popped['type'] == 'link':
                            link_list.append(dict(link_buf))
                            in_link = False
                            link_buf = {}
                    continue

                # ---- action <a .../>: action with non-default flags ----
                if tag_name == 'a':
                    n  = ga(full, 'n') or ''
                    f_ = ga(full, 'f') or ''
                    comp = nearest_comp()
                    if comp:
                        comp.actions[n] = f_
                    continue

                # ---- <p ...> element ----
                if tag_name != 'p':
                    continue  # ignore bajaObjectGraph, xml declaration, etc.

                n      = ga(full, 'n') or ''
                h      = ga(full, 'h')   # None when absent
                t      = ga(full, 't') or ''
                v      = ga(full, 'v')
                m_attr = ga(full, 'm') or ''
                f_attr = ga(full, 'f') or ''

                # Register module prefixes  e.g. m='CRP=ColdRoomPan'
                if m_attr:
                    for part in m_attr.split():
                        if '=' in part:
                            pk, mv = part.split('=', 1)
                            prefix_map[pk] = mv

                # --- Inside a link: capture child data ---
                if in_link:
                    if is_self_cls:
                        if n == 'sourceOrd' and v and v.startswith('h:'):
                            link_buf['src_h'] = v[2:]
                        elif n == 'sourceSlotName' and v:
                            link_buf['src_slot'] = v
                        elif n == 'targetSlotName' and v:
                            link_buf['tgt_slot'] = v
                        # other link children (relationId etc.) are ignored
                    elif not is_self_cls and h is None:
                        # Non-self-closing inside link (unusual) - track as other
                        parent = stack[-1] if stack else {}
                        stack.append({'type': 'other', 'handle': None,
                                      'path': parent.get('path', ''), 'comp': None})
                    continue

                # --- Link wrapper: <p n='...' t='b:Link'> ---
                if t == 'b:Link' and not is_self_cls:
                    comp = nearest_comp()
                    in_link = True
                    link_buf = {
                        'container_h': comp.handle if comp else None,
                        'src_h': None, 'src_slot': None, 'tgt_slot': None,
                    }
                    parent = stack[-1] if stack else {}
                    stack.append({'type': 'link', 'handle': None,
                                  'path': parent.get('path', ''), 'comp': None})
                    continue

                # --- Component node: <p n='...' h='...' t='...'> (single-quoted h=) ---
                # The bog uses single quotes for components; h= is only on components.
                # Check: h is not None (= h attr was present in the tag, any quoting).
                if h is not None and not is_self_cls:
                    pfx    = t.split(':')[0] if ':' in t else ''
                    module = prefix_map.get(pfx, '')
                    parent = stack[-1] if stack else None
                    parent_path = parent['path'] if parent else ''
                    path = (parent_path + '/' + n).lstrip('/')

                    handle_count[h] = handle_count.get(h, 0) + 1
                    comp = Comp(n, h, t, pfx, module, path)
                    handle_map[h] = comp
                    stack.append({'type': 'comp', 'handle': h, 'path': path, 'comp': comp})
                    continue

                # --- Fallback child: marks nearest writable as having explicit fallback ---
                if n == 'fallback':
                    comp = nearest_comp()
                    if comp:
                        comp.has_fallback = True
                    if not is_self_cls:
                        parent = stack[-1] if stack else {}
                        stack.append({'type': 'other', 'handle': None,
                                      'path': parent.get('path', '') + '/fallback',
                                      'comp': None})
                    continue

                # --- Self-closing value property: <p n="slot" t="..." v="..." f="..."/> ---
                # Track when v= is present OR f= is present (for mode-type entries).
                # Only record direct children of the component; sub-slots of compound
                # properties (e.g. StatusNumeric.value) live inside an 'other' frame.
                # Also skip platform-managed slot names (wsAnnotation, status sub-slots).
                _PLATFORM_SLOTS = frozenset({'wsAnnotation', 'value', 'status', 'displayName'})
                if is_self_cls and (v is not None or f_attr):
                    # Only record if immediate parent is the comp itself (not a nested 'other')
                    _parent = stack[-1] if stack else None
                    if _parent and _parent['type'] == 'comp' and n not in _PLATFORM_SLOTS:
                        _parent['comp'].slots[n] = {'type': t, 'value': v, 'flags': f_attr}
                    elif _parent and _parent['type'] != 'comp':
                        pass  # sub-slot of compound property; ignore for CHECK5/6
                    elif _parent and _parent['type'] == 'comp' and n in _PLATFORM_SLOTS:
                        pass  # platform-managed; skip
                    continue

                # --- Non-self-closing composite property (in8, BackupRecord, etc.) ---
                if not is_self_cls and h is None:
                    parent = stack[-1] if stack else {}
                    ppath = parent.get('path', '')
                    stack.append({'type': 'other', 'handle': None,
                                  'path': (ppath + '/' + n).lstrip('/') if n else ppath,
                                  'comp': None})

except FileNotFoundError:
    print(f'bog-audit: cannot read {BOG_XML}', file=sys.stderr)
    sys.exit(3)
except Exception as exc:
    print(f'bog-audit: parse error: {exc}', file=sys.stderr)
    sys.exit(3)

# ================================================================
# 3. Run checks
# ================================================================
own_modules = MODULES
has_src = bool(SRC_DIR)

# ---- CHECK10: duplicate handles ----
for h, cnt in handle_count.items():
    if cnt > 1 and h in handle_map:
        c = handle_map[h]
        emit('CHECK10', 'FAIL', c.path,
             f'handle h:{h} appears {cnt} times (duplicate)')

# ---- CHECK9: orphan handle (link sourceOrd has no component) ----
for lk in link_list:
    src_h = lk.get('src_h')
    if src_h and src_h not in handle_map:
        emit('CHECK9', 'FAIL', f'h:{src_h}',
             'link sourceOrd references an unknown handle (orphan)')

# ---- CHECK1: inventory INFO (own-module component count) ----
module_counts = defaultdict(int)
for h, comp in handle_map.items():
    if comp.module in own_modules:
        module_counts[comp.module] += 1

if not module_counts:
    print(f'bog-audit: no components found for modules: {", ".join(sorted(own_modules))}',
          file=sys.stderr)
    sys.exit(3)

for mod in sorted(module_counts):
    emit('CHECK1', 'PASS', mod, f'components={module_counts[mod]}')

# ---- CHECK8: HOA leftover (writable with active manual override in in1..in8) ----
# Detects self-closing in[1-8] with a non-null/non-false value.
# Compound in8 elements (t='b:StatusBoolean') are not tracked here (inner
# sub-properties fall into the nearest comp's slot dict; that's conservative).
for h, comp in handle_map.items():
    if not comp.is_writable:
        continue
    for slot_n, slot_v in comp.slots.items():
        if not re.match(r'^in[1-8]$', slot_n):
            continue
        v_str = slot_v.get('value') or ''
        if v_str.lower() not in ('', 'false', '0', 'null'):
            emit('CHECK8', 'WARN', comp.path + '/' + slot_n,
                 f'HOA override active (value={v_str!r})')

# ---- CHECK11: proxy-link-safety (B810) ----
# Per writable: if any link has an own-module source AND the writable has no
# explicit fallback -> FAIL.  Emit exactly one row per writable.
links_by_container = defaultdict(list)
for lk in link_list:
    ch = lk.get('container_h')
    if ch:
        links_by_container[ch].append(lk)

for container_h in sorted(links_by_container):
    lks = links_by_container[container_h]
    container = handle_map.get(container_h)
    if not container or not container.is_writable:
        continue
    # Find the first link that has an own-module source
    own_lk = None
    for lk in lks:
        src_h = lk.get('src_h')
        if src_h:
            src_comp = handle_map.get(src_h)
            if src_comp and src_comp.module in own_modules:
                own_lk = lk
                break
    if own_lk is None:
        continue  # no own-module link in this writable
    if container.has_fallback:
        continue  # explicit fallback present — safe
    emit('CHECK11', 'FAIL', container.path,
         f'own-module h:{own_lk["src_h"]}.{own_lk["src_slot"]} '
         f'-> writable with no explicit fallback (relay holds last state on stop/reload)')

# ---- CHECK12: dashboard-write-to-LINK_TARGET (B816, advisory) ----
if has_src and servlet_writes:
    link_targets = {lk.get('tgt_slot') for lk in link_list}
    conflict = servlet_writes & link_targets
    for slot_name in sorted(conflict):
        for lk in link_list:
            if lk.get('tgt_slot') == slot_name:
                ch = lk.get('container_h')
                container = handle_map.get(ch)
                path = (container.path + '/' + slot_name) if container else slot_name
                emit('CHECK12', 'WARN', path,
                     'servlet-written slot is also a link target '
                     '(servlet write is ephemeral — link wins)')
                break

# ---- Source-dir-dependent checks (CHECK2-7) ----
if not has_src:
    for chk in ('CHECK2', 'CHECK3', 'CHECK4', 'CHECK5', 'CHECK6', 'CHECK7'):
        emit(chk, 'SKIP', '--source-dir', 'not provided; skipping source-dependent check')
else:
    for h, comp in sorted(handle_map.items(), key=lambda x: x[1].path):
        if comp.module not in own_modules:
            continue
        cls    = comp.class_name()
        s_slts = src_slots.get(cls, {})
        s_acts = src_actions.get(cls, {})

        # CHECK2: action flag drift
        for act_n, bog_f in comp.actions.items():
            if act_n not in s_acts:
                continue
            expected_f = s_acts[act_n]
            if bog_f != expected_f:
                status = 'FAIL' if STRICT else 'WARN'
                emit('CHECK2', status, comp.path + '/' + act_n,
                     f"flag-drift: bog f='{bog_f}' source expects f='{expected_f}'")

        # CHECK3: out-of-facet (value below source MIN)
        for slot_n, slot_v in comp.slots.items():
            if slot_n not in s_slts:
                continue
            min_val = s_slts[slot_n].get('min')
            if min_val is None:
                continue
            v_str = slot_v.get('value') or ''
            try:
                if float(v_str) < min_val:
                    emit('CHECK3', 'WARN', comp.path + '/' + slot_n,
                         f'value {v_str} below source MIN {min_val}')
            except (ValueError, TypeError):
                pass

        # CHECK4: transient-persisted
        for slot_n, s_info in s_slts.items():
            if 'TRANSIENT' not in s_info.get('flags', ''):
                continue
            if slot_n in comp.slots:
                v = comp.slots[slot_n].get('value') or ''
                if v.lower() not in ('', 'null', 'false', '0'):
                    emit('CHECK4', 'WARN', comp.path + '/' + slot_n,
                         'TRANSIENT slot has persisted value in bog')

        # CHECK5: bog-extra (slot in bog not in source) — skip READONLY platform slots
        # Superclass awareness: if the class extends a framework superclass (not declared
        # in our own source tree), a FROZEN bog slot (no t= attribute, i.e. type=='') may
        # be inherited from that superclass and is not statically decidable — emit WARN.
        # A DYNAMIC slot (t= present) or a slot on a BComponent-extending class is FAIL.
        _own_classes = set(src_slots.keys()) | set(src_actions.keys())
        _super = src_extends.get(cls, '')
        _extends_own = _super in _own_classes or _super in ('', 'BComponent', 'BAbstractService')
        for slot_n, slot_v in comp.slots.items():
            if slot_n in s_slts:
                continue
            bog_flags = slot_v.get('flags') or ''
            # Skip platform-managed READONLY slots (wsAnnotation f="r" etc.)
            if 'r' in bog_flags.lower():
                continue
            bog_type = slot_v.get('type') or ''
            # Frozen slot (no t= in bog) on a class extending a framework superclass:
            # the slot may be inherited — not statically decidable, emit WARN
            if not bog_type and not _extends_own and _super:
                emit('CHECK5', 'WARN', comp.path + '/' + slot_n,
                     f'frozen slot not in source {cls} — possibly inherited from {_super} '
                     f'(not statically decidable; add @NiagaraProperty override to suppress)')
            else:
                emit('CHECK5', 'FAIL', comp.path + '/' + slot_n,
                     f'slot in bog not in source {cls} (ghost/orphan)')

        # CHECK6: src-missing (slot in source not in bog)
        for slot_n in s_slts:
            if slot_n not in comp.slots:
                emit('CHECK6', 'WARN', comp.path + '/' + slot_n,
                     f'slot in source {cls} not in bog')

    # CHECK7: link-dangling (link targetSlotName absent from target component source)
    for lk in link_list:
        ch = lk.get('container_h')
        if not ch:
            continue
        container = handle_map.get(ch)
        if not container or container.module not in own_modules:
            continue
        tgt_slot = lk.get('tgt_slot')
        if not tgt_slot:
            continue
        cls    = container.class_name()
        s_slts = src_slots.get(cls, {})
        if tgt_slot not in s_slts:
            emit('CHECK7', 'FAIL', container.path,
                 f'link targetSlotName "{tgt_slot}" not in source {cls} (dangling)')

# ================================================================
# 4. Station-logic post-processing checks (CHECK13-CHECK19, D17)
# ================================================================
# These run over the handle-graph and link-graph built above.
# No new parser state — post-processing only. [ev: retro campaign8-station-logic]

# ---- CHECK13: relay-double-source FAIL ----
# Two distinct source handles drive the same target slot on the same WRITABLE PROXY container.
# Restricted to c:BooleanWritable / c:NumericWritable / c:EnumWritable — the CHECK11 target set.
# Alarm routing (a:AlarmClass routeAlarm fan-in) is NOT a relay double-source.
_WRITABLE_RE = re.compile(r'(Boolean|Numeric|Enum)Writable', re.IGNORECASE)
_relay_tgt_srcs = defaultdict(lambda: defaultdict(set))
for lk in link_list:
    ch  = lk.get('container_h')
    tgt = lk.get('tgt_slot')
    src = lk.get('src_h')
    if ch and tgt and src:
        _ct13 = handle_map.get(ch)
        if not _ct13 or not _WRITABLE_RE.search(_ct13.type_ or ''):
            continue  # only writable proxy points are relay targets
        _relay_tgt_srcs[ch][tgt].add(src)

for _ch in sorted(_relay_tgt_srcs):
    _container = handle_map.get(_ch)
    _cpath = _container.path if _container else f'h:{_ch}'
    for _tgt_slot, _src_set in sorted(_relay_tgt_srcs[_ch].items()):
        if len(_src_set) < 2:
            continue
        _srcs = ', '.join(f'h:{s}' for s in sorted(_src_set))
        emit('CHECK13', 'FAIL', _cpath,
             f'relay slot {_tgt_slot!r} driven by {len(_src_set)} distinct sources: {_srcs}')

# ---- CHECK14: own-output-unlinked WARN ----
# Own-module component has an output slot with no outgoing relay link.
# Restricted to true OUTPUT slots: name ends with 'Out' or matches 'condenser[0-9]+'.
# Config inputs (fanMode, valveMode, *Setpoint, *Limit, *Mode) are excluded by name pattern.
# Suppressed for defrost/resistance-specific outputs when hasDefrost is absent/false.
#
# Detection strategy: union of
#   (a) bog-stored OPERATOR slots matching _OUT_SLOT_RE, and
#   (b) type-inferred slots — *Out slots observed as link SOURCES on other instances of
#       the same own-module type within this station.  This catches TRANSIENT output slots
#       (e.g. evapOut on EvaporatorUnit) that are never persisted to the bog but are
#       linked on all compliant instances; a unit missing the link is the defect.
_DEFROST_SLOT_RE = re.compile(r'(defrost|resistance)', re.IGNORECASE)
_OUT_SLOT_RE     = re.compile(r'(Out$|condenser\d+$)', re.IGNORECASE)
_used_outputs    = {(lk['src_h'], lk['src_slot'])
                    for lk in link_list
                    if lk.get('src_h') and lk.get('src_slot')}

# Build per-type registry of *Out slots seen as link sources in this station
_type_out_slots = defaultdict(set)  # comp.type_ -> set of slot names
for _lk14 in link_list:
    _sh14 = _lk14.get('src_h'); _ss14 = _lk14.get('src_slot') or ''
    if not _sh14 or not _OUT_SLOT_RE.search(_ss14): continue
    _sc14 = handle_map.get(_sh14)
    if _sc14 and _sc14.module in own_modules:
        _type_out_slots[_sc14.type_].add(_ss14)

for _h, _comp in sorted(handle_map.items(), key=lambda x: x[1].path):
    if _comp.module not in own_modules:
        continue
    _hd_val = (_comp.slots.get('hasDefrost') or {}).get('value', '')
    _hd_off  = _hd_val.lower() in ('false', '0', '')
    # Candidate output slots: bog-stored OPERATOR outputs + type-inferred *Out slots
    _cand_slots = set()
    for _sn, _si in _comp.slots.items():
        if 'o' in (_si.get('flags') or '') and _OUT_SLOT_RE.search(_sn):
            _cand_slots.add(_sn)
    _cand_slots |= _type_out_slots.get(_comp.type_, set())
    for _sn in sorted(_cand_slots):
        if (_h, _sn) in _used_outputs:
            continue
        if _DEFROST_SLOT_RE.search(_sn) and _hd_off:
            continue   # suppress defrost/resistance output when hasDefrost is off/absent
        emit('CHECK14', 'WARN', _comp.path,
             f'output slot {_sn!r} has no outgoing relay link (own-output-unlinked)')

# ---- CHECK15: sensor-crossed-by-name WARN ----
# A link whose sourceSlotName contains C{n} (cold-room label) originates from a component
# whose name carries a different numeric suffix (E-unit index mismatch).
_C_NUM_RE    = re.compile(r'C(\d+)', re.IGNORECASE)
_COMP_IDX_RE = re.compile(r'[_-](\d+)$')

for lk in link_list:
    _src_h   = lk.get('src_h')
    _src_slt = lk.get('src_slot') or ''
    if not _src_h or not _src_slt:
        continue
    _sc = handle_map.get(_src_h)
    if not _sc or _sc.module not in own_modules:
        continue
    _cm = _C_NUM_RE.search(_src_slt)
    if not _cm:
        continue
    _c_n = int(_cm.group(1))
    _em = _COMP_IDX_RE.search(_sc.name)
    if not _em:
        continue
    _e_n = int(_em.group(1))
    if _c_n != _e_n:
        _ch2 = lk.get('container_h')
        _ct2 = handle_map.get(_ch2)
        _cp2 = _ct2.path if _ct2 else f'h:{_ch2}'
        emit('CHECK15', 'WARN', _cp2,
             f'slot {_src_slt!r} (C-room {_c_n}) sourced from {_sc.name!r} (unit index {_e_n}) — label mismatch')

# ---- CHECK16: hasDefrost <-> DefrostController sibling FAIL (BOTH directions) ----
# Forward: own-module component with hasDefrost=true must have a DefrostController sibling.
# Reverse: own-module DefrostController must have a hasDefrost=true unit sibling.
_par_children = defaultdict(list)
for _h, _comp in handle_map.items():
    _pparts = _comp.path.rsplit('/', 1)
    _ppath  = _pparts[0] if len(_pparts) > 1 else ''
    _par_children[_ppath].append(_comp)

for _ppath, _children in _par_children.items():
    _hd_comps = [
        c for c in _children
        if c.module in own_modules
        and (c.slots.get('hasDefrost') or {}).get('value', '').lower() == 'true'
    ]
    _dc_comps = [
        c for c in _children
        if 'DefrostController' in (c.name or '') or 'DefrostController' in (c.type_ or '')
    ]
    _plabel = repr(_ppath) if _ppath else '(root)'
    # Forward: hasDefrost=true without DefrostController sibling
    for _comp in _hd_comps:
        if not _dc_comps:
            emit('CHECK16', 'FAIL', _comp.path,
                 f'hasDefrost=true but no DefrostController sibling under {_plabel}')
    # Reverse: DefrostController without hasDefrost=true unit sibling
    if _dc_comps and not _hd_comps:
        for _dc in _dc_comps:
            if _dc.module not in own_modules:
                continue
            emit('CHECK16', 'FAIL', _dc.path,
                 f'DefrostController without hasDefrost=true unit sibling under {_plabel}')

# ---- CHECK17: roomN-index-mismatch FAIL ----
# A ColdRoom component directly contains a link whose targetSlotName carries an evap
# tile index that differs from the ColdRoom's own numeric suffix.
_COLDROOM_TYPE_RE = re.compile(r':ColdRoom$', re.IGNORECASE)
_EVAP_NUM_RE      = re.compile(r'evap(\d+)', re.IGNORECASE)
_SUFFIX_NUM_RE    = re.compile(r'[_-](\d+)$')

for lk in link_list:
    _ch3 = lk.get('container_h')
    _ts3 = lk.get('tgt_slot') or ''
    if not _ch3 or not _ts3:
        continue
    _ct3 = handle_map.get(_ch3)
    if not _ct3 or _ct3.module not in own_modules:
        continue
    if not _COLDROOM_TYPE_RE.search(_ct3.type_ or ''):
        continue
    _rm  = _SUFFIX_NUM_RE.search(_ct3.name)
    if not _rm:
        continue
    _room_idx = int(_rm.group(1))
    _em3 = _EVAP_NUM_RE.search(_ts3)
    if not _em3:
        continue
    _evap_idx = int(_em3.group(1))
    if _room_idx != _evap_idx:
        emit('CHECK17', 'FAIL', _ct3.path,
             f'link {_ts3!r} carries evap-index {_evap_idx} but room suffix is {_room_idx}')

# ---- CHECK18: evaporator unit tile-number consistency FAIL ----
# For each own-module EvaporatorUnit with a numeric suffix N, every evap-tile reference
# appearing in its links (incoming: stored in the unit as container; outgoing: unit is source)
# must carry the same tile number.  Crossing EvaporatorUnit_1 ↔ _3 produces two distinct
# tile numbers in the same unit's link set — FAIL.
_EVAP_UNIT_TYPE_RE = re.compile(r':EvaporatorUnit\b', re.IGNORECASE)
_unit_tile_sets = defaultdict(set)  # unit_handle -> set of tile numbers observed in links

for lk in link_list:
    _ss18 = lk.get('src_slot') or ''
    _ts18 = lk.get('tgt_slot') or ''

    # Case A: unit is the CONTAINER (incoming link stored in unit, e.g. panel/ColdRoom → unit).
    # Tile number may appear in either src_slot (panel slot: 'evap3ValveMode') or
    # tgt_slot (unit slot: 'evap3Hoa') depending on where the encoding lives.
    _ch18 = lk.get('container_h')
    if _ch18:
        _cu18 = handle_map.get(_ch18)
        if (_cu18 and _cu18.module in own_modules
                and _EVAP_UNIT_TYPE_RE.search(_cu18.type_ or '')):
            for _slotname in (_ss18, _ts18):
                _em18 = _EVAP_NUM_RE.search(_slotname)
                if _em18:
                    _unit_tile_sets[_ch18].add(int(_em18.group(1)))

    # Case B: unit is the SOURCE (outgoing link, e.g. unit → panel state slot 'evap3FanState').
    # Tile number is in the target slot name.
    _src_h18 = lk.get('src_h')
    if _src_h18:
        _su18 = handle_map.get(_src_h18)
        if (_su18 and _su18.module in own_modules
                and _EVAP_UNIT_TYPE_RE.search(_su18.type_ or '')):
            _em18b = _EVAP_NUM_RE.search(_ts18)
            if _em18b:
                _unit_tile_sets[_src_h18].add(int(_em18b.group(1)))

for _uh18, _tiles18 in sorted(_unit_tile_sets.items()):
    if len(_tiles18) < 2:
        continue
    _uc18 = handle_map.get(_uh18)
    if not _uc18:
        continue
    _sm18 = _SUFFIX_NUM_RE.search(_uc18.name)
    if not _sm18:
        continue   # skip units with no numeric suffix
    _tiles_str = ', '.join(str(n) for n in sorted(_tiles18))
    emit('CHECK18', 'FAIL', _uc18.path,
         f'link tile numbers {_tiles_str} disagree across HOA/state/freeze links (tile-number mismatch)')

# ---- CHECK19: link-direction WARN ----
# Expected wiring: panel → control for CONFIG slots (setpoints, modes);
#                  control → panel for STATE slots (readbacks, outputs).
# WARN when: (a) a CONTROL source writes a config slot INTO a PANEL container, or
#            (b) a PANEL source writes a state slot INTO a CONTROL container.
# CONTROL types: ColdRoom, EvaporatorUnit, DefrostController, CompressorControl (-rt).
# PANEL types:   RoomPanel, DashboardService, or module == DashboardPan.
# Panel→control config (normal) and control→control (any) are NOT flagged.
_CTRL_TYPE_RE   = re.compile(r'(ColdRoom|EvaporatorUnit|DefrostController|CompressorControl)',
                              re.IGNORECASE)
_PANEL_TYPE_RE  = re.compile(r'(RoomPanel|DashboardService)', re.IGNORECASE)
_CONFIG_SLOT_RE = re.compile(r'(setpoint|mode|limit|protect|diff)', re.IGNORECASE)
_STATE_SLOT_RE  = re.compile(r'(State|Out|status)$', re.IGNORECASE)
# Note: 'Temp' intentionally omitted — config thresholds like terminateOnResistanceTemp
# also end in Temp and are valid panel→control setpoints, not state readbacks (R20.8).

def _is_ctrl(c):
    return bool(_CTRL_TYPE_RE.search(c.type_ or '') or _CTRL_TYPE_RE.search(c.name or ''))

def _is_panel(c):
    return (bool(_PANEL_TYPE_RE.search(c.type_ or ''))
            or 'dashboardpan' in (c.module or '').lower())

for lk in link_list:
    _src_h5 = lk.get('src_h')
    _ts5    = lk.get('tgt_slot') or ''
    if not _src_h5 or not _ts5:
        continue
    _sc5 = handle_map.get(_src_h5)
    if not _sc5 or _sc5.module not in own_modules:
        continue
    _ch5 = lk.get('container_h')
    _ct5 = handle_map.get(_ch5)
    if not _ct5:
        continue
    _cp5 = _ct5.path
    # (a) control writes config slot into panel container — backward
    if _is_ctrl(_sc5) and _is_panel(_ct5) and _CONFIG_SLOT_RE.search(_ts5):
        emit('CHECK19', 'WARN', _cp5,
             f'control {_sc5.path!r} writes config slot {_ts5!r} into panel container (reverse direction)')
    # (b) panel writes state slot into control container — backward
    elif _is_panel(_sc5) and _is_ctrl(_ct5) and _STATE_SLOT_RE.search(_ts5):
        emit('CHECK19', 'WARN', _cp5,
             f'panel {_sc5.path!r} writes state slot {_ts5!r} into control container (reverse direction)')

elapsed = time.time() - START
print(f'# bog-audit: parse time {elapsed:.3f}s', flush=True)
sys.exit(1 if FAILED else 0)
PYEOF

# --- invoke the audit engine ---
BOG_XML="$XML" \
BOG_MODULES="${MODULES[*]}" \
BOG_SRC="$SOURCE_DIR" \
BOG_STRICT="$STRICT" \
python3 "$_TMP/audit.py"
STATUS=$?

exit "$STATUS"
