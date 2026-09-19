# Value types — BFrozenEnum, BDynamicEnum, BSimple, BStruct, BFacets, BUnit

Authoring your own value types. Format per entry: what / how / gotcha.
Do NOT contradict the slots-flags theme in `types/logic.md`.

---

## 1 · BFrozenEnum — canonical modern pattern

The standard type for a module-private discrete selector (e.g. `BFanMode`, `BCompressorStage`).

```java
@NiagaraType
@NiagaraEnum(range = {
    @Range(value = "cooling", ordinal = 0),
    @Range(value = "heating", ordinal = 1),
    @Range(value = "off",     ordinal = 2)
})
public final class BOpMode extends BFrozenEnum {
    public static final int COOLING = 0;
    public static final int HEATING = 1;
    public static final int OFF     = 2;

    public static final BOpMode Cooling = new BOpMode(0);
    public static final BOpMode Heating = new BOpMode(1);
    public static final BOpMode Off     = new BOpMode(2);
    public static final BOpMode DEFAULT = Cooling;
    public static final Type TYPE = Sys.loadType(BOpMode.class);

    private BOpMode(int ordinal) { super(ordinal); }

    /** make(int) — do NOT use a switch; let the range resolve by ordinal */
    public static BOpMode make(int ordinal) {
        return (BOpMode) Cooling.getRange().get(ordinal, false);
    }
    /** make(String) — resolves by tag string from the lexicon key */
    public static BOpMode make(String tag) {
        return (BOpMode) Cooling.getRange().get(tag);
    }

    @Override public Type getType() { return TYPE; }
}
```

**Ordinals must be declared explicitly** (`ordinal = N` in `@Range`). Auto-ordinals change on any
insertion and corrupt persisted BOG values silently. [ev: code BEovOpModeEnum.java; corpus B4]

**`make(int)` uses `anyInstance.getRange().get(ordinal, false)`**, not a switch.
The `false` flag means "return null on unknown ordinal" rather than throwing — caller checks null. [ev: code BEovOpModeEnum.java]

Display names (Workbench property sheet, oBIX `<enum>` list) come from the **lexicon** (`module.lexicon`), keyed
by `BOpMode.cooling`, `BOpMode.heating`, `BOpMode.off`. The tag itself is the wire value; the lexicon
key produces the translated label. [ev: corpus B4 §4.2]

**Cross-module link rule (from `types/logic.md`):** a `BFrozenEnum` value linked ACROSS two custom modules forces
a compile-time module dependency that the build system does not resolve easily. Use a plain `double` for
cross-module links; `BFrozenEnum` stays for INTERNAL slots only. [ev: retro coldroompan-fan-mode-defrost L19]

---

## 2 · BDynamicEnum + BEnumRange — driver dropdowns

Use when ordinals and labels are discovered at runtime (e.g. BACnet object-type enums, protocol command sets).

```java
// Build a range at discovery time
BEnumRange range = BEnumRange.make(
    new int[]   { 0,    1,    2,    3 },
    new String[]{ "Off","Low","Med","High" }
);
// Auto-ordinal shortcut (ordinals 0,1,2,…)
BEnumRange rangeAuto = BEnumRange.make(new String[]{ "Off", "Low", "Med", "High" });

// Instantiate a value
BDynamicEnum val = BDynamicEnum.make(1, range);      // explicit range
BDynamicEnum val2 = BDynamicEnum.make(1);            // range from the slot's facets at runtime

// Attach as a facet so Workbench renders a dropdown
BFacets f = BFacets.makeEnum(range);
```

**BDynamicEnum.make(ordinal)** (no range arg) resolves the range from the slot's facets at read time —
suitable for a proxy-ext slot whose range is set as a facet on the parent point. [ev: code BBacnetActionCommand.java]

**At discovery**, set the facet on the proxy-ext point property:
```java
point.setFacets(propSlot, BFacets.makeEnum(discoveredRange));
```
Workbench then renders a dropdown automatically. [ev: code AaPhpAttributeConversion.java]

---

## 3 · Custom BSimple — immutable leaf value

Use for a value that serializes as a single atom: a compound identifier, a rate descriptor,
a config key. Examples: `BHistoryId` (device + name pair), `BSampleRate` (auto/cov/fixed+interval).

```java
@NiagaraType
public final class BMyValue extends BSimple {
    public static final BMyValue DEFAULT = make("default");
    public static final Type TYPE = Sys.loadType(BMyValue.class);

    private final String data;
    private int hashCode = -1;          // lazy cache

    private BMyValue(String data) { this.data = data; }

    public static BMyValue make(String data) {
        BMyValue v = new BMyValue(data);
        return (BMyValue) v.intern();   // identity dedup for high-cardinality types
    }

    // ── binary I/O ──────────────────────────────────────────────────────────
    @Override public void encode(DataOutput out) throws IOException {
        out.writeUTF(data);
    }
    @Override public BObject decode(DataInput in) throws IOException {
        return make(in.readUTF());
    }

    // ── text/BOG I/O (v="…" in the .bog file) ───────────────────────────────
    @Override public String encodeToString() throws IOException { return data; }
    @Override public BObject decodeFromString(String s) throws IOException { return make(s); }

    // ── mandatory identity ───────────────────────────────────────────────────
    @Override public boolean equals(Object o) {
        return o instanceof BMyValue && data.equals(((BMyValue) o).data);
    }
    @Override public int hashCode() {
        try {
            if (hashCode == -1) hashCode = encodeToString().hashCode();
            return hashCode;
        } catch (Exception e) { return System.identityHashCode(this); }
    }
}
```

**The four mandatory I/O methods** — missing any one of them leaves BOG or binary serialization broken
with no compile-time error. BOG stores the value as `v="encodeToString()"` and restores via
`decodeFromString`. [ev: code BSampleRate.java; BHistoryId.java]

**`BSimple` throws if `hashCode()` is not overridden** — the base class default throws an exception to
enforce override. Lazy `encodeToString().hashCode()` is the common pattern. [ev: code BSampleRate.java:86-96]

**`newCopy()` returns `this`** — BSimple instances are immutable; the inherited `newCopy()` is correct
as-is (returns `this`). Do NOT override it unless the type has mutable state (it shouldn't).

**`.intern()`** in `make()` deduplicates instances by value — essential for high-cardinality types (e.g.
`BHistoryId`) to avoid heap pressure when thousands of points each hold an id. [ev: code BHistoryId.java:52]

---

## 4 · Custom BStruct — composite frozen value

Use for a small fixed-slot bag of BValue fields: action parameter dialogs, discovery learn entries,
configuration structs. Examples: `BAlarmTimestamps` (4 slots), `BAaPhpLearnDeviceEntry` (2 slots).

```java
@NiagaraType
@NiagaraProperties({
    @NiagaraProperty(name = "setpoint",    type = "double",   defaultValue = "0.0",
                     facets = {@Facet(value = "BFacets.makeNumeric(BUnit.getUnit(\"celsius\"),1,-50,100)")}),
    @NiagaraProperty(name = "hysteresis",  type = "double",   defaultValue = "1.0"),
    @NiagaraProperty(name = "label",       type = "String",   defaultValue = "")
})
public class BMyConfig extends BStruct {
    public static final Property setpoint   = newProperty(0, 0.0, /* facets */ null);
    public static final Property hysteresis = newProperty(0, 1.0, null);
    public static final Property label      = newProperty(0, "", null);
    public static final Type TYPE = Sys.loadType(BMyConfig.class);

    public BMyConfig() {}   // REQUIRED public no-arg ctor

    public double getSetpoint()  { return getDouble(setpoint); }
    public void   setSetpoint(double v) { setDouble(setpoint, v, null); }
    // … getters/setters for other slots …

    @Override public Type getType() { return TYPE; }
}
```

**Slot machinery handles serialization** — do NOT write `encode`/`decode`; they are inherited
and driven by the declared slots. [ev: code BAlarmTimestamps.java; BExtensionName.java]

**Only frozen `@NiagaraProperty` slots** — NO `add()`, NO actions, NO topics, NO children.
If you need lifecycle or children, use `BComponent`. [ev: corpus B4 §4.3]

**BStruct vs BComponent:**

| | BStruct | BComponent |
|---|---|---|
| Lifecycle (`started`/`stopped`) | — | ✓ |
| Children | — | ✓ |
| Actions / Topics | — | ✓ |
| Serialization | slot machinery | slot machinery |
| Use as action param | ✓ | — |
| Use as discovery entry | ✓ | — |

A `BStruct` used as a `@NiagaraAction` parameter renders in Workbench as a multi-field argument dialog
automatically. [ev: corpus B4; code BAaPhpLearnDeviceEntry.java]

---

## 5 · BFacets — UI hints on slots

BFacets are UI hints. **The server does NOT validate min/max on writes** — clamp defensively in code.

```java
// Numeric: unit, decimal precision, display min, display max
BFacets f = BFacets.makeNumeric(BUnit.getUnit("celsius"), 1, -50.0, 100.0);

// Integer: unit (nullable), min, max, radix (10=decimal, 16=hex)
BFacets fi = BFacets.makeInt(null, 0, 255, 16);          // hex display
BFacets fd = BFacets.makeInt(BUnit.NULL, 0, 31, 10);      // decimal

// Enum dropdown
BFacets fe = BFacets.makeEnum(range);

// Boolean display text
BFacets fb = BFacets.makeBoolean("Running", "Stopped");

// Arbitrary key-value
BFacets fa = BFacets.make("fieldEditor", "mymodule:MyFE");

// Merge two facets (b wins on collision)
BFacets fm = BFacets.make(existingFacets, newFacets);

// Remove a key
BFacets fr = BFacets.makeRemove(existingFacets, BFacets.UNITS);

// Sentinel — "use slot's own facets" / "no override"
BFacets.DEFAULT
```

**Standard keys** (constants on `BFacets`): `UNITS`, `PRECISION`, `MIN`, `MAX`, `RANGE`,
`FIELD_EDITOR`, `trueText`, `falseText`. [ev: code BAaPhpLearnDeviceEntry.java; BAaPhpDevice.java; AaPhpAttributeConversion.java]

**Projection pattern** — declare facets once on a `facets` config slot and project onto outputs:
```java
// in @NiagaraProperty annotation:
facets = { @Facet(value = "BFacets.makeNumeric(BUnit.getUnit(\"celsius\"), 1, null, null)") }
// in execute(): propagate to the output slot at runtime via getSlotFacets()
```
[ev: `types/logic.md` §Tridium rt idioms]

---

## 6 · BUnit — physical units

```java
BUnit celsius     = BUnit.getUnit("celsius");     // from install units.xml
BUnit dimensionless = BUnit.NULL;                 // no unit

// Convert (handles offset units like °C→°F correctly)
double fahren = celsius.convertTo(BUnit.getUnit("fahrenheit"), 22.0);

// Custom unit (rare — prefer units.xml entries)
BUnit custom = BUnit.make("custom_unit_name", "abbrev", dimension, …);
```

Units come from `units.xml` bundled with the Niagara installation. Use `BUnit.NULL` for
dimensionless quantities (counts, booleans-as-doubles, stage numbers). [ev: corpus B4 §4.2]

**Gotcha:** `BUnit.getUnit(String)` returns null if the key is not in units.xml — guard the result
before passing it to `BFacets.makeNumeric`. A null unit causes a NullPointerException at facet
construction time.

---

## 7 · Persistence contract

| Type | BOG format | Restore path |
|------|-----------|--------------|
| BSimple | `v="encodeToString()"` | `decodeFromString(String)` |
| BStruct | slot machinery (one `v=` per slot) | slot machinery |
| BFrozenEnum | ordinal int as string | `make(int)` via range |
| BDynamicEnum | ordinal int, range from facets | `make(ordinal, range)` |

**`newCopy()` semantics:**
- BSimple: returns `this` (immutable — inherited default is correct).
- BStruct: returns a deep copy via slot machinery (inherited — do not override).
- BComponent: returns a structural deep copy; `REMOVE_ON_CLONE` slots are excluded. [ev: corpus B4 §4.3.3]

**`REMOVE_ON_CLONE`** on a dynamic slot ensures it is NOT replicated when the parent component is
copied (exported, imported, cloned in Workbench). Use for runtime-populated results (learn entries,
transient state bags) that must not survive into a copy. [ev: `types/logic.md` §Flags]

**Slots added dynamically** at runtime are NOT persisted unless `PERSISTENT` flag is set. Transient
discovery result slots should carry `TRANSIENT|REMOVE_ON_CLONE`. [ev: corpus B4]

---

**See also:** `types/logic.md` (slot flags, BStatus producers), `types/actions.md` (BStruct as action parameter),
`types/logic-authoring.md` (ORD, BQL, framework SPIs).
