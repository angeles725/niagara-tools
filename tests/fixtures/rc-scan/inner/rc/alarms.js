function renderAlarm(row, a, list) {
  row.innerHTML = "<td>" + a.message + "</td>";
  list.insertAdjacentHTML("beforeend", `<li>${a.sourceLabel}</li>`);
}
