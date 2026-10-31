function renderLevels(levels) {
  return levels.map(function (l) { return '<li>' + l + '</li>'; }).join('');
}

function renderEquipment(eq) {
  return '<div id="eqPanel">' + eq.name + '</div>';
}

const eqHistory = (eq) => eq.history || [];
