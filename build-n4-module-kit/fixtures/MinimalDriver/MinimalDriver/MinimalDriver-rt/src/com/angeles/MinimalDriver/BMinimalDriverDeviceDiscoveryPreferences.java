/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.discover.BNDiscoveryPreferences;

/**
 * BMinimalDriverDeviceDiscoveryPreferences — what a device scan asks for: a comma-separated host list
 * (IP literals preferred) and the protocol port. {@code doNotAskAgain} is false so the dialog opens.
 */
@NiagaraType
@NiagaraProperty(
  name = "doNotAskAgain",
  type = "boolean",
  defaultValue = "false",
  override = true
)
@NiagaraProperty(
  name = "hosts",
  type = "String",
  defaultValue = ""
)
@NiagaraProperty(
  name = "udpPort",
  type = "int",
  defaultValue = "0"
)
public class BMinimalDriverDeviceDiscoveryPreferences
  extends BNDiscoveryPreferences
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverDeviceDiscoveryPreferences() {}

  @Override
  public Type getDiscoveryLeafType()
  {
    return BMinimalDriverDeviceDiscoveryLeaf.TYPE;
  }
}
