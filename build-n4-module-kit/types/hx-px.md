# Type: Hx and Px in a module (hx-px)

How a module ships a Px graphic, an Hx view or a Px widget, and which of the three web technologies to pick. This card
routes and states the rules that bite; the long recipes stay where they are: the Hx view skeleton in
`types/wb-widgets.md` § Hx view authoring, the Px binding table in § PX authoring, the Px target-media rules in
`types/dashboard.md` § ux "Px rules", the theme files in `types/theme.md`.

Evidence: `[CERT-doc]` the developer guide copy 4.15.6.20 under niagara-research (`niagara-help/versions/4.15.6.20/
devguide-clean/`); `[CERT]` source of the 4.14.0.162 corpus (vineflower copies of `hx-wb`, `kitPx-wb`, `bajaui-wb`);
`[ev: corpus B<n> §…]` a research block; `[INFER]` the kit's reading, not stated by a source.

## 1. Which technology: Hx, Px or bajaux

| You need | Pick | Why |
|---|---|---|
| An equipment graphic bound to slots, drawn by an engineer, no Java | **Px** (`.px` file) | a Px file is a widget tree with data bindings, made in the PxEditor |
| A new browser UI on a component type (form, manager, tool) | **bajaux** `@AgentOn` view | Hx is the legacy framework; new UI is bajaux or a servlet-SPA |
| A bespoke dashboard with its own JSON API | **servlet-SPA** | `types/dashboard.md` |
| A Workbench view or Px widget of yours that must also open in a browser on an Hx profile | **Hx** peer of the view or widget | Hx reuses the Workbench model, and Px is its main reuse target |
| Extending a module that is already Hx | **Hx** | no JavaScript build, no client state |

- **Hx is the classic server-rendered framework and it is maintained but legacy.** The page is written by the server
  from a servlet view; several profile methods are deprecated since 4.13. `[ev: corpus B433 §433.1]`
  `[ev: corpus B433 §433.7]`. The guide gives Hx the job of "transparently produce lightweight HTML-only interfaces
  automatically based on the workbench views", with "Px is the main reuse target". `[CERT-doc]`
  `niagara-help/versions/4.15.6.20/devguide-clean/hx.txt:10` "Limited support exists for standard views like the
  Property Sheet".
- **A Px page is rendered by one of three engines, chosen by the target media and the user's web profile.** Workbench
  paints every widget; Hx accepts only a widget that has an Hx agent; the ux media renders through bajaux from 4.10.
  `[CERT]` `hx-wb` `com/tridium/hx/BHxPxMedia.java:36-37` (`AgentFilter.is(BHxPxWidget.TYPE)` and
  `…is(BHxPxBinding.TYPE)`); `[CERT-doc]`
  `niagara-help/versions/4.15.6.20/devguide-clean/px.txt:30` "viewing a px file in hx only supports a subset of
  widgets". The media choice and its portability rules are in `types/dashboard.md`;
  Mobile Px is obsolete (`types/frontend-standard.md`).
- **An Hx-only station has no bajaux Px.** A station whose `kitPx` has no `kitPx-ux` jar (EC-Net 4.3) opens every Px
  page in a browser through Hx. `[ev: corpus B853 §1]`. Check the target's modules before choosing widgets.
- **The profile is per user, not per module.** The Hx profile lives on the user (`BWebProfileConfig`); the same
  `view:` ORD opens an Hx page for one user and a bajaux page for another. `[ev: corpus B433 §433.5]`. Test under the
  profile the operator has.
- **A Px page cannot run script.** The grammar has no `<script>`; behaviour comes from bindings and converters.
  `[INFER]` (corpus B291 §291.1: "a `.px` has no scripting"). To put JavaScript on a Px page, embed a bajaux widget:
  `BWebWidgetWrapper` takes the widget as an ORD in its `js` property. `[ev: corpus B895 §895.1]`
  `BWebWidget` keeps only a registered view from that ORD, so a `file:` ORD is silently dropped. `[CERT]`
  `workbench-wb` `BWebWidget.java:571`.

## 2. File layout in a module

| Piece | Where | Notes |
|---|---|---|
| `.px` graphic | a file, in the station file space (e.g. `file:^px/<name>.px`) or a module resource | a file; it cannot sit in a component folder |
| Px widget / binding / converter Java | `-wb` (`BWidget` subclass) | the widget tree is a Workbench model |
| Hx peer of a widget or binding | `-wb`, `BHxPxWidget` / `BHxPxBinding` subclass | registered as an agent on the widget type |
| Hx view | `-wb` (`BHxView`, depends on `hx-wb`) | registered `@AgentOn` the component type, or on `BWbView` |
| bajaux widget / view | `-ux` (`rc/`, `BJsBuild`, `BIJavaScript`) | `types/wb-widgets.md` § -ux module JS toolchain |
| Hx and bajaux theme CSS | the theme module: `hx/theme.css`, `ux/theme.css` | `types/theme.md` |

- **A `.px` is a file, and its bindings reach into the component space.** It cannot be placed in a component folder;
  `file:^` is an anchor on `<station>/shared/`, not on the station folder, and it resolves the same from any host
  graphic. `[ev: corpus B289 §289.1]` `[ev: corpus B289 §289.3]`
- **A vendor Px library ships `-wb` only.** The galileoKitPx module carries its widgets, bindings and Hx render agent
  in one `-wb` jar (corpus B423 §423.1 lists no `-rt` or `-ux` part). `[ev: corpus B1079 §2]`
- **Register an Hx view on `BWbView` to share one ORD, or on the component type for an Hx-only page; a Px widget
  gets a `BHxPxWidget` peer.** The recipe and its `op.get()` contract: `types/wb-widgets.md` § Hx view authoring.
  The Hx profile swaps a Workbench view for its registered Hx peer. `[ev: corpus B433 §433.5]`
- **Name every module the Px uses in its `<import>` list.** A widget, binding or converter from an unlisted module
  fails to load with `PxDecoder … Unknown type <X>`; `converters` (`ObjectToString`, `IBooleanToSimple`) and `control`
  (`Override`) are the usual misses. `[ev: corpus B853 §3]`. `PopupBinding` needs `kitPx`.
  `[ev: corpus B289 §289.5]`
- **Edit a hand-written `.px` as the PxEditor writes it.** The encoder writes a start tag and all its attributes
  on one line (vineflower `bajaui-wb` `PxEncoder.java:115-121`); a tag split over lines was seen to raise
  `XException: Expecting '='`. `[INFER]` (corpus B181 §181.5).

## 3. Binding ORDs and their audit

- **Bind with relative ORDs and one `.px` serves every component of the type.** Bindings in a Px view resolve against
  the component the view is on. `[CERT-doc]` `niagara-help/versions/4.15.6.20/devguide-clean/px.txt:24` "always
  resolved relative to the current ord". Absolute `station:|slot:/…` ORDs pin the page to one station layout; the
  `$(var)` and tag-based alternatives are in `types/dashboard.md` § ux "Px rules".
- **A `SetPointBinding` ORD must end on the property being written.** If bound to a component or a read-only property
  it tries a `set` action instead. `[CERT]` `kitPx-wb` `com/tridium/kitpx/BSetPointBinding.java:49-52` ("must resolve
  down to the specific property").
- **A binding to a typed action must leave `actionArg` empty in Hx.** `BHxPxActionBinding` returns the stored string
  unconverted, and `BHxPxValueBinding.handleAction` passes it to `invoke`; for `BNumericWritable.set` (a `BDouble`
  parameter) the write fails with no message. Empty `actionArg` opens the Set dialog, whose result is typed. Actions
  with no parameter (`active`, `inactive`, `auto`) invoke directly; `<Override name="actionArg"/>` (empty) skips
  the duration prompt. `[CERT]` `kitPx-wb` `BHxPxActionBinding.java:114-117`; `[CERT]` `hx-wb`
  `BHxPxValueBinding.java:191-208`; `[ev: corpus B853 §4]` `[ev: corpus B853 §5]`. The bajaux `ActionBinding.js`
  decodes the string, so the same page behaves differently per engine: test every action binding in the browser,
  not only in Workbench. `[ev: corpus B853 §1]`
- **`n4_px_binding_audit` finds bindings whose ORD does not resolve.** Run it after every change to a `.px` or to
  the slots it binds (mcp-n4, read-only): `root` audits every Px view below it, or `file_ord` with `base` audits one
  file; relative ORDs are read against the component of the view (or `base`). A binding is the `ord` of any element
  whose tag ends in `Binding`. Each distinct target loads once over BOX: it resolves, or it is `missing` (listed in
  `broken`) only when its deepest ancestor that loads has no slot of that name. Anything the tool cannot prove is
  `unknown` with a `why` and is not a failure: another scheme (`history:`, `file:`, `module:`), a chained ORD, a
  relative ORD with no `base`, a handle that does not load. Read `counts` and `truncated` before saying "clean":
  views past `max_views` (50) or bindings past `max_bindings` (500) are not audited. `mcp-n4-kit/README.md:495-506`;
  `go/internal/mcpn4/testdata/tools/n4_px_binding_audit.json`.
- **What the audit does not prove:** that a resolved ORD is the right component, that an action takes the argument
  the binding sends (the Hx failure above), or that the widget has an Hx agent. Those need the browser.
  `[INFER]`
- **`PxInclude` reloads when its `ord` or `variables` change.** An include is a widget, so `visible` binds to a point
  like any widget (a collapsible menu); it loads on a background thread, so its root is null until loaded
  (corpus B188 §188.2). `[CERT]` `bajaui-wb` `BPxInclude.java:132`.
- **A one-button toggle needs no scripting:** `ToggleButton` carries a boolean in `selected` and a `SetPointBinding`
  writes it to a point. `[CERT]` `bajaui-wb` `BToggleButton.java:51`.

## 4. Theming

- **Hx styling is CSS only.** Core colours and fonts come from `module://hx/javax/baja/hx/default.css`. Write the
  framework class first and yours after it, avoid `style`, and let a profile's own stylesheet load last.
  `[CERT-doc]` `niagara-help/versions/4.15.6.20/devguide-clean/hx-theming.txt:15` "The default class should always
  come first". A theme module restyles it from `hx/theme.css`: `types/theme.md` § 5 and § 8.
- **Hx and bajaux read different files of the theme module.** A copy of a donor theme keeps the donor's font URLs in
  `hx/theme.css`: search all three CSS files for the donor name (`types/theme.md`).
  `[ev: code themeDistech-ux extracted/hx/theme.css lines 4-16]`
- **A Px widget draws from the theme unless a property says otherwise.** A font or colour set on the widget, or
  linked, ignores the NSS theme; leave it default. `types/theme.md` § 9. A custom-painted widget asks the theme
  for its colours: `types/wb-widgets.md` § Custom-painted Workbench widget.
- **A servlet-SPA is not themed by Niagara.** Only Tridium's own pages get the theme CSS (`types/theme.md`).

## 5. Events and live values

- **An Hx event starts in the browser and answers with JavaScript.** The form is sent with every event, so
  `op.getFormValue()` reads the page state; events ride on `HxView.process`. A `javax.baja.hx.Command` is a user
  event with a display name and is not `javax.baja.ui.Command`. `[CERT-doc]`
  `niagara-help/versions/4.15.6.20/devguide-clean/hx-events.txt:8` "Events always originate from the client browser",
  `:14` "is not the same class as the javax.baja.ui.Command". The dispatch is header-keyed
  (`EVENT_PATH`, `EVENT_ID`) and a view that overrides `process` calls `super.process(op)`. `[ev: corpus B433 §433.4]`
- **Live values in Hx are polled, every 5 s by default.** There is no push: the page starts the poll with
  `hx.started(isDynamic, pollFreq)`, `hx.poll.freq` defaults to 5000 ms and a view opts in with `setDynamic()`.
  `[CERT]` `hx-wb` `BHxProfile.java:345`, `HxUtil.java:1179`; `[ev: corpus B433 §433.6]`. Plan for a lag of up to
  5 s on anything not caused by the user's own click; an action invoked through a Px binding refreshes at once
  (`HxUtils.forceUpdate`, `BHxPxValueBinding.java:196-207`) `[CERT]`. In Hx the Px `visible` property also updates
  every 5 s (`types/dashboard.md`).
- **A bajaux Px page lives on a BOX subscription.** The widget subscribes and the station pushes changes; an Hx page
  does not. `[ev: corpus B752 §752.4]`.
- **Chromeless Hx page:** the view parameter `fullScreen=true` (on a `|view:` query) drops the profile's chrome;
  the default is `false`. `[CERT]` `hx-wb` `BHxProfile.java:87,206-207`.

## 6. Security and permissions

- **A Px or Hx page is not an access control.** A Px binding invokes an action as the logged-in user: Hx passes the
  `HxOp` as the invoke `Context`, and its action popup lists only the actions the user's permissions allow
  (operator-flag actions need `operatorInvoke`, bit 4; the others `adminInvoke`, bit 64). The station permission on
  the target component is the gate, so grant that, not the page (`types/security.md` § 2.4: `invoke` checks the
  `Context` it is given). `[CERT]` `hx-wb` `BHxPxValueBinding.java:165-170,196`.
- **`requiredPermissions` on an agent only decides whether the view is offered.** Vendor views register with none and
  expose unrestricted calls; real authorization is a server-side check. `[ev: corpus B752 §752.5]`
- **A custom widget that gates on a user mixin or a PIN fails closed.** A missing mixin hides and disables the widget;
  a pin of `-1` lets any level pass once the mixin exists. `types/security.md` § 2.3; the integer-pin slots:
  `types/wb-widgets.md` § Integer `visibilityPin`/`actionPin`. `[ev: corpus B1079 §1]`
- **Do not bypass the profile's CSRF token.** Every Hx page writes it into its form and verifies it on POST; Hx code
  reads it with `hx.getCsrfToken()` and bajaux with `csrfUtil.getCsrfToken()`. `[CERT-doc]`
  `niagara-help/versions/4.15.6.20/devguide-clean/security/csrfProtection.txt:81` "Bajaux and Hx views have APIs to
  access the CSRF token". `[ev: corpus B433 §433.3]`. Token carriers: `types/security.md` § 3.1.
- **Escape every value an Hx view writes**, and keep the station's `X-Frame-Options` and CSP defaults: Hx pages need
  `Sameorigin` (`types/security.md` § 4.3 and § 7; `types/wb-widgets.md` § Hx view authoring).
- **A Velocity Px (`.pxvm`) needs the `axvelocity` licence feature** and can no longer be opened in the PxEditor;
  keep the editable part in a `PxInclude` file. Generate ORDs from the component tree, not from text a user
  typed `[INFER]`. `[CERT-doc]` `niagara-help/versions/4.15.6.20/devguide-clean/velocity/pxVelocity.txt:10` "does
  require the ‘axvelocity’ license feature", `:158` "any Px file with VTL code".

## 7. Pitfalls found in the research blocks

- **A widget with no Hx agent is silently absent in Hx while Workbench draws it.** The media warns in the PxEditor
  only; nothing fails at runtime. Open the page in the browser profile before shipping. `[CERT]` `hx-wb`
  `BHxPxMedia.java:36-37`; `[ev: corpus B853 §1]`
- **A fresh image on `file:^` is served without a restart.** Station files are sent with `max-age=0` and revalidated;
  only module resources get the 30-day lifetime. A highlight image that does not show is not a cache problem.
  `[CERT]` `web-rt` `com/tridium/web/servlets/FileServlet.java:244-246`.
- **An `EnumWritable` range is fixed in the component.** It does not follow a name held in another slot; renaming
  the source means editing every range. `[CERT]` `control-rt` `BEnumWritable.java:294-306`
- **Shape the point for the browser.** A fixed-value numeric button cannot work in Hx, so drive a boolean writable
  with `active`/`inactive`/`auto`, or let the user type a numeric value in its Set dialog. `[ev: corpus B853 §7]`
- **Copying a wired block between pages:** plain paste drops the links whose source is outside the selection; use
  Paste Special with "Keep all links" and a copy count. `[ev: corpus B853 §2]`
