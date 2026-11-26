// eslint.config.mjs — the kit's ESLint flat config for dashboard `-ux` browser code (rc/**/*.js).
// No imports: it loads with a bare `eslint` install (toolbelt/eslint/package.json pins it).
// ecmaVersion 2020 = the Chromium 83 HMI panel floor, so a parse error IS a floor violation.
// Classic scripts share one global scope across files, so no-unused-vars checks locals only
// (`vars: "local"`): a top-level function used by another file is not "unused".
// report-module.sh runs it on every `-ux` artifact's src/rc (vendor/, ext/ and *.min.js excluded)
// with the row formatter in toolbelt/eslint/rows-formatter.cjs. Error -> FAIL, warning -> WARN.
// Content-hashed bundle output (name-<8+ hex>.js) is build output, not project source: vendor-floor
// judges it. A file that parses only as an ES module is linted again with eslint.module.config.mjs.
// [ev: issue #1157]
// [ev: retro dashboard-frontend-standard Δ10] [ev: retro dashboard-frontend-standard Δ4]

const browserGlobals = Object.fromEntries([
  'window', 'document', 'navigator', 'location', 'history', 'console', 'fetch', 'Headers',
  'Request', 'Response', 'AbortController', 'URL', 'URLSearchParams', 'FormData', 'Blob',
  'FileReader', 'localStorage', 'sessionStorage', 'setTimeout', 'clearTimeout', 'setInterval',
  'clearInterval', 'requestAnimationFrame', 'cancelAnimationFrame', 'getComputedStyle',
  'matchMedia', 'performance', 'alert', 'confirm', 'prompt', 'Event', 'CustomEvent',
  'MutationObserver', 'ResizeObserver', 'IntersectionObserver', 'HTMLElement', 'Node',
  'DOMParser', 'XMLHttpRequest', 'WebSocket', 'Image', 'atob', 'btoa', 'screen', 'self',
  // DJS1 dual-export shim: `if (typeof module !== "undefined") module.exports = ...`
  'module',
].map((name) => [name, 'readonly']));

// minimatch has no {8,}: eight hex digits, then any further hex digits (extglob), then .js.
export const hashedBundle = `**/*-${'[0-9a-f]'.repeat(8)}*([0-9a-f]).js`;

export default [
  {
    ignores: ['**/vendor/**', '**/ext/**', '**/*.min.js', hashedBundle],
  },
  {
    files: ['**/*.js'],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: 'script',
      globals: browserGlobals,
    },
    linterOptions: {
      reportUnusedDisableDirectives: 'warn',
    },
    rules: {
      'no-unused-vars': ['warn', { vars: 'local', args: 'after-used', caughtErrors: 'none' }],
      'max-lines-per-function': ['warn', { max: 60, skipBlankLines: true, skipComments: true }],
      'no-console': ['error', { allow: ['error'] }],
      eqeqeq: ['error', 'always', { null: 'ignore' }],
    },
  },
];
