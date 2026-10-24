/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import com.tridium.ndriver.comm.NMessage;

import com.angeles.MinimalDriver.MinimalDriverCodec.Frame;

/**
 * MinimalDriverResponse — one decoded response frame. Built by {@link MinimalDriverMessageFactory}
 * on the comm receive thread; matched to its request by the same tag (§ 1.7, § 1.10).
 */
public class MinimalDriverResponse
  extends NMessage
{
  public MinimalDriverResponse(Frame frame)
  {
    this.frame = frame;
  }

  public Frame getFrame()
  {
    return frame;
  }

  @Override
  public boolean isResponse()
  {
    return true;
  }

  @Override
  public Object getTag()
  {
    return frame.seq;
  }

  @Override
  public String toTraceString()
  {
    return "MinimalDriver " + frame + " from " + getAddress();
  }

  @Override
  public String toString()
  {
    return toTraceString();
  }

  private final Frame frame;
}
