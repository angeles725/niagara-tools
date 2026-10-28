/*
 * store.js — the poll loop and the page state; pages subscribe, nothing else calls the api service.
 * Recovery: a read failure after reloadAfterMs without a successful read reloads the page, so a kiosk does
 * not stay blank after a station restart (types/dashboard.md, kiosk recovery ladder).
 */
var MinimalDashStore = (function () {
  function create(api, cfg) {
    var state = { status: null, error: null };
    var subscribers = [];
    var lastOkAt = Date.now();
    function publish(next) {
      state = next;
      subscribers.forEach(function (fn) { fn(state); });
    }
    function poll() {
      return api.status(cfg).then(function (d) {
        lastOkAt = Date.now();
        publish({ status: d, error: null });
      }).catch(function (e) {
        publish({ status: state.status, error: String(e) });
        if (Date.now() - lastOkAt > cfg.reloadAfterMs) { location.reload(); }
      });
    }
    return {
      subscribe: function (fn) { subscribers.push(fn); fn(state); },
      start: function () { poll(); return setInterval(poll, cfg.pollMs); }
    };
  }
  return { create: create };
}());
if (typeof module !== 'undefined') { module.exports = MinimalDashStore; }
