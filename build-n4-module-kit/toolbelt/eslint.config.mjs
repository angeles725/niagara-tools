// eslint.config.mjs — the kit's ESLint flat config for dashboard `-ux` browser code (rc/**/*.js).
// No imports: it loads with a bare `eslint` install (toolbelt/eslint/package.json pins it).
// ecmaVersion 2020 = the Chromium 83 HMI panel floor, so a parse error IS a floor violation.
// Classic scripts share one global scope across files, so no-unused-vars checks locals only
// (`vars: "local"`): a top-level function used by another file is not "unused".
// report-module.sh runs it on every `-ux` artifact's src/rc (vendor/, ext/ and *.min.js excluded)
// with the row formatter in toolbelt/eslint/rows-formatter.cjs. Error -> FAIL, warning -> WARN.
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

export default [
  {
    ignores: ['**/vendor/**', '**/ext/**', '**/*.min.js'],
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
