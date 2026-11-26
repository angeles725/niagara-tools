// eslint.module.config.mjs — the kit flat config for an `-ux` file that parses only as an ES module
// (loaded with <script type="module">). Same rules, ecmaVersion and ignores as eslint.config.mjs;
// only sourceType changes. report-module.sh lints a file with it when the script-mode run reports
// exactly "'import' and 'export' may appear only with 'sourceType: module'" for that file, so any
// other parse error (syntax above ecmaVersion 2020) stays a FAIL. [ev: issue #1157]
import base from './eslint.config.mjs';

export default base.map((o) => (o.languageOptions
  ? { ...o, languageOptions: { ...o.languageOptions, sourceType: 'module' } }
  : o));
