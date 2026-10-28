/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.discover.BINDiscoveryObject;
import com.tridium.ndriver.discover.BNDiscoveryJob;
import com.tridium.ndriver.discover.BNDiscoveryPreferences;
import com.tridium.ndriver.point.BNPointDeviceExt;

import com.angeles.MinimalDriver.model.MinimalDriverCodec;
import com.angeles.MinimalDriver.model.MinimalDriverCodec.Frame;

/**
 * BMinimalDriverPointDeviceExt — the device's points extension and point discovery host.
 *
 * <p>Point discovery asks the device how many registers it has and offers one leaf per register.
 * The job is submitted by the inherited {@code doSubmitDiscoveryJob(prefs)}: Baja dispatches the
 * framework's own action to that one-argument do method, so a subclass overload taking a Context
 * would never be called; point discovery therefore runs with the framework's null Context
 * (ComplexIntrospector resolves {@code do<Action>(param)} before {@code do<Action>(param, Context)}).</p>
 */
@NiagaraType
@NiagaraProperty(
  name = "discoveryPreferences",
  type = "BNDiscoveryPreferences",
  defaultValue = "new BMinimalDriverPointDiscoveryPreferences()",
  override = true
)
public class BMinimalDriverPointDeviceExt
  extends BNPointDeviceExt
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  /** Upper bound on offered registers, so a corrupt count cannot flood the discovery table. */
  public static final int MAX_DISCOVERED_REGISTERS = 256;

  public BMinimalDriverPointDeviceExt() {}

  @Override
  public Type getDeviceType()
  {
    return BMinimalDriverDevice.TYPE;
  }

  @Override
  public Type getPointFolderType()
  {
    return BMinimalDriverPointFolder.TYPE;
  }

  @Override
  public Type getProxyExtType()
  {
    return BMinimalDriverProxyExt.TYPE;
  }

  @Override
  public BINDiscoveryObject[] getDiscoveryObjects(BNDiscoveryPreferences prefs)
    throws Exception
  {
    BNDiscoveryJob job = prefs.getJob();
    if (job == null || !job.isAlive()) return null;
    if (!(getDevice() instanceof BMinimalDriverDevice))
    {
      job.discoverFail("points extension is not under a MinimalDriver device");
      return null;
    }
    BMinimalDriverDevice device = (BMinimalDriverDevice)getDevice();
    if (!device.hasAddress() || device.isDown() || device.isDisabled())
    {
      job.discoverFail("device " + device.getName() + " is not reachable (address, down or disabled)");
      return null;
    }
    job.heartbeat();
    Frame count = device.minimalDriverNetwork().send(device.peer(), MinimalDriverCodec.CMD_COUNT, 0, 0);
    int n = Math.max(0, Math.min(count.value, MAX_DISCOVERED_REGISTERS));
    BINDiscoveryObject[] leaves = new BINDiscoveryObject[n];
    for (int reg = 0; reg < n; reg++)
    {
      if (!job.isAlive()) return null;
      leaves[reg] = new BMinimalDriverPointDiscoveryLeaf(reg);
    }
    job.log().message("device " + device.getName() + " reports " + count.value + " registers, offering " + n);
    job.progress(100);
    return leaves;
  }
}
