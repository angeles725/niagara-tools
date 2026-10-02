function tidy(s, o, arr) {
  var a = s.replaceAll("-", "_");
  var b = Object.hasOwn(o, "k");
  var c = arr.findLast(function (x) { return x > 1; });
  return [a, b, c];
}
