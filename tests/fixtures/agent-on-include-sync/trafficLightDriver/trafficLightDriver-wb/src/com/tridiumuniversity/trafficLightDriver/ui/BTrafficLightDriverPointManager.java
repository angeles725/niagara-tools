// Fixture for tests/lint-agent-on-include-sync.bats: the course Bonus Lab point manager's package, @NiagaraType
// annotation and class declaration as the reference file declares them (sources/n4-dev-course course/Labs/Bonus
// Labs/Driver Discovery and UI/Customizing Point Manager); its body is not reproduced (this repository is public).
package com.tridiumuniversity.trafficLightDriver.ui;

import javax.baja.nre.annotations.*;

import com.tridium.ndriver.ui.point.BNPointManager;

@NiagaraType(
  agent = @AgentOn(types = {
    "trafficLightDriver:TrafficLightDriverPointDeviceExt",
    "trafficLightDriver:TrafficLightDriverPointFolder"
  })
)
public class BTrafficLightDriverPointManager
  extends BNPointManager
{
}
