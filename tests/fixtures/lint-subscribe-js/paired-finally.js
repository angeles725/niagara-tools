// The getting-started tutorial: unsubscribe and detach when the dialog closes.
require(['baja!', 'dialogs'], function (baja, dialogs) {
  var sub = new baja.Subscriber();
  dialogs.showOk(function () {
    baja.Ord.make('station:|slot:/Ramp').get({ subscriber: sub });
  })
  .promise()
  .finally(function () {
    sub.unsubscribeAll();
    sub.detach();
  });
});
