function simTick(state) {
  return state + 1;
}

const eqHistory = (eq) => eq.simulated || [];
const simPanel = '<div id="simPanel"></div>';
