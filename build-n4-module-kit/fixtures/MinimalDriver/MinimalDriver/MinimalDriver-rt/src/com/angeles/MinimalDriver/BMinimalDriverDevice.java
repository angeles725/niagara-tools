/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.Facet;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.BNDevice;
import com.tridium.ndriver.datatypes.BIpAddress;
import com.tridium.ndriver.util.SfUtil;

import com.angeles.MinimalDriver.MinimalDriverCodec.Frame;

/**
 * BMinimalDriverDevice — one device on the network, addressed by its own ipAddress and udpPort.
 *
 * <p>{@link #doPing()} is a real protocol ping: it sends a PING request and only an answer from this
 * device makes it UP; no answer, a wrong sender or a device error makes it DOWN, and DOWN reaches
 * every proxy point under it (§ 5). It runs on the network's worker thread (BNDevice.postPing), so
 * the blocking send is allowed there.</p>
 *
 * <p>Prefer an IP literal for {@code ipAddress}: a host name is resolved on every send (§ 1.9, § 5).</p>
 */
@NiagaraType
@NiagaraProperty(
  name = "ipAddress",
  type = "String",
  defaultValue = "",
  facets = @Facet("SfUtil.incl(SfUtil.MGR_EDIT)")
)
@NiagaraProperty(
  name = "udpPort",
  type = "int",
  defaultValue = "0",
  facets = @Facet("SfUtil.incl(SfUtil.MGR_EDIT)")
)
@NiagaraProperty(
  name = "points",
  type = "BMinimalDriverPointDeviceExt",
  defaultValue = "new BMinimalDriverPointDeviceExt()"
)
public class BMinimalDriverDevice
  extends BNDevice
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverDevice() {}

  @Override
  public Type getNetworkType()
  {
    return BMinimalDriverNetwork.TYPE;
  }

  @Override
  public void doPing()
    throws Exception
  {
    if (!hasAddress())
    {
      pingFail("ipAddress and udpPort are not set");
      return;
    }
    try
    {
      Frame answer = minimalDriverNetwork().send(peer(), MinimalDriverCodec.CMD_PING, 0, 0);
      if (answer.isError()) pingFail("device error " + answer.status);
      else pingOk();
    }
    catch (Exception e)
    {
      pingFail(String.valueOf(e.getMessage() != null ? e.getMessage() : e));
    }
  }

  ////////////////////////////////////////////////////////////////
  // Address
  ////////////////////////////////////////////////////////////////

  public boolean hasAddress()
  {
    return !getIpAddress().trim().isEmpty() && getUdpPort() > 0 && getUdpPort() <= 0xFFFF;
  }

  /** The peer address set on every request (§ 1.9); built fresh so a host/port edit applies at once. */
  public BIpAddress peer()
  {
    return new BIpAddress(getIpAddress().trim(), getUdpPort());
  }

  public BMinimalDriverNetwork minimalDriverNetwork()
  {
    return (BMinimalDriverNetwork)getNetwork();
  }
}
