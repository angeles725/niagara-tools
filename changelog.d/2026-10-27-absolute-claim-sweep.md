### Changed — absolute-claim sweep of types/

- `types/logic-authoring.md`: a proxy-ext `write(Context)` that posts the write to a worker returns `true` (a write
  is pending; finish with `writeOk`/`writeFail`), per the devguide BProxyExt contract; the BQL `get()` comment names
  why the caller's Context matters (read permissions).
- `types/dashboard.md`: slot lexicon keys stay bare, and the one prefixed key is a type's `<Type>.displayName`.
