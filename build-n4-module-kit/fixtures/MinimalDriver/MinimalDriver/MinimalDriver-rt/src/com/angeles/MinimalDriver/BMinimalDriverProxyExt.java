/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.driver.point.BReadWriteMode;
import javax.baja.driver.util.BPollFrequency;
import javax.baja.nre.annotations.Facet;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.status.BStatusNumeric;
import javax.baja.status.BStatusValue;
import javax.baja.sys.Context;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.comm.NCommTimeoutException;
import com.tridium.ndriver.point.BNProxyExt;
import com.tridium.ndriver.poll.BINPollable;
import com.tridium.ndriver.util.SfUtil;

import com.angeles.MinimalDriver.MinimalDriverCodec.Frame;

/**
 * BMinimalDriverProxyExt — proxies one device register into a numeric control point.
 *
 * <p>Reads are polled while the point is subscribed (§ 2, § 6); {@link #doPoll()} runs on the
 * network's poll thread. Writes are posted to the network's worker and {@link #write} returns true
 * (pending) at once, never blocking the engine thread (§ 1.7, § 2). A poll timeout fails the point and
 * asks the device for a fresh ping, because a failed poll is not a failed ping (§ 5).</p>
 */
@NiagaraType
@NiagaraProperty(
  name = "register",
  type = "int",
  defaultValue = "0",
  facets = @Facet("SfUtil.incl(SfUtil.MGR_EDIT)")
)
@NiagaraProperty(
  name = "pollFrequency",
  type = "BPollFrequency",
  defaultValue = "BPollFrequency.normal",
  facets = @Facet("SfUtil.incl(SfUtil.MGR_EDIT)")
)
public class BMinimalDriverProxyExt
  extends BNProxyExt
  implements BINPollable
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverProxyExt() {}

  @Override
  public Type getDeviceExtType()
  {
    return BMinimalDriverPointDeviceExt.TYPE;
  }

  @Override
  public BReadWriteMode getMode()
  {
    return BReadWriteMode.readWrite;
  }

  ////////////////////////////////////////////////////////////////
  // Read
  ////////////////////////////////////////////////////////////////

  @Override
  public void readSubscribed(Context cx)
    throws Exception
  {
    BMinimalDriverNetwork net = network();
    if (net != null) net.getPollScheduler().subscribe(this);
  }

  @Override
  public void readUnsubscribed(Context cx)
    throws Exception
  {
    BMinimalDriverNetwork net = network();
    if (net != null) net.getPollScheduler().unsubscribe(this);
  }

  /** Poll thread: read the register; every failure is reported on the point, never swallowed. */
  @Override
  public void doPoll()
  {
    BMinimalDriverDevice dev = device();
    if (dev == null) { readFail("not under a MinimalDriver device"); return; }
    if (!(getParentPoint().getOutStatusValue() instanceof BStatusNumeric))
    {
      readFail("MinimalDriver proxies numeric points only");
      return;
    }
    if (!dev.hasAddress()) { readFail("device ipAddress and udpPort are not set"); return; }
    try
    {
      Frame answer = dev.minimalDriverNetwork().send(dev.peer(), MinimalDriverCodec.CMD_READ, getRegister(), 0);
      readOk(new BStatusNumeric(answer.value));
    }
    catch (NCommTimeoutException e)
    {
      readFail("no answer from " + dev.peer());
      dev.ping();
    }
    catch (Exception e)
    {
      readFail(String.valueOf(e.getMessage() != null ? e.getMessage() : e));
    }
  }

  ////////////////////////////////////////////////////////////////
  // Write
  ////////////////////////////////////////////////////////////////

  /**
   * Engine thread: validate, post the send to the network's worker and return true (pending); the
   * worker calls writeOk or writeFail. A value that cannot be sent fails at once and returns false.
   */
  @Override
  public boolean write(Context cx)
    throws Exception
  {
    final BMinimalDriverDevice dev = device();
    final BStatusValue wv = getWriteValue();
    if (dev == null || !dev.hasAddress())
    {
      writeFail("device ipAddress and udpPort are not set");
      return false;
    }
    if (!(wv instanceof BStatusNumeric))
    {
      writeFail("MinimalDriver writes numeric values only");
      return false;
    }
    final int value;
    try
    {
      value = MinimalDriverCodec.toWireValue(((BStatusNumeric)wv).getValue());
    }
    catch (IllegalArgumentException e)
    {
      writeFail(e.getMessage());
      return false;
    }
    final int register = getRegister();
    dev.postAsync(new Runnable()
    {
      @Override
      public void run()
      {
        try
        {
          dev.minimalDriverNetwork().send(dev.peer(), MinimalDriverCodec.CMD_WRITE, register, value);
          writeOk(wv);
        }
        catch (Exception e)
        {
          writeFail(String.valueOf(e.getMessage() != null ? e.getMessage() : e));
        }
      }
    });
    return true;
  }

  ////////////////////////////////////////////////////////////////
  // Parents (every cast guarded, § 1.7)
  ////////////////////////////////////////////////////////////////

  private BMinimalDriverDevice device()
  {
    return getDevice() instanceof BMinimalDriverDevice ? (BMinimalDriverDevice)getDevice() : null;
  }

  private BMinimalDriverNetwork network()
  {
    return getNetwork() instanceof BMinimalDriverNetwork ? (BMinimalDriverNetwork)getNetwork() : null;
  }
}
