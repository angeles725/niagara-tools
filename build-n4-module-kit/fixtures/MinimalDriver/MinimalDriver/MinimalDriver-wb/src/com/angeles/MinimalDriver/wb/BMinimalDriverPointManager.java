/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver.wb;

import javax.baja.nre.annotations.AgentOn;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.ui.point.BNPointManager;

/**
 * BMinimalDriverPointManager — the Workbench point manager for MinimalDriver points.
 *
 * <p>An agent on this driver's points extension and point folder that extends the NDriver point
 * manager replaces the generic one in the view list (AgentInfoUtil.getAgentsHelp), so the driver
 * ships its own -wb manager (types/driver-authoring.md § 1.3, § 1.6). Columns come from the proxy
 * extension's {@code SfUtil.incl(SfUtil.MGR_EDIT)} slots (register, pollFrequency) and discovery from
 * the points extension; override {@code makeModel()} here to add columns of your own.</p>
 */
@NiagaraType(agent = @AgentOn(types = { "MinimalDriver:MinimalDriverPointDeviceExt", "MinimalDriver:MinimalDriverPointFolder" }))
public class BMinimalDriverPointManager
  extends BNPointManager
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverPointManager() {}
}
