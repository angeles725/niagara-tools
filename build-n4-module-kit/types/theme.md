# Type: theme / branding module — zero-Java ux-profile module

A theme module customizes Niagara's visual identity across browser (bajaux/Hx) and Workbench
(JavaFX/Swing). It is a **pure-resource ux jar** — no Java, no `@NiagaraType`, no palette.
`[ev: code themeDistech-ux module.xml]`

---

## 1 · What a theme module is

- `runtimeProfile="ux"`, `autoload="true"` — NRE loads it at station start automatically.
- Zero Java: no `.java` source, no types in `module.xml`.
- Sole deliverable: CSS, fonts, NSS, and icon overrides packed into a ux jar.

```xml
<!-- META-INF/module.xml (generated) — key fields -->
<module name='themeDistech-ux' runtimeProfile='ux' autoload='true' nre='true'
        moduleName='themeDistech' preferredSymbol='tdis'>
  <types></types>          <!-- MUST be present and empty -->
  <defs>
    <def name='themeName' value='themeDistech'/>
  </defs>
</module>
```

`<types></types>` empty is mandatory — registering a type here turns it into a code module.
`[ev: code themeDistech-ux module.xml]`

---

## 2 · Registration — the `themeName` def

The **only** registration mechanism is:

```xml
<defs>
  <def name='themeName' value='<your-theme-name>'/>
</defs>
```

The platform scans all installed ux modules for a `themeName` def and lists them in the
Workbench theme picker (Platform → User Interface → Look and Feel). The value is the display
name; it must be unique across installed modules. `[ev: code themeDistech-ux module.xml]`

No `module-include.xml` type entries, no palette, no lexicon keys are required.

---

## 3 · Asset layout

```
<themeName>-ux/
├── ux/theme.css          # bajaux active browser profile (Hx 4.x+ / HTML5)
├── hx/theme.css          # legacy Hx browser profile
├── fx/theme.css          # JavaFX Workbench browser-side styling
├── nss/theme.nss         # Workbench Niagara Style Sheet (Swing/FX widgets)
├── fonts/                # OTF / WOFF font files
│   ├── MyFont-Regular.otf.woff
│   └── MyFont-Regular.ttf
├── less/                 # LESS source (optional; compiled to ux/theme.css)
├── sprite/               # sprite images
└── imageOverrides/       # icon substitutions per type context (see §4)
```

Fonts are served at `/module/<moduleName>/fonts/<filename>`. The `moduleName` in the URL is
the `name` attribute on `<module>` with the `-ux` suffix stripped — i.e. the `moduleName`
attribute value. `[ev: code themeDistech-ux extracted/ux/theme.css]`

---

## 4 · `imageOverrides/` — icon substitution

`imageOverrides/` mirrors the Java package path of each target module's icon resources. The
framework substitutes any PNG found here for the matching canonical icon, per type context.

```
imageOverrides/
  bacnet/com/tridium/bacnet/ui/icons/   ← mirrors target module's jar resource path
      bacObject.png
  history/com/tridium/history/ui/icons/
      delete.png
  icons/x16/                            ← generic 16×16 framework icons (~800 in themeDistech)
      action.png  alarm.png  …
  icons/sprite/sprite.png
```

**Path rule:** must exactly match the resource path inside the target jar. A wrong path
silently leaves the default icon in place — no error at deploy. `[ev: code themeDistech-ux imageOverrides/]`

---

## 5 · CSS per profile

Each profile's `theme.css` declares `@font-face` rules and overrides framework CSS variables.
Font URLs use the absolute `/module/<name>/fonts/…` path:

```css
/* ux/theme.css — correct: references THIS module's fonts */
@font-face {
  font-family: 'SourceSansPro';
  font-weight: normal;
  src: local('?'),
       url('/module/themeDistech/fonts/SourceSansPro-Regular.otf.woff') format('woff'),
       url('/module/themeDistech/fonts/SourceSansPro-Regular.ttf') format('truetype');
}
```

**Gotcha — copy/paste font-URL bug (observed in themeDistech):** `hx/theme.css` was cloned
from another theme (`themeZebra`) and still references `/module/themeZebra/fonts/…`. The Hx
browser loads the font from the WRONG module. The file compiles and deploys without error; the
wrong font appears only at runtime under the Hx browser profile. Always search all three CSS
files for the donor module name after copying. `[ev: code themeDistech-ux extracted/hx/theme.css lines 4-16]`

---

## 6 · NSS — Workbench widget styling

`nss/theme.nss` uses Niagara Style Sheet syntax for Workbench Swing/FX widgets.

```nss
// Variable definitions MUST appear before any rule block.
// Gotcha: no space between function name and opening parenthesis.

#define buttonVertical =
  lineargradient(
    stop(0%, #b0cbf5) stop(100%, #d9e8ff)
    angle(90) );

#define menuBar =
  lineargradient(
    stop(0%, #f7f7f7) stop(100%, #f4f5eb)
    angle(90) );

// Reference a variable with $:
.button { background: $buttonVertical; }
```

Key NSS rules:
- `#define name = value;` declares a variable; `$name` expands it anywhere.
- `lineargradient(stop(pct, color) … angle(deg))` is the gradient function; the angle is
  degrees (0 = horizontal, 90 = vertical top-to-bottom).
- No space between `lineargradient` and `(` — the parser rejects `lineargradient (…)`.
- All `#define` blocks must precede rule blocks in the file.

`[ev: code themeDistech-ux extracted/nss/theme.nss]`

---

## 7 · Build notes

- The `-ux` Gradle plugin compiles LESS (if present) and packages resources into the ux jar.
  No Java compilation step runs.
- `module-include.xml` must still be present but holds only `<types/>`.
- `nodeHome` in `gradle.properties` is required (LESS runs via Node). `[ev: types/structure.md §L10]`
- No `module.palette` or `module.lexicon` is needed for a pure theme.

---

**See also:** `types/structure.md` (L9 empty-skeleton check, L10 `nodeHome` rule).
