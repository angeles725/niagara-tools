# Lexicons and i18n — resolution, translation modules, runtime text, browser `lex!`

Load with any module that shows text to an operator. Key forms (slot, `<Type>.displayName`, enum tag,
`%lexicon()%`) are in `types/structure.md` § Lexicon key catalogue; packaging rules (bare keys, ASCII only, one
translation per key, verify in both clients) are in `types/dashboard.md` § Module packaging. This card adds how a
lookup resolves, how a translation ships, where runtime text comes from, and the browser side.

Derivations: corpus B1305 (devguide + `javax.baja` source), B1330 (Lexicon guide), B1236 (course deck 7).
Paths: `D=organized/docSource/docSource-doc/extracted/baja/javax/baja/util` (4.15.6.20 source),
`V=organized/baja/baja/vineflower/javax/baja` (4.14.0.162 decompile), under niagara-research; devguide and guide
paths are under `niagara-help/versions/4.15.6.20/devguide-clean/` and `niagara-help/guides-clean/Lexicon/`.

## 1 · How a lookup resolves `[ev: retro devguide415-lexicons-ux-i18n Δ2]`

- **Files.** The root `module.lexicon` in the jar is the fallback; a language file comes from a lexicon module or from
  `!lexicon/<lang>/<module>.lexicon` (devguide `localization.txt:18`).
- **Chain.** `Lexicon.get(key)` reads the language properties, then the root `module.lexicon`, then returns `null`; a
  full locale (`fr_CA`) with no file falls back to the two-letter language. `getText(key[, args])` returns the KEY
  when it is absent, so a text equal to its key means a missing key. [CERT] `D/Lexicon.java:265-270,326-338,346-360`.
  A slot or type display name falls back to `toFriendly` instead (`types/structure.md` § Lexicon key catalogue).
- **Locale.** Workbench uses its VM locale (OS default or `-locale:<lang>`) for the whole session, whatever station
  user logs in. A browser session uses `User.language`, then `Accept-Language`, then the station VM default
  (devguide `localization.txt:12-14,96-108`). Station code passes the caller's `Context`; a `null` Context means the
  station VM locale, not the operator's.

## 2 · Shipping a translation: lexicon modules `[ev: retro devguide415-r2-packaging-platform-signing Δ5]`

- **Manifest.** A module provides lexicons for OTHER modules through `moduleManifest { lexicons { … } }`, which the
  plugin writes as `<lexicons><lexicon module=… resource=… language=… default=…/></lexicons>` in `module.xml`
  (devguide `build.txt:363-430`). Never in `module-include.xml` (`types/module-wiring.md`). The base jar is unchanged.
- **Lexicon Module Builder** (Workbench Lexicon Tool): name `customLexicon<Xxxx>`, `<Vendor>Lexicon<Xxxx>` or
  `<Brand>Lexicon<Xxxx>`; Preferred Symbol 1-8 letters; Vendor and Version required. An optional Brand Pattern loads
  the lexicons only when the licence brand matches. The module may not hold Java classes or the default
  `niagaraLexicon<xx>` jars, and source file names must be unique (`devkit-LexiconModuleBuilder.txt:20-56`).
- **Customer edits** made in the Lexicon Editor are saved as files under `!lexicon/<lang>` (the folder appears on the
  first save) and reach browser clients after a station restart. A Display Name set on a slot overrides any lexicon
  value (`aLexiconOverview.txt:20-24`, `ID-1086-00000188.txt:40`).

## 3 · Runtime text in Java `[ev: retro n4-course-lexicon-logging-testing Δ2]` `[ev: retro devguide415-lexicons-ux-i18n Δ3]`

```java
private static final LexiconModule LEX = LexiconModule.make(BWorkOrderService.class);   // 4.8+
configFail(LEX.getText("workOrderService.badStatusCode", cx, statusCode));
```

- **Operator text from the lexicon, integrator logs in English.** Fault causes, validator messages and job-log lines
  (`types/logic-authoring.md` § Background jobs, `log().message(module, key, args)`) are lexicon keys looked up with
  the caller's `Context`; logger messages stay literal (course Lab 7 `BWorkOrderService.java:107-185,272`; corpus
  B1236 §1236.4, §1236.7). `types/observability.md` § 3 shows the fault-slot side.
- **MessageFormat.** `getText(key, cx, args…)` formats with `MessageFormat`: `{0}` placeholders, and `''` for one
  apostrophe in an entry read with arguments. [CERT] `V/util/LexiconModule.java:107-113`.
- **Exceptions.** Throw `new LocalizableRuntimeException("<module>", "<key>", new Object[] { … })` (checked:
  `LocalizableException`). Its `getMessage()` was formatted at construction with a `null` Context (VM locale); show it
  to a user with `toString(cx)`. A missing key renders as `<key> {args}`. [CERT]
  `V/sys/LocalizableRuntimeException.java:8-9,40-42`, `V/sys/LocalizableException.java:51-70`.
- **Deferred text.** `LexiconText.toLocalizable(module, key, args…)` returns a `Localizable` to render later with the
  reader's Context. [CERT] `V/util/LexiconText.java:65-78`.

## 4 · Browser side: `lex!` `[ev: retro devguide415-lexicons-ux-i18n Δ5]`

```js
require(["lex!myModule,bajaui"], function (lexicons) {
  var text = lexicons[0].get("status.alarmCount", count);   // {0} ← count
});
```

- `lex.get(key, …args)` replaces `{n}` only; there is no MessageFormat, so an entry written with `''` for Java shows
  both quotes in the browser. A missing key returns the `def` argument or an empty string, not the key. Use
  `getSafe` (4.8+) for text placed in HTML. (jsdoc `js-ux/lex_lex.js.txt:275-288,422-436,987-997`.)
- **Lexicon content is public.** Every `web:LexiconRpc` method is `@NiagaraRpc(permissions="unrestricted")`, and
  `getLexicon(moduleName, lang, cx)` returns the full root + language property set of any installed module in any
  language (web-rt decompile `com/tridium/web/rpc/BLexiconRpc.java:44-70`). Put only display text in
  `module.lexicon`: never credentials, tokens, internal hostnames or configuration.
