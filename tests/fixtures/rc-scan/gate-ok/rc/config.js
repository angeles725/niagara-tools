function lockForm() {
  saveBtn.classList.add("is-disabled");
  saveBtn.setAttribute("aria-disabled", "true");
  cancelBtn.disabled = false;
}
function requireLogin(ev) { if (!session.user) { openLogin(); ev.preventDefault(); } }
