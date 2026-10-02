function lockForm() {
  // the save button opens the login gate on click
  saveBtn.disabled = true;
}
function requireLogin(ev) { if (!session.user) { openLogin(); ev.preventDefault(); } }
