function fetchT(url, opts, ms) {
  const c = new AbortController();
  const t = setTimeout(function () { c.abort(); }, ms);
  return fetch(url, Object.assign({}, opts, {
    signal: c.signal
  })).finally(function () { clearTimeout(t); });
}
function apiFetch(url) { return fetchT(url, {}, 8000); }
function legacy(url) { return fetch(url); } // rc-scan: allow fetch-no-signal
