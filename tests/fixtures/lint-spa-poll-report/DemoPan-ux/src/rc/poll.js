let lastOkAt = 0;
async function poll() {
  try {
    await load();
  } catch (err) {
    paint("error");
  }
  lastOkAt = Date.now();
  if (Date.now() - lastOkAt > 3e4) location.reload();
}
setTimeout(poll, 5000);
