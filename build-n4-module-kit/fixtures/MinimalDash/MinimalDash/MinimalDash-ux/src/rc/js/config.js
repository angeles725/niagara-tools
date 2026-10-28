/*
 * config.js — the page's timing and API base, one object (types/frontend-standard.md § Component layout).
 * Classic script + the DJS1 dual-export shim: the panel loads it with <script src>, a Node harness requires it.
 */
var MinimalDashConfig = {
  apiBase: '/minimaldash/api',
  pollMs: 5000,
  timeoutMs: 4000,
  reloadAfterMs: 60000
};
if (typeof module !== 'undefined') { module.exports = MinimalDashConfig; }
