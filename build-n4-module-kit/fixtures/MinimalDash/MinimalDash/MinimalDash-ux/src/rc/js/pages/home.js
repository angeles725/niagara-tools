/*
 * pages/home.js — one file per nav tab: composes components and subscribes to the store.
 */
var MinimalDashHome = {
  mount: function (root, store) {
    var el = root.querySelector('#status');
    store.subscribe(function (state) { MinimalDashStatus(el, state); });
  }
};
if (typeof module !== 'undefined') { module.exports = MinimalDashHome; }
