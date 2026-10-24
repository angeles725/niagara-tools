/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.point.BNPointFolder;

/**
 * BMinimalDriverPointFolder — groups proxy points under a device's points extension.
 */
@NiagaraType
public class BMinimalDriverPointFolder
  extends BNPointFolder
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverPointFolder() {}
}
