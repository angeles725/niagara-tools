/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.registry.TypeInfo;
import javax.baja.sys.BComponent;
import javax.baja.sys.Flags;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.discover.BNDiscoveryLeaf;

/**
 * BMinimalDriverDeviceDiscoveryLeaf — one device that answered a discovery ping (§ 1.11 step 5).
 * The manager adds it as a {@link BMinimalDriverDevice} with this ipAddress and udpPort.
 */
@NiagaraType
@NiagaraProperty(
  name = "ipAddress",
  type = "String",
  defaultValue = "",
  flags = Flags.READONLY
)
@NiagaraProperty(
  name = "udpPort",
  type = "int",
  defaultValue = "0",
  flags = Flags.READONLY
)
@NiagaraProperty(
  name = "deviceId",
  type = "int",
  defaultValue = "0",
  flags = Flags.READONLY
)
public class BMinimalDriverDeviceDiscoveryLeaf
  extends BNDiscoveryLeaf
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverDeviceDiscoveryLeaf() {}

  public BMinimalDriverDeviceDiscoveryLeaf(String host, int port, int deviceId)
  {
    setIpAddress(host);
    setUdpPort(port);
    setDeviceId(deviceId);
  }

  @Override
  public TypeInfo[] getValidDatabaseTypes()
  {
    return new TypeInfo[] { BMinimalDriverDevice.TYPE.getTypeInfo() };
  }

  @Override
  public String getDiscoveryName()
  {
    return "Device" + getDeviceId();
  }

  @Override
  public void updateTarget(BComponent target)
  {
    if (!(target instanceof BMinimalDriverDevice)) return;
    BMinimalDriverDevice dev = (BMinimalDriverDevice)target;
    dev.setIpAddress(getIpAddress());
    dev.setUdpPort(getUdpPort());
  }

  /** A device already in the station at the same ipAddress and udpPort is greyed out. */
  @Override
  public boolean isExisting(BComponent target)
  {
    if (!(target instanceof BMinimalDriverDevice)) return false;
    BMinimalDriverDevice dev = (BMinimalDriverDevice)target;
    return dev.getIpAddress().trim().equals(getIpAddress()) && dev.getUdpPort() == getUdpPort();
  }
}
