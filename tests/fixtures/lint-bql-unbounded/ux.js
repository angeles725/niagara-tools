'use strict';

// search the station
function load(cx) {
  var all = baja.Ord.make('station:|slot:/|bql:select * from baja:Component');
  var top = baja.Ord.make('station:|slot:/|bql:select top 50 name from baja:Component');
  var hist = baja.Ord.make("history:/S/H|bql:select timestamp, value");
  var tpl = baja.Ord.make(`alarm:|bql:select * from openAlarms`);
  var dyn = baja.Ord.make(`alarm:|bql:select * where uuid = '${id}'`);
  var cat = baja.Ord.make('station:|slot:/|bql:select * where name = ' + q);
  // lint-bql-unbounded: operator-triggered, the row count is shown first
  var waived = baja.Ord.make('alarm:|bql:select *');
  var s = "http://example.com/a"; // a // in a string is no comment
}
