var a = new baja.Subscriber(); var u = 'http://x'; // lint-subscribe-js: a quoted URL before the marker
var b = new baja.Subscriber(); var t = `it's http://x`; // lint-subscribe-js: an apostrophe in a template
var c = new baja.Subscriber(); var d = "it's"; // lint-subscribe-js: an apostrophe in a double-quoted string
