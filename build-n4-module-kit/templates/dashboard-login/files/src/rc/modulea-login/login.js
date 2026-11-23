/* ModuleA login page script (no inline handlers). Mirrors Niagara's digestLoginN4.js and resetN4.js:
   the password never posts in clear: SCRAM runs through /login/core/auth.min.js (ScramShaClient), then the
   browser goes to /j_security_check/ whatever the outcome (the server decides and redirects); a forced change
   sends only the base64 of the new password as resetToken. */
(function (win, doc) {
  'use strict';

  var RULES = [
    ['minimumLength', 'At least N characters', function (p) { return p.length; }],
    ['minimumUpperCase', 'At least N capital letter(s)', count(/[A-Z]/g)],
    ['minimumLowerCase', 'At least N lower-case letter(s)', count(/[a-z]/g)],
    ['minimumDigits', 'At least N digit(s)', count(/[0-9]/g)],
    ['minimumSpecial', 'At least N special character(s)', count(/[^A-Za-z0-9]/g)]
  ];

  function count(re) { return function (p) { return (p.match(re) || []).length; }; }

  /** The rule list for a password: [{text, met}], plus the maximum and the match with the confirmation. */
  function checkRules(rules, pw, confirm) {
    var out = [];
    RULES.forEach(function (r) {
      var n = rules && rules[r[0]];
      if (n > 0) out.push({ text: r[1].replace('N', n), met: r[2](pw) >= n });
    });
    if (rules && rules.maximumLength > 0) {
      out.push({ text: 'At most ' + rules.maximumLength + ' characters', met: pw.length <= rules.maximumLength });
    }
    out.push({ text: 'Both passwords match', met: pw.length > 0 && pw === confirm });
    return out;
  }

  function wireLogin() {
    var form = doc.getElementById('login-form');
    if (!form) return;
    form.addEventListener('submit', function (e) {
      e.preventDefault();
      var go = function () { win.location.replace('/j_security_check/'); };
      if (typeof win.ScramShaClient === 'undefined' || typeof win.sjcl === 'undefined') { go(); return; }
      doc.getElementById('login-submit').disabled = true;
      win.ScramShaClient.authenticate('/', doc.getElementById('username').value,
        doc.getElementById('password').value, win.sjcl.hash.sha256, { ok: go, fail: go });
    });
  }

  function wireReset() {
    var form = doc.getElementById('reset-form');
    if (!form) return;
    var rules = {};
    try { rules = JSON.parse(form.getAttribute('data-rules') || '{}'); } catch (ignore) { rules = {}; }
    var pw = doc.getElementById('new-password'), confirm = doc.getElementById('confirm-password');
    var list = doc.getElementById('rules'), submit = doc.getElementById('login-submit');
    var paint = function () {
      var state = checkRules(rules, pw.value, confirm.value);
      list.innerHTML = '';
      state.forEach(function (r) {
        var li = doc.createElement('li');
        li.className = r.met ? 'met' : '';
        li.textContent = r.text + (r.met ? ' (done)' : '');
        list.appendChild(li);
      });
      submit.disabled = !state.every(function (r) { return r.met; });
    };
    pw.addEventListener('input', paint);
    confirm.addEventListener('input', paint);
    paint();
    form.addEventListener('submit', function (e) {
      e.preventDefault();
      if (submit.disabled) return;
      var token = win.sjcl.codec.base64.fromBits(win.sjcl.codec.utf8String.toBits(pw.value));
      pw.value = '';
      confirm.value = '';
      doc.getElementById('resetToken').value = token;
      form.submit();
    });
  }

  wireLogin();
  wireReset();
})(window, document);
