<!-- review-status: folded -->
# 2026-09-18 · kit · factory-composition-new-types-deltas

**Session**: corpus-mining for build-n4-module (operator: generate all possible kit deltas — factory/composition/new types).
**Delta count**: 8

## What happened

Mined B4 (type hierarchy/slot system), B734 (point-type taxonomy), B737 (engine thread + composition),
B738 (proxyExt/facets/propagateFlags/icon how-to), B955 (envCtrlDriver SDK example — driver
type-factory hooks, discovery, async write), and B778–B785 (module-authoring-exemplars: service/
ORD-scheme/subscription, child containers, palette/lexicon/@AgentOn, categories/relations/hierarchy,
query/search/index, templates, rdb dialect extension).

Duplicate scan confirmed the following are **already folded** (no re-proposal):

| Topic | Folded in |
|---|---|
| N4 extension idiom: subclass base + `<type>` + SPI object | `logic-authoring.md §Author-side SPIs` [ev: B778/B782/B785] |
| Custom SERVICE (`getServiceTypes`) + ORD SCHEME + server-side subscription | `logic-authoring.md §Author-side SPIs` [ev: B778] |
| Child-tree containers (frozen/dynamic/typed BFolder, no BComponentList, isChildLegal/isParentLegal) | `logic-authoring.md §Child-tree containers` [ev: B779] |
| Categories (no author side), relations (BRelation is concrete), hierarchy (BLevelDef compose) | `logic-authoring.md §Grouping and relating` [ev: B781] |
| Query/search/index (BQuery + BIAgent → BITable uniform pattern) | `logic-authoring.md §Query/search/index surface` [ev: B782] |
| Template = .ntpl artifact via BTemplateConfig + BConfigBinding, NOT a subclass SPI | `logic-authoring.md §Templates` [ev: B783] |
| Background jobs: BSimpleJob + run(Context) | `logic-authoring.md §Background jobs` [ev: B774] |
| Point extension: extends BPointExtension, onExecute | `logic-authoring.md §Authoring a point extension` [ev: B772] |
| Composition over flat slots: above ~12–15 slots group into child BComponents (L21) | `types/logic.md §Composition & organization` [ev: B737/B749/B750] |
| Internal BFrozenEnum vs cross-module double linking rule | `types/logic.md §Linking across custom modules` [ev: B828/L19] |
| DEFAULT_ON_CLONE flag for calc state | `types/logic.md §Tridium rt idioms §Flags` |
| rdb dialect extension (BRdbms, three abstract methods, RdbmsDialect) | `logic-authoring.md §Author-side SPIs` [ev: B785] |
| Analytics node (BOutputBlock, BBlockPin, no @AgentOn) | `logic-authoring.md §Author-side SPIs` [ev: B773] |
| @AgentOn dual-surface (Java annotation + module.xml) | `logic-authoring.md §Author-side SPIs` / `types/wb-widgets.md` [ev: B780] |

Eight net-new deltas remain from the vein.

## Evidence

- **B4 §4.2.3 — type hierarchy decision table**: the 3-question rule for a new type:
  (1) atomic+indivisible → `BSimple`; (2) compound+no actions/lifecycle → `BStruct`;
  (3) live component (actions, children, lifecycle) → `BComponent`. With concrete authoring
  checklists: BSimple needs `encode/decode/encodeToString/decodeFromString`, `equals+hashCode`,
  a `DEFAULT` constant, a private-ctor `make()` factory, and `getType()`. BStruct needs frozen
  Properties + no-arg ctor + `getType()`, NO dynamic slots and NO actions/topics. BFrozenEnum
  needs static `int ORD_NAME` constants, static instances, private ctor `super(ordinal)`,
  `make(int)` + `make(String)` factories. Explicit ordinals on `@Range` entries (not just
  auto-numbered) keep serialization stable when new values are inserted. None of this boilerplate
  is in the kit today. `[ev: corpus B4 §4.2.3 §4.2.7]`

- **B738 §738.4 — BIcon recipe**: METHODOLOGY.md §Icon currently says `src/rc/` and
  `BIcon.make(BOrd.make("module://<mod>/rc/icon16.png"))`. B738 shows the Tridium-canonical
  resource path is `icons/x16/` in the jar root (from `BIcon.java:69-71`):
  `BIcon.std(fileName)` → `"module://icons/x16/" + fileName`. SVG uses
  `BIcon.make(BOrd.make("module://<mod>/icons/myicon.svg"))`. Layered badge icons use
  `BIcon.make(BOrdList)`. The field MUST be `static final` (zero counter-examples in
  Tridium's first-party corpus). The current METHODOLOGY.md path is wrong. `[ev: corpus B738 §738.4]`

- **B738 §738.3 — propagateFlags author recipe**: the kit's LC4 kitControl note warns that
  `propagateFlags=0` is a hazard (a pre-existing CONSUMER warning). B738 §738.3 gives the
  AUTHOR side: to let operators control which input status bits propagate through a custom
  component, declare a `propagateFlags BStatus SUMMARY|OPERATOR` slot (default `BStatus.ok`)
  and mask the aggregated input status before writing the output. This is entirely absent from
  the kit's rt authoring recipes. `[ev: corpus B738 §738.3]`

- **B955 §955.1 — driver type-factory hooks**: a `BPointDeviceExt` subclass overrides three
  hooks so the framework builds the right driver subtree:
  `getDeviceType()→BMyDevice.TYPE`,
  `getProxyExtType()→BMyProxyExt.TYPE`,
  `getPointFolderType()→BMyPointFolder.TYPE`
  (`points/BEnvCtrlPointDeviceExt.java:45,53,63`). The trivial `BDeviceFolder`/`BPointFolder`
  subclasses exist solely to give the framework a concrete driver-specific `Type` to instantiate
  — an author copies them verbatim with an empty body + generated `TYPE`. Without these hooks,
  the framework cannot build the driver-specific subtree. Not in the kit's §Authoring a driver.
  `[ev: corpus B955 §955.1]`

- **B955 §955.2 — BStruct as discovery learn-entry / transient result bag**: discovery jobs
  extend `BSimpleJob`; each discovered entity populates a `BStruct` value-object
  (`BDeviceLearnEntry extends BStruct`, `BPointLearnEntry extends BStruct`) added to a
  `BFolder` flagged `HIDDEN|READONLY|TRANSIENT` on the job itself. The wb manager later reads
  the transient folder and instantiates the real component tree. Rule: "the learn-entry must
  carry enough to reconstruct the device representation when we add it into the running
  station." Not in the kit. `[ev: corpus B955 §955.2]`

- **B955 §955.5 — CoalesceQueue async write worker recipe**: the canonical write path in a
  proxy ext: `write(Context)` → posts an `ASYNC` `postWrite` action and returns `false`;
  `post(Action, BValue, Context)` is overridden to wrap every action into an `Invocation` and
  hand it to `getWriteHandler().postWork(...)`. The serializer is a `BWriteWorker extends
  BWorker` holding a `CoalesceQueue(1000)` + a background `Worker`. CoalesceQueue collapses
  duplicate-key pending writes that have not yet been sent — prevents write storms on rapid
  setpoint changes. Not in the kit's §Authoring a driver. `[ev: corpus B955 §955.5]`

- **B4 §4.3.3 — REMOVE_ON_CLONE flag for transient dynamic result sets**: dynamic slots
  carrying `REMOVE_ON_CLONE` are automatically discarded when `newCopy()` clones the component
  (export/import). Use it for runtime-populated result slots (search results, learn entries,
  transient state) that must NOT replicate to a copy. METHODOLOGY.md and `types/logic.md §Flags`
  cover TRANSIENT, READONLY, SUMMARY, OPERATOR, DEFAULT_ON_CLONE, ASYNC, FAN_IN —
  REMOVE_ON_CLONE is absent. `[ev: corpus B4 §4.3.3]`

- **B955 §955.6 — ProxyExt polls by parent-point value type**: in `poll()`, a `BProxyExt`
  inspects `getParentPoint().getOutStatusValue() instanceof BStatusNumeric|BStatusBoolean|
  BStatusString|BStatusEnum` to wrap the raw field response to the right Baja type, then calls
  `readOk(BStatusNumeric|...)` accordingly. The proxy ext must be value-type-agnostic; the
  containing control point owns the type. Never hardcode the response type in the proxy ext — the
  point's out-value class IS the contract. Not in the kit's §Authoring a driver.
  `[ev: corpus B955 §955.6]`

## Proposed kit deltas (propose-never-apply)

| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | `types/logic-authoring.md`: add new `§ New-type authoring checklists` — the 3-question type-decision rule (B4 §4.2.3 table: atomic → BSimple; compound+no lifecycle → BStruct; live component → BComponent) + authoring boilerplates for each: BSimple (encode/decode/encodeToString/decodeFromString, equals+hashCode, DEFAULT, private-ctor make(), TYPE); BStruct (extend BStruct, frozen Properties only — NO dynamic slots NO actions, public no-arg ctor, getType()); BFrozenEnum (static int ORD constants + static instances + private ctor super(ordinal) + make(int)/make(String)/make(String) + TYPE; use EXPLICIT ordinals on @Range entries for serialization stability when new values are inserted later) | `types/logic-authoring.md` (new section after §Minimal module) | `[ev: corpus B4 §4.2.3 §4.2.7]` |
| Δ2 | `METHODOLOGY.md §rt (components) §Icon`: correct and expand — resource dir is `src/rc/icons/x16/` (packaged to `icons/x16/` in the jar); `BIcon.std(fileName)` resolves to `"module://icons/x16/" + fileName` (NOT `src/rc/`); for SVG: `BIcon.make(BOrd.make("module://<mod>/icons/myicon.svg"))`; for layered badge icons: `BIcon.make(BOrdList)`; MUST be a `static final` field (Tridium zero counter-examples) — building a BIcon per call is wrong | `METHODOLOGY.md §rt (components) §Icon` | `[ev: corpus B738 §738.4]` |
| Δ3 | `types/logic.md §Composition & organization`: add "propagateFlags operator slot recipe" bullet — to give operators control over which input status bits propagate through a custom component, declare `@NiagaraProperty(name="propagateFlags", type="BStatus", defaultValue="BStatus.ok", flags=SUMMARY|OPERATOR)`; in execute/changed, mask the aggregated input status: `propagated = inputStatus.bitAnd(getPropagateFlags())`; write `out.setStatus(propagated)`. Gives operators the same status-propagation control kitControl blocks provide without a code change | `types/logic.md §Composition & organization` | `[ev: corpus B738 §738.3]` |
| Δ4 | `types/logic-authoring.md §Authoring a driver`: add "type-factory hook recipe" subsection — a `BPointDeviceExt` subclass overrides `getDeviceType()`, `getProxyExtType()`, `getPointFolderType()` to return concrete driver-specific `Type` objects; trivial `extends BDeviceFolder` / `extends BPointFolder` subclasses (empty body + generated TYPE) exist solely to provide a concrete driver-specific Type for the framework to instantiate; copy these verbatim — they are the load-bearing driver composition seam | `types/logic-authoring.md §Authoring a driver` | `[ev: corpus B955 §955.1]` |
| Δ5 | `types/logic-authoring.md §Authoring a driver`: add "discovery learn-entry recipe" subsection — discovery jobs `extend BSimpleJob`; each discovered entity is a `B<X>LearnEntry extends BStruct` (deviceName/deviceId or pointName/pointId/pointType fields) added to a `HIDDEN|READONLY|TRANSIENT BFolder` on the job; the wb manager reads this transient folder and builds the real component tree; the learn-entry must carry enough state to reconstruct the device/point representation at instantiation time | `types/logic-authoring.md §Authoring a driver` | `[ev: corpus B955 §955.2]` |
| Δ6 | `types/logic-authoring.md §Authoring a driver`: add "async write + CoalesceQueue recipe" subsection — `write(Context)` posts an `ASYNC` action `postWrite` and returns `false`; override `post(Action, BValue, Context)` to wrap as an `Invocation` and route to `getWriteHandler().postWork(...)`; the handler is `extends BWorker` holding a `CoalesceQueue(1000)` + background `Worker`; CoalesceQueue collapses duplicate-key pending writes — rapid setpoint changes coalesce instead of flooding the wire; this is the canonical "never block the calling thread" write pattern | `types/logic-authoring.md §Authoring a driver` | `[ev: corpus B955 §955.5]` |
| Δ7 | `types/logic.md §Tridium rt idioms to adopt §Flags`: add `REMOVE_ON_CLONE` flag to the flags bullet — dynamic slots carrying `REMOVE_ON_CLONE` are discarded by `newCopy()` on clone/export/import; use it for runtime-populated result slots (search results, learn entries, transient state bags) that must NOT replicate to a copy | `types/logic.md §Tridium rt idioms §Flags` | `[ev: corpus B4 §4.3.3]` |
| Δ8 | `types/logic-authoring.md §Authoring a driver`: add "value-type-agnostic proxy poll recipe" bullet — in `poll()`, inspect `getParentPoint().getOutStatusValue() instanceof BStatusNumeric|BStatusBoolean|BStatusString|BStatusEnum` to select the right `readOk(...)` overload; the proxy ext must be value-type-agnostic — the parent control point owns the output type; never hardcode a fixed type in the proxy ext | `types/logic-authoring.md §Authoring a driver` | `[ev: corpus B955 §955.6]` |

## Lessons

- B4's type-decision table and new-type boilerplates are foundational and early-corpus, yet the kit
  covers them only implicitly (the minimal-module scaffold shows a BComponent; BSimple/BStruct/
  BFrozenEnum boilerplates are absent). Δ1 fills a genuine first-principles gap.
- METHODOLOGY.md §Icon has a wrong resource path. `src/rc/` is the source tree layout; `icons/x16/`
  is the packaged jar path (what BIcon.std resolves against). They differ. Δ2 is a correctness fix,
  not an enhancement.
- propagateFlags is documented as a kitControl hazard (LC4) but the KIT'S author-side recipe for
  wiring it into a custom component is absent. Δ3 closes the author/consumer asymmetry.
- B955 is the only first-party, comment-bearing SDK example source in the corpus. Its three
  load-bearing driver patterns (type-factory hooks, BStruct learn-entries, CoalesceQueue write
  worker) are cleanly extractable and completely absent from §Authoring a driver. Δ4–Δ6 and Δ8
  fill that gap together.
- REMOVE_ON_CLONE (Δ7) is a natural companion to DEFAULT_ON_CLONE (already in the kit). The
  asymmetry was a minor oversight.

---
**Status**: PENDING — INDEX row appended: `| 2026-09-18-factory-composition-new-types-deltas.md | kit | 2026-09-18 | pending | 8 |`
