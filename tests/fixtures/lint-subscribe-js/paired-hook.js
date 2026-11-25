define(['baja!', 'bajaux/Widget'], function (baja, Widget) {
  function Live() { Widget.apply(this, arguments); }
  Live.prototype.doInitialize = function () {
    this.$sub = new baja.Subscriber();
    return this.$sub.subscribe(this.value());
  };
  Live.prototype.doDestroy = function () {
    return this.$sub.unsubscribeAll();
  };
  return Live;
});
