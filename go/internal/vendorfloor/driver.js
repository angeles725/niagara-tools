'use strict';
// lint-vendor-floor acorn driver: embedded in go/internal/vendorfloor and run as
//   node - <request-json> <file>...
// It only parses and tokenizes. Every verdict, message and exit code belongs to the Go command.
//
// Request (argv[2]): {"acorn": <require() spec>, "ecmaVersion": <floor>, "names": [<identifier>...]}
// Output: one JSON record per line on stdout; the last one is "unavailable", "fault" or "done".
//   {"type":"unavailable"}          acorn could not be loaded
//   {"type":"file","index":<n>,     one per file, in argument order
//    "syntax":null | {"line":<n|0>,"message":<acorn text>,"module":<parses as an ES module>},
//    "names":[{"name","prev","prev2","next","line"}...]}   each requested name token with its neighbours
//   {"type":"fault","message":<m>}  an exception ended the run (an unreadable file, a bad request)
//   {"type":"done"}                 every file was reported
const fs = require('fs');

const [requestArg, ...files] = process.argv.slice(2);
const emit = (record) => process.stdout.write(JSON.stringify(record) + '\n');

// The parse error of src at ecmaVersion/sourceType, or null when it parses.
function parseError(acorn, src, ecmaVersion, sourceType) {
  try {
    acorn.parse(src, { ecmaVersion, sourceType, allowHashBang: true, locations: true });
    return null;
  } catch (e) {
    return e;
  }
}

// Why src is not a classic script at the floor, or null when it is one.
function syntaxOf(acorn, src, ecmaVersion) {
  const err = parseError(acorn, src, ecmaVersion, 'script');
  if (!err) return null;
  return {
    line: err.loc ? err.loc.line : 0,
    message: err.message,
    module: parseError(acorn, src, ecmaVersion, 'module') === null,
  };
}

// The tokens of src at 'latest' for one sourceType, kept up to the first tokenizer error.
function tokenRun(acorn, src, sourceType) {
  const out = [];
  try {
    for (const t of acorn.tokenizer(src, { ecmaVersion: 'latest', locations: true,
      allowHashBang: true, sourceType })) out.push(t);
    return { out, done: true };
  } catch (e) {
    return { out, done: false };
  }
}

// Tokens as a classic (sloppy) script first, the floor standard for vendored libraries, and as a module
// only when the script tokenizer stops (an ES-module-only library). The longer run wins, so a
// sloppy-only token (legacy octal, `with`) does not end the scan early.
// Mutation: VF9 -- tokenize as a module only so a sloppy script's API calls after a legacy octal vanish
function tokens(acorn, src) {
  const script = tokenRun(acorn, src, 'script');
  if (script.done) return script.out;
  const mod = tokenRun(acorn, src, 'module');
  return (mod.done || mod.out.length > script.out.length) ? mod.out : script.out;
}

function tokText(t) {
  if (t.type.label === 'name') return t.value;
  if (t.type.keyword) return t.type.keyword;
  return t.type.label;
}

// The name tokens in wanted, each with the text of the two tokens before it and the one after.
// Tokens, not text: a name inside a comment or a string is not a name token.
function namesOf(tk, wanted) {
  const out = [];
  for (let i = 0; i < tk.length; i++) {
    const t = tk[i];
    if (t.type.label !== 'name' || !wanted.has(t.value)) continue;
    out.push({
      name: t.value,
      prev: i > 0 ? tokText(tk[i - 1]) : '',
      prev2: i > 1 ? tokText(tk[i - 2]) : '',
      next: i + 1 < tk.length ? tokText(tk[i + 1]) : '',
      line: t.loc.start.line,
    });
  }
  return out;
}

function main() {
  const request = JSON.parse(requestArg);
  let acorn;
  try {
    acorn = require(request.acorn);
  } catch (e) {
    emit({ type: 'unavailable' });
    return;
  }
  const wanted = new Set(request.names);
  files.forEach((file, index) => {
    const src = fs.readFileSync(file, 'utf8');
    emit({
      type: 'file',
      index,
      syntax: syntaxOf(acorn, src, request.ecmaVersion),
      names: namesOf(tokens(acorn, src), wanted),
    });
  });
  emit({ type: 'done' });
}

try {
  main();
} catch (e) {
  emit({ type: 'fault', message: e.message });
}
