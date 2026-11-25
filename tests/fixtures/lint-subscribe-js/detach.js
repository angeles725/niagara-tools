function open(baja) {
  var sub = new baja.Subscriber();
  sub.attach('changed', onChange);
  return function close() { sub.detach(); };
}
