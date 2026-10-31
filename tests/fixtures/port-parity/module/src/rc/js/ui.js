function renderLevels(levels) {
  return levels.map(function (l) { return '<li>' + l + '</li>'; }).join('');
}

function levelTitle(level) {
  return level.name;
}
