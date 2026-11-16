# Type: theme / branding module — zero-Java ux-profile module

A theme module customizes Niagara's visual identity across browser (bajaux/Hx) and Workbench
(JavaFX/Swing). It is a **pure-resource ux jar** — no Java, no `@NiagaraType`, no palette.
`[ev: code themeDistech-ux module.xml]`
Login-page branding from a theme module (`logo`, `loginCss`, `favicon` on `WebService`): `types/horizons.md` § 2.

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
  `selectedHxTheme` / `selectedWorkbenchTheme` ("Hx Theme … defaults to Zebra", guide `User/baja-User.txt:54` "drop-down list (defaults to Zebra)"); the
  Workbench theme is picked in Tools › Options › General (`User/aAboutWorkbenchThemes.txt:26` "To change the Workbench theme, click Tools >"). There is no Platform ›
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
  icons/x16/                            ← generic 16×16 framework icons (405 in themeDistech; 802 PNGs under imageOverrides/)
      action.png  alarm.png  …
  icons/sprite/sprite.png
```

**Path rule:** must exactly match the resource path inside the target jar. A wrong path
silently leaves the default icon in place — no error at deploy [INFER]. `[ev: code themeDistech-ux imageOverrides/]`

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
- `nodeHome` is required (LESS runs via Node). Set it in the user-level `gradle.properties`, never in the module
  tree: `lint-structure` L10 WARNs the key in a client tree (`types/structure.md` § L10 doctrine).
- No `module.palette` or `module.lexicon` is needed for a pure theme.
- **Declare the kind, or `lint-structure` L9 FAILs the part.** A `-ux` part with no Java source and no named palette
  entry is what L9 calls an empty skeleton, and a theme is exactly that shape. The lint never infers a theme from the
  missing files; the module says it in the header of `<module-root>/MODULE-MAP.md` (`types/structure.md` § Module map):

  ```
  # MODULE-MAP — themeAcme

  modularity: strict
  type: theme

  | Concern | Part | Files | Tests | Change here when |
  |---|---|---|---|---|
  | plumbing | ux | themeAcme-ux/themeAcme-ux.gradle.kts; themeAcme-ux/module-include.xml | — | build or packaging only |
  | plumbing | ux | themeAcme-ux/nss/theme.nss; themeAcme-ux/ux/theme.css; themeAcme-ux/hx/theme.css | — | a colour, font or icon of the theme changes |
  ```

  With `type: theme` the part passes L9 when it holds at least one `.nss` file (§ 2: a theme without one is never
  listed) and FAILs without one. Theme assets carry no test, so their rows are `plumbing` rows. Pinned by the
  `kind-theme-*` golden cases of `go/internal/lint/rules/structure` and `TestDeclaredKindMapPassesModuleMap`.

---

## 8 · The official authoring flow

Source for §§ 8-10: [CERT-doc] the two theme pages of the developer guide,
`niagara-help/versions/4.15.6.20/devguide-clean/themes/themeCreation.txt` and
`niagara-help/versions/4.15.6.20/devguide-clean/themes/themingCustomWidgets.txt` (under niagara-research).

**Start from a shipped theme, not from nothing.** The guide's flow, in the kit's terms:

1. Pick the shipped theme closest to the target look (`themeZebra` or `themeLucid`).
2. Create `theme<Name>/theme<Name>-ux/`, extract the donor's `-ux` jar into `src/`, delete `META-INF/`.
3. Move `Gruntfile.js`, `package.json` and `cssTemplate.mustache` from `src/` to the part root (they drive LESS and
   the sprite, below).
4. Replace **every** reference to the donor's module name in those files and in the CSS (§ 5 shows the bug a leftover
   one causes).
5. Declare the module (`runtimeProfile` `ux`) and the `themeName` def (§ 2), build the jar, install it, then choose
   the theme in Workbench (Tools › Options › General) and restart Workbench.

**Which page wins on build syntax.** `themeCreation` still shows a Groovy `.gradle` file, a `<defs>` block typed into
`module-include.xml` and `build.xml` resource entries; the build page of the same guide, which § 2 follows, declares
the def in `moduleManifest { defines { … } }` of a Kotlin build script. Both are official; the build page describes
the current plugin, so § 2 and § 7 stand, and the theme page is read for the asset rules only.

**What each technology styles**, so a change goes into the right file:

| Surface | Technology | File |
|---|---|---|
| Workbench nav tree, Px widgets, dialogs, wire-sheet glyphs | bajaui | `nss/*.nss` (§ 6, § 9) |
| Workbench menu bar, toolbar, scroll bars | JavaFX | `fx/theme.css`, JavaFX CSS (`-fx-…` properties) |
| Classic web views | Hx | `hx/theme.css`, applied on top of the framework's default Hx styles |
| Property Sheet, Web Chart, User Manager and other HTML5 views, the Shell Hx profile | bajaux | `ux/theme.css` |

- **NSS is not CSS and never reaches the browser:** it styles Workbench only.
- **bajaux:** the base class set lives in `module://web/rc/theme/theme.css`; a theme redefines those classes in
  `ux/theme.css`, and may add rules for widgets that do not use them (dialogs, Property Sheet, charts). The station
  serves a visual demonstration of the classes at `/module/web/rc/theme/test.html?theme=<Name>`.
- **JavaFX IDs** worth knowing: `#menu-bar-profile`, `#tool-bar-profile`, `#view-profile-foreground` /
  `-background`, `#content-profile-foreground` / `-background`. A container with no background of its own shows the
  one behind it.
- **LESS is optional.** The donor keeps a shared colour palette in `less/palette.less`, compiled into both
  `hx/theme.css` and `ux/theme.css`; `grunt watch:css` recompiles on save and `grunt less` once (Node.js and
  `grunt-cli` installed, `npm install` in the part directory). Without LESS, edit the two CSS files directly.
- **Branding default:** `brand.properties` may set `workbench.theme.default` and `workbench.theme.locked=true` to
  pin a brand's Workbench to one theme (`types/distribution.md` § 4 for the overlay that carries the file).

## 9 · NSS, icons, fonts and sprites — the rules the guide adds

**NSS syntax** (extends § 6):

- A file is a tree of key-value pairs: a value is a string (`color: black;`) or a nested node in braces. Properties
  cascade **down the braces**: a node inherits its parent's values and overrides the ones it sets.
- `.name` inside a node is a style class, `#name` a style ID; an ID beats a class.
- `#define x = value;` then `$x`. A `#define` cannot use another `#define`.
- **Several `.nss` files give several themes from one module:** each file is listed on its own. `theme.nss` is
  listed under the module's name without `theme`; any other file as `<Module without theme>/<file name>`. They share
  the module's icons.
- **Explicit widget properties win over the theme.** A Px widget whose font or colour is set on its property sheet,
  or linked to a Px property, ignores NSS; leave the property at its null/default value to let the theme paint it.
- Known limits: `padding` tags do not apply per widget, and the `table` font also styles drop-downs.

**Icon overrides** (extends § 4):

- A framework icon is addressed by ORD (`module://icons/x16/refresh.png`,
  `module://alarm/com/tridium/alarm/icons/filterRed.png`); the override is the same path under
  `imageOverrides/<module>/…`. Lookup order: the active theme, then the default icons module, then an error icon.
- **The replacement must have the same pixel dimensions as the original** (a 16×16 icon is replaced by a 16×16).
- The images must be packaged: list the `imageOverrides` paths among the jar's resources in the build script.

**Fonts in Workbench** (the CSS `@font-face` rules of § 5 cover the browser):

- Put the files in `fonts/`. For NSS **only TrueType (`.ttf`) is supported**, and the name used in the style sheet
  is the family name embedded in the font, not the file name. A font value follows the `BFont` string format
  (`12pt Tahoma`).
- **Declare families, do not hard-code one font:**

  ```nss
  font-families {
    sansserif: MyBrandSans, Tahoma, Arial;
    monospaced: Consolas, Courier New;
  }
  font: 12pt sansserif;
  ```

  The first installed font of each list is used; `serif`, `dialog` and `dialoginput` are the other families. A
  family can then be overridden per language from the `bajaui` lexicon (`font.sansserif=…`), which a hard-coded
  `font: 12pt Tahoma;` ignores: use families whenever the theme must render a non-Latin script
  (`types/lexicon.md`).

**Sprites — required as soon as the theme carries images:**

- bajaux views draw icons from one sprite image positioned by CSS, so a module's whole icon set costs two requests.
  A theme that contains images must also ship `sprite/sprite.png` and `sprite/sprite.css`.
- The donor's Grunt configuration generates them: `grunt buildsprite` runs `sprite` (builds both files),
  `imagemin` (lossless compression) and `concat` (prepends the `@noSnoop` tag, which switches off the station HTML writer that
  would otherwise break the direct URLs inside `sprite.css`).
- The class of an icon is derived from its ORD: `icon-` plus the ORD from the module name on, separators as
  hyphens, no extension (`module://icons/x16/add.png` → `.icon-icons-x16-add`). Each rule also hides an inner
  `img`; that is how bajaux detects whether an icon is in the sprite. Build icon HTML with the RequireJS module
  `bajaux/icon/iconUtils`, never by hand.
- **Regenerate the sprite after every change to `imageOverrides/`,** and check that `sprite.css` points at
  `/module/theme<Name>/sprite/sprite.png`, not at the donor (the § 5 bug again). `themeZebra` carries no image
  overrides, so its sprite is empty: use `themeLucid` as the example of a generated one.

## 10 · Making a custom widget themable

A theme styles your widget only if the widget asks the theme. NSS says **what** to paint (font, colours, insets),
never how: a `paint()` that hard-codes a colour ignores every theme.

**bajaux / HTML widget.** The active theme's name is a class on `<body>`. Style from the shared `ux` classes first;
where a theme needs its own look, scope a rule by the body class:

```css
body.Zebra .myWidget { background-color: gray; }
body.Lucid .myWidget { background-color: beige; }
```

A widget ported from outside Niagara that uses none of the shared classes looks the same in every stock theme;
only a custom theme's `ux/theme.css` can restyle it (a documented limitation).

**bajaui (Workbench) widget.**

- Ask the theme for each value **and pass the widget itself**, so its style class and ID are honoured:
  `Theme.label().getTextBrush(this)`, `Theme.button().getControlBackground(this)`. The classes are in
  `com.tridium.ui.theme`: an implementation package, not `javax.baja` API, so recheck it on each framework version you target
  `[INFER]`.
- `getStyleSelector()` on a `BWidget` names the NSS node the widget reads; without it each `Theme.<kind>()` object
  uses its default node. The ones a custom widget usually needs:

  | Theme object | Default NSS node | | Theme method | NSS tag |
  |---|---|---|---|---|
  | `Theme.widget()` | the NSS root | | `getTextBrush` | `color` |
  | `Theme.label()` | `label` | | `getTextFont`, `getPlainText` | `font` |
  | `Theme.button()` | `button` | | `getBoldText` | `bold-font` |
  | `Theme.pane()` | `pane` | | `getControlBackground` | `background-color` |
  | `Theme.table()` | `table` | | `getControlForeground` | `outline` |
  | `Theme.textField()` | `text-field` | | `getSelectionBackground` / `getSelectionForeground` | `selection-background-color` / `selection-color` |
  | `Theme.wiresheet()` | `wire-sheet` | | `getWindowBackground` | `window-background-color` |

  The full lists are in `themingCustomWidgets`.
- **The tag name decides the value type** when a theme author adds a tag for your widget: a name ending in `color`,
  `outline`, `highlight` or `shadow` parses as a `BBrush`, `font` as a `BFont`, `padding` as `BInsets`, `image` as a
  `BImage`, `height` or `width` as a number, anything else as a string. Name custom tags accordingly.
- **Scroll panes and text editor panes:** read the pane background with `Theme.scrollPane().getWindowBackground()`
  (likewise `Theme.textEditorPane()`); `getControlBackground()` on them returns the root background colour.

Custom-painted widgets: `types/wb-widgets.md`. Checking a theme without changing user options: `wb -theme:<name>`
(`types/diagnostics.md` § 4.3).

---

**See also:** `types/structure.md` (L9 empty-skeleton check and the `type: theme` declaration, L10 `nodeHome` rule).
