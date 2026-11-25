### Changed

- `go/internal/jssrc` (`Spans`, `CommentAt`, `LineCommentAt`): one pass over a JavaScript source gives its comment and literal spans (strings, templates with `${}` nesting, regex literals). `lint-bql-unbounded` and `lint-subscribe-js` read their `// lint-…:` waiver comment through it instead of two copies of `jsCommentAt`; findings and goldens unchanged (`lint-niagara-rpc` had no JS comment copy: it already tokenizes JS with `jssrc`). A quote inside a regex literal no longer hides a trailing waiver comment.
