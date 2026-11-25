var a = new baja.Subscriber(); // lint-subscribe-js: torn down by the page manager in Page.js
// lint-subscribe-js: handed to the shared list, which unsubscribes it
var b = new baja.Subscriber();
var c = new baja.Subscriber();
var d = 1; // lint-subscribe-js: an unrelated reason on a code line
var e = new baja.Subscriber();
