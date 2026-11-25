'use strict';
require(['baja!'], function (baja) {
  var sub = new baja.Subscriber();
  sub.attach('changed', function () {});
  baja.Ord.make('station:|slot:/Ramp').get({ subscriber: sub });
});
