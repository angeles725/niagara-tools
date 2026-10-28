/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.comm.IMessageFactory;
import com.tridium.ndriver.comm.NLinkMessageFactory;
import com.tridium.ndriver.datatypes.BUdpCommConfig;

import com.angeles.MinimalDriver.model.MinimalDriverCodec;

/**
 * BMinimalDriverUdpCommConfig — the network's UDP comm config, subclassed so the stack builds this
 * driver's messages.
 *
 * <p>A bare BUdpCommConfig keeps the base message factory, whose NMessage throws on every received
 * datagram (§ 1.9), so this subclass supplies both factories. Its {@code address} is the LOCAL bind
 * address (default {@code local:-1}: every interface, ephemeral port); each device's ipAddress and udpPort
 * travel on its requests. A TCP driver subclasses BTcpCommConfig the same way and keeps
 * {@code serverEnabled} at its false default (§ 1.6).</p>
 */
@NiagaraType
public class BMinimalDriverUdpCommConfig
  extends BUdpCommConfig
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverUdpCommConfig() {}

  @Override
  protected IMessageFactory makeMessageFactory()
  {
    return new MinimalDriverMessageFactory();
  }

  /** Receive buffers sized above one frame, so an oversized datagram is rejected, not cut (§ 1.9). */
  @Override
  protected NLinkMessageFactory makeLinkMessageFactory()
  {
    return new NLinkMessageFactory(MinimalDriverCodec.BUFFER_LEN);
  }
}
