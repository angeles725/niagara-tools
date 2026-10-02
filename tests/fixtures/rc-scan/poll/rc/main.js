async function poll() { await fetchT("/api/data", {}, 8000); }
setInterval(poll, 5000);
setInterval(async function () { await poll(); }, 5000);
const refresh = async () => { await poll(); };
setInterval(refresh, 10000);
