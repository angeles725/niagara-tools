/*
 * MinimalDashContractSpec.js — the consumer side of the ux-rt API contract (types/dashboard.md DUX-CONTRACT1).
 *
 * Every field services/api.js says the front end reads (MinimalDashApi.reads) must be declared for its route in
 * src/rc/js/contract.json. The servlet side is StatusHandlerTest (srcTest/test/.../ux/route), over the same
 * file. Offline, no browser: node srcTest/rc/spec/MinimalDashContractSpec.js (exit 0 pass, 1 a missing field).
 * Not a Jasmine/Karma spec: the spec aggregator does not list it.
 */
var fs = require('fs');
var path = require('path');
var ux = path.join(__dirname, '..', '..', '..');
var contract = JSON.parse(fs.readFileSync(path.join(ux, 'src', 'rc', 'js', 'contract.json'), 'utf8'));
var api = require(path.join(ux, 'src', 'rc', 'js', 'services', 'api.js'));
var problems = [];
Object.keys(api.reads).forEach(function (route) {
  var c = contract.routes[route];
  if (!c) { problems.push('route ' + route + ' is not in contract.json'); return; }
  api.reads[route].forEach(function (field) {
    if (!Object.prototype.hasOwnProperty.call(c.fields, field)) {
      problems.push(route + ': field ' + field + ' read by the front end is not in contract.json');
    }
  });
});
if (problems.length) {
  console.error(problems.join('\n'));
  process.exit(1);
}
console.log('contract: ' + Object.keys(api.reads).length + ' route(s) OK');
