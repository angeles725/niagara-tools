'use strict';

function load() {
  var a = baja.Ord.make('alarm:|bql:select *'); var u = 'http://x'; // lint-bql-unbounded: a quoted URL before the marker
  var b = baja.Ord.make('alarm:|bql:select *'); var t = `it's http://x`; // lint-bql-unbounded: an apostrophe in a template
  var c = baja.Ord.make('alarm:|bql:select *'); var d = "it's"; // lint-bql-unbounded: an apostrophe in a double-quoted string
}
