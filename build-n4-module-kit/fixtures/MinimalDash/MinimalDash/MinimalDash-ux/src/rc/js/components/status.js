/*
 * components/status.js — a pure function (element, state) -> void that patches the DOM.
 */
var MinimalDashStatus = function (el, state) {
  if (state.error) {
    el.textContent = 'error: ' + state.error;
  } else if (state.status) {
    el.textContent = 'status: ' + state.status.status;
  } else {
    el.textContent = 'loading…';
  }
};
if (typeof module !== 'undefined') { module.exports = MinimalDashStatus; }
