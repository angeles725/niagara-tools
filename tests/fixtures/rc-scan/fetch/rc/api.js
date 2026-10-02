function read(url) {
  return fetch(url, { credentials: "same-origin" })
    .then(function (r) { return r.json(); });
}
function probe() { return window.fetch("/api/version"); }
