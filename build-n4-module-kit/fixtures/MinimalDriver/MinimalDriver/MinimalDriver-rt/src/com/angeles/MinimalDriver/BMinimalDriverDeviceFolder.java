/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.BNDeviceFolder;

/**
 * BMinimalDriverDeviceFolder — groups devices under the network; the device manager works in it.
 */
@NiagaraType
public class BMinimalDriverDeviceFolder
  extends BNDeviceFolder
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverDeviceFolder() {}
}
