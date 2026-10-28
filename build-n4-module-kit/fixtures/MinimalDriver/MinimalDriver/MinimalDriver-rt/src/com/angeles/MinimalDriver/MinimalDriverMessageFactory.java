/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import java.io.InputStream;
import java.util.logging.Level;
import java.util.logging.Logger;

import com.tridium.ndriver.comm.IMessageFactory;
import com.tridium.ndriver.comm.LinkMessage;
import com.tridium.ndriver.comm.NMessage;
import com.tridium.ndriver.datatypes.BAddress;

import com.angeles.MinimalDriver.model.MinimalDriverCodec;
import com.angeles.MinimalDriver.model.MinimalDriverCodec.Frame;

/**
 * MinimalDriverMessageFactory — turns one received datagram into a {@link MinimalDriverResponse}.
 *
 * <p>Without this factory the comm config keeps the base one, whose plain NMessage throws on every
 * received frame (§ 1.9). A frame that is not a valid response is dropped by returning null (NComm
 * ignores it) and logged at FINE, never INFO (§ 1.7). Runs on the comm receive thread.</p>
 */
public class MinimalDriverMessageFactory
  implements IMessageFactory
{
  @Override
  public NMessage makeMessage(LinkMessage lm)
    throws Exception
  {
    byte[] buf = new byte[MinimalDriverCodec.BUFFER_LEN];
    int len = readAll(lm.getInputStream(), buf);
    Frame frame;
    try
    {
      frame = MinimalDriverCodec.decode(buf, 0, len);
    }
    catch (IllegalArgumentException e)
    {
      if (LOG.isLoggable(Level.FINE)) LOG.fine("dropped frame from " + lm.address + ": " + e.getMessage());
      return null;
    }
    if (!frame.isResponse())
    {
      if (LOG.isLoggable(Level.FINE)) LOG.fine("dropped unsolicited request frame from " + lm.address);
      return null;
    }
    MinimalDriverResponse resp = new MinimalDriverResponse(frame);
    if (lm.address instanceof BAddress) resp.setAddress((BAddress)lm.address);
    return resp;
  }

  /** Read until end of stream or a full buffer; a longer datagram shows up as a too-long frame. */
  private static int readAll(InputStream in, byte[] buf)
    throws Exception
  {
    int n = 0;
    while (n < buf.length)
    {
      int r = in.read(buf, n, buf.length - n);
      if (r < 0) break;
      n += r;
    }
    return n;
  }

  private static final Logger LOG = Logger.getLogger("MinimalDriver.comm");
}
