/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

import javax.baja.naming.BOrd;
import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Context;
import javax.baja.sys.Flags;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.BNNetwork;
import com.tridium.ndriver.comm.NComm;
import com.tridium.ndriver.comm.NMessage;
import com.tridium.ndriver.datatypes.BIpAddress;
import com.tridium.ndriver.discover.BINDiscoveryHost;
import com.tridium.ndriver.discover.BINDiscoveryObject;
import com.tridium.ndriver.discover.BNDiscoveryJob;
import com.tridium.ndriver.discover.BNDiscoveryPreferences;
import com.tridium.ndriver.poll.BNPollScheduler;

import com.angeles.MinimalDriver.MinimalDriverCodec.Frame;

/**
 * BMinimalDriverNetwork — the NDriver network: owns the UDP comm stack, the poll scheduler and
 * device discovery. [ev: kit types/driver-authoring.md § 1.6-§ 1.11]
 *
 * <p>All device I/O goes through {@link #send}, which blocks for the matched response; call it only
 * from the network's worker, the poll thread or a discovery job, never the engine thread (§ 1.7).</p>
 */
@NiagaraType
@NiagaraProperty(
  name = "pollScheduler",
  type = "BNPollScheduler",
  defaultValue = "new BNPollScheduler()"
)
@NiagaraProperty(
  name = "udpConfig",
  type = "BMinimalDriverUdpCommConfig",
  defaultValue = "new BMinimalDriverUdpCommConfig()"
)
@NiagaraProperty(
  name = "discoveryPreferences",
  type = "BNDiscoveryPreferences",
  defaultValue = "new BMinimalDriverDeviceDiscoveryPreferences()",
  flags = Flags.HIDDEN
)
@NiagaraAction(
  name = "submitDiscoveryJob",
  parameterType = "BNDiscoveryPreferences",
  defaultValue = "new BMinimalDriverDeviceDiscoveryPreferences()",
  returnType = "BOrd",
  flags = Flags.HIDDEN
)
public class BMinimalDriverNetwork
  extends BNNetwork
  implements BINDiscoveryHost
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverNetwork() {}

  @Override
  public String getNetworkName()
  {
    return "MinimalDriverNetwork";
  }

  @Override
  public Type getDeviceType()
  {
    return BMinimalDriverDevice.TYPE;
  }

  @Override
  public Type getDeviceFolderType()
  {
    return BMinimalDriverDeviceFolder.TYPE;
  }

  /** Keep the comm stack's state in step with the network's (enabled, fault). */
  @Override
  public void changed(Property p, Context cx)
  {
    super.changed(p, cx);
    if (isRunning() && p == status) getUdpConfig().statusUpdate();
  }

  ////////////////////////////////////////////////////////////////
  // Comm
  ////////////////////////////////////////////////////////////////

  /** The UDP stack; {@code comm()} starts it on first use. */
  public NComm comm()
  {
    return (NComm)getUdpConfig().comm();
  }

  /**
   * Send one request to {@code to} and return the device's answer. Blocks until the matched response,
   * the timeout or the retries run out.
   *
   * @throws Exception NCommTimeoutException on no answer, NCommException on a failed response
   *   (wrong sender, wrong correlation or a device error), IllegalStateException while the stack stops
   */
  public Frame send(BIpAddress to, int cmd, int register, int value)
    throws Exception
  {
    Frame req = MinimalDriverCodec.request(MinimalDriverCodec.nextSeq(seq.getAndIncrement()), cmd, register, value);
    NMessage resp = comm().sendRequest(new MinimalDriverRequest(to, req));
    if (!(resp instanceof MinimalDriverResponse))
      throw new IllegalStateException("comm stopped before " + to + " answered");
    return ((MinimalDriverResponse)resp).getFrame();
  }

  ////////////////////////////////////////////////////////////////
  // Discovery (§ 1.11)
  ////////////////////////////////////////////////////////////////

  /**
   * Runs as the user who asked: the job is submitted with the action's Context, never null (§ 1.6).
   * This network declares the action itself, so the Context form is the do method Baja dispatches to.
   */
  public BOrd doSubmitDiscoveryJob(BNDiscoveryPreferences preferences, Context cx)
  {
    setDiscoveryPreferences((BNDiscoveryPreferences)preferences.newCopy());
    BNDiscoveryJob job = new BNDiscoveryJob(this);
    job.setDiscoveryPreferences((BNDiscoveryPreferences)preferences.newCopy());
    return job.submit(cx);
  }

  /**
   * Ping every host in the preferences on the preferences' port; each one that answers becomes a
   * discovery leaf. Runs on the job thread: stops as soon as the job is no longer alive, reports
   * progress per host and logs each result to the job log.
   */
  @Override
  public BINDiscoveryObject[] getDiscoveryObjects(BNDiscoveryPreferences prefs)
    throws Exception
  {
    BNDiscoveryJob job = prefs.getJob();
    if (job == null || !job.isAlive()) return null;
    if (!(prefs instanceof BMinimalDriverDeviceDiscoveryPreferences))
    {
      job.discoverFail("unexpected discovery preferences " + prefs.getType());
      return null;
    }
    BMinimalDriverDeviceDiscoveryPreferences p = (BMinimalDriverDeviceDiscoveryPreferences)prefs;
    String[] hosts = MinimalDriverCodec.splitHosts(p.getHosts());
    if (hosts.length == 0 || p.getUdpPort() <= 0)
    {
      job.discoverFail("set the hosts list and a port above 0 in the discovery preferences");
      return null;
    }

    List<BINDiscoveryObject> found = new ArrayList<>();
    for (int i = 0; i < hosts.length; i++)
    {
      if (!job.isAlive()) return null;
      job.progress(MinimalDriverCodec.percent(i, hosts.length));
      try
      {
        Frame answer = send(new BIpAddress(hosts[i], p.getUdpPort()), MinimalDriverCodec.CMD_PING, 0, 0);
        found.add(new BMinimalDriverDeviceDiscoveryLeaf(hosts[i], p.getUdpPort(), answer.value));
        job.log().message("found device " + answer.value + " at " + hosts[i] + ":" + p.getUdpPort());
      }
      catch (Exception e)
      {
        job.log().message("no answer from " + hosts[i] + ":" + p.getUdpPort() + ": " + e);
      }
    }
    job.progress(100);
    return found.toArray(new BINDiscoveryObject[0]);
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  /** One sequence counter per network: requests to different devices share the one UDP stack. */
  private final AtomicInteger seq = new AtomicInteger();
}
