define(['baja!', 'bajaux/mixin/subscriberMixIn'], function (baja, subscriberMixIn) {
  return function View() {
    subscriberMixIn(this);
    this.getSubscriber = function () { return this.$sub || (this.$sub = new baja.Subscriber()); };
  };
});
