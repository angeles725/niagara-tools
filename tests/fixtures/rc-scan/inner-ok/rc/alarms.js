function renderAlarm(row, a, cell, box) {
  row.innerHTML = "<td>" + esc(a.message) + "</td>";
  cell.textContent = a.message;
  box.innerHTML = "<b>" + "static" + "</b>";
}
