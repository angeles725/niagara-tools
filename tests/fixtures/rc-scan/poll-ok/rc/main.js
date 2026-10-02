async function poll() {
  try { await fetchT("/api/data", {}, 8000); } finally { setTimeout(poll, 5000); }
}
function tick() { clock.textContent = new Date().toISOString(); }
setInterval(tick, 1000);
