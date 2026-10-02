// rows-formatter.cjs — ESLint formatter that prints kit rows (one per message), so
// report-module.sh relays ESLint like every other toolbelt member.
//   FAIL|WARN  eslint  <file>:<line>  <rule>: <message>
// severity 2 (error, incl. a parse error above ecmaVersion 2020) -> FAIL; 1 (warning) -> WARN.
// [ev: retro dashboard-frontend-standard Δ10]
'use strict';
const path = require('path');

module.exports = function rowsFormatter(results) {
  const rows = [];
  for (const r of results) {
    const file = path.relative(process.cwd(), r.filePath) || r.filePath;
    for (const m of r.messages) {
      const st = m.severity === 2 ? 'FAIL' : 'WARN';
      const rule = m.ruleId || (m.fatal ? 'parse' : 'eslint');
      const msg = String(m.message).replace(/\s+/g, ' ').trim();
      rows.push(`${st}  eslint  ${file}:${m.line || 0}  ${rule}: ${msg}`);
    }
  }
  return rows.length ? rows.join('\n') + '\n' : '';
};
