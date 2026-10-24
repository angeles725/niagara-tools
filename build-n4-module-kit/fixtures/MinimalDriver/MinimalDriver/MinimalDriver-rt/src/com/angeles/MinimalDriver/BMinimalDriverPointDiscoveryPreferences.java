/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.discover.BNDiscoveryPreferences;

/**
 * BMinimalDriverPointDiscoveryPreferences — point discovery needs no input (the device reports its
 * register count); this type only names the leaf type the point manager shows.
 */
@NiagaraType
public class BMinimalDriverPointDiscoveryPreferences
  extends BNDiscoveryPreferences
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverPointDiscoveryPreferences() {}

  @Override
  public Type getDiscoveryLeafType()
  {
    return BMinimalDriverPointDiscoveryLeaf.TYPE;
  }
}
