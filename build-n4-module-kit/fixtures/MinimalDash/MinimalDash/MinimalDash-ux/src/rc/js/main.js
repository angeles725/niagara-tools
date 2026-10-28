/*
 * main.js — boot only: wire the store to the visible page and start polling.
 */
(function () {
  var store = MinimalDashStore.create(MinimalDashApi, MinimalDashConfig);
  MinimalDashHome.mount(document, store);
  store.start();
}());
