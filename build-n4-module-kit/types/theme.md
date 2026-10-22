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

The **only** registration mechanism is a registry def, declared in the part's build script
(`moduleManifest { defines { define("themeName", "themeAcme") } }`, devguide `build.txt:337-358`), which the
plugin writes into the generated `module.xml`:

```xml
<defs>
  <def name='themeName' value='themeAcme'/>
</defs>
```

- **The value is the module name, and the module name starts with `theme`.** Discovery reads every `themeName`
  def and resolves `module://<value>/nss` (vineflower `CustomThemeModuleManager.java:24,47`); Hx and bajaux load
  `module://theme<Tag>/hx/theme.css` and `…/ux/theme.css` (vineflower `BHxProfile.java:291`, vineflower `UxUtil.java:23`). Every shipped
  theme follows it (`themeDistech` declares `value='themeDistech'`).
- **The picker lists `nss/*.nss` files.** A theme with no `nss/` directory, or no `.nss` in it, is never listed; the
  label is the file name with the `theme` prefix stripped (`theme.nss` in `themeAcme` → `Acme`,
  vineflower `CustomThemeModuleManager.java:64-72`).
- **A missing theme falls back silently** to Zebra, then Lucid, then the first listed theme (`getDefaultThemeEnum`).
- **Choosing a theme is per web profile, not per module:** the user's (or prototype's) web profile carries
  `selectedHxTheme` / `selectedWorkbenchTheme` ("Hx Theme … defaults to Zebra", guide `User/baja-User.txt:54`); the
  Workbench theme is picked in Tools › Options › General (`User/aAboutWorkbenchThemes.txt:26`). There is no Platform ›
  User Interface picker.

`[ev: corpus B1328 §1328.2 C1]` `[ev: corpus B1328 row 3]` `[ev: retro devguide415-r2-web-ui-hmi Δ2]`

No `module-include.xml` type entries, no palette, no lexicon keys are required.

---

## 3 · Asset layout

```
theme<Name>-ux/            # module name = the themeName def value
├── ux/theme.css          # bajaux active browser profile (Hx 4.x+ / HTML5)
├── hx/theme.css          # legacy Hx browser profile
├── fx/theme.css          # JavaFX Workbench browser-side styling
├── nss/theme.nss         # Workbench Niagara Style Sheet (Swing/FX widgets); REQUIRED — no .nss, not listed
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
- `module-include.xml` must still be present but holds only `<types/>`; the `themeName` def goes in
  `moduleManifest { defines { … } }` (§ 2).
- **A servlet SPA is not themed automatically.** Tridium injects the theme CSS only into its own pages (Hx profile,
  vineflower `WbWebWidgetServlet.java:133,143-149`, the bajaux resolver); a module's own HTML must link
  `/module/web/rc/theme/theme.css` and `/module/theme<Name>/ux/theme.css` itself, or render through a bajaux
  view (`types/frontend-standard.md` § 1). `[ev: corpus B1349 §1349.2]` `[ev: retro devguide415-r3-web-session-headers Δ4]`
- `nodeHome` in `gradle.properties` is required (LESS runs via Node). `[ev: types/structure.md §L10]`
- No `module.palette` or `module.lexicon` is needed for a pure theme.

---

**See also:** `types/structure.md` (L9 empty-skeleton check, L10 `nodeHome` rule).
