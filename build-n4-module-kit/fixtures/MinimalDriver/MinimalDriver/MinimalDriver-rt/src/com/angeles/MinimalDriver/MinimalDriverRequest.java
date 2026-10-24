/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import java.io.OutputStream;

import com.tridium.ndriver.comm.NMessage;
import com.tridium.ndriver.datatypes.BIpAddress;

import com.angeles.MinimalDriver.MinimalDriverCodec.Frame;

/**
 * MinimalDriverRequest — one request on the wire. The byte layout lives in {@link MinimalDriverCodec};
 * this class adds what NComm needs: the destination, the tag, the timeout and the response check.
 *
 * <p>The peer address travels on each message, never on the comm config (§ 1.9). The tag is the
 * frame's sequence number, so several requests can be outstanding at once (§ 1.7).</p>
 */
public class MinimalDriverRequest
  extends NMessage
{
  /**
   * Response timeout in ms, on top of the framework's 1000 ms padding (§ 1.7). Set it from the real
   * device latency; one retry covers a lost datagram.
   */
  public static final int RESPONSE_TIMEOUT_MS = 1000;
  public static final int RETRY_COUNT = 1;

  public MinimalDriverRequest(BIpAddress to, Frame frame)
  {
    super(to);
    this.frame = frame;
    setResponseTimeOut(RESPONSE_TIMEOUT_MS);
    setRetryCount(RETRY_COUNT);
  }

  public Frame getFrame()
  {
    return frame;
  }

  @Override
  public boolean toOutputStream(OutputStream out)
    throws Exception
  {
    out.write(MinimalDriverCodec.encode(frame));
    return false;   // one datagram, no further fragments
  }

  /** The sequence number; the response returns the same key (§ 1.7). */
  @Override
  public Object getTag()
  {
    return frame.seq;
  }

  /**
   * Accept only a response from the device this request went to (UDP: any host can send a matching
   * tag, § 1.9) that answers this exact request and carries no device error.
   */
  @Override
  public int validateResponse(NMessage resp)
    throws Exception
  {
    if (!(resp instanceof MinimalDriverResponse)) return FAILED_RESPONSE;
    if (getAddress() == null || resp.getAddress() == null || !getAddress().sameDevice(resp.getAddress()))
      return FAILED_RESPONSE;
    Frame answer = ((MinimalDriverResponse)resp).getFrame();
    if (!MinimalDriverCodec.answers(frame, answer)) return FAILED_RESPONSE;
    return answer.isError() ? FAILED_RESPONSE : SUCCESS_RESPONSE;
  }

  @Override
  public String toTraceString()
  {
    return "MinimalDriver " + frame + " to " + getAddress();
  }

  @Override
  public String toString()
  {
    return toTraceString();
  }

  private final Frame frame;
}
