/*
 * services/api.js — the only file that calls fetch (types/frontend-standard.md § Component layout).
 * Every request carries X-Requested-With (the servlet's XHR guard) and an abort signal (a timeout).
 */
var MinimalDashApi = (function () {
  function getJson(cfg, path) {
    var ctl = new AbortController();
    var timer = setTimeout(function () { ctl.abort(); }, cfg.timeoutMs);
    return fetch(cfg.apiBase + path, {
      headers: { 'X-Requested-With': 'XMLHttpRequest' },
      signal: ctl.signal
    }).then(function (r) {
      clearTimeout(timer);
      if (!r.ok) { throw new Error('HTTP ' + r.status); }
      return r.json();
    }, function (e) {
      clearTimeout(timer);
      throw e;
    });
  }
  return {
    // The response fields the front end reads, per route: each must be declared in js/contract.json
    // (srcTest/rc/spec/MinimalDashContractSpec.js checks it; types/dashboard.md DUX-CONTRACT1).
    reads: { 'GET /api/status': ['status'] },
    status: function (cfg) { return getJson(cfg, '/status'); }
  };
}());
if (typeof module !== 'undefined') { module.exports = MinimalDashApi; }
