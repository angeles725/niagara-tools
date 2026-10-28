/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver.model;

import org.junit.Test;
import static org.junit.Assert.*;

import com.angeles.MinimalDriver.model.MinimalDriverCodec.Frame;

/**
 * Pure JUnit test for {@link MinimalDriverCodec}: zero Baja, runs in WSL with
 * {@code toolbelt/run-pure-test.sh <MinimalDriver-rt> com.angeles.MinimalDriver.model.MinimalDriverCodecTest}.
 *
 * <p>Feeds round trips plus truncated, oversized, corrupted and foreign frames, and pins the
 * correlation rule. Each rejection test fails if its check is dropped from the codec.
 * [ev: kit types/driver-authoring.md § 12]</p>
 */
public class MinimalDriverCodecTest
{
  private static byte[] responseBytes(int seq, int cmd, int status, int register, int value)
  {
    return MinimalDriverCodec.encode(new Frame(seq, cmd | MinimalDriverCodec.RESPONSE_BIT, status, register, value));
  }

  private static String reason(byte[] buf, int off, int len)
  {
    try
    {
      MinimalDriverCodec.decode(buf, off, len);
      return null;
    }
    catch (IllegalArgumentException e)
    {
      return e.getMessage();
    }
  }

  @Test
  public void requestRoundTripsThroughTheWire()
  {
    Frame req = MinimalDriverCodec.request(7, MinimalDriverCodec.CMD_WRITE, 0x1234, -5);
    byte[] wire = MinimalDriverCodec.encode(req);
    assertEquals(MinimalDriverCodec.FRAME_LEN, wire.length);
    Frame back = MinimalDriverCodec.decode(wire, 0, wire.length);
    assertEquals(7, back.seq);
    assertEquals(MinimalDriverCodec.CMD_WRITE, back.command());
    assertFalse(back.isResponse());
    assertEquals(0x1234, back.register);
    assertEquals(-5, back.value);
  }

  @Test
  public void responseCarriesTheResponseBitAndStatus()
  {
    byte[] wire = responseBytes(200, MinimalDriverCodec.CMD_READ, 3, 9, 42);
    Frame f = MinimalDriverCodec.decode(wire, 0, wire.length);
    assertTrue(f.isResponse());
    assertTrue("a non-zero status is a device error", f.isError());
    assertEquals(200, f.seq);
    assertEquals(MinimalDriverCodec.CMD_READ, f.command());
  }

  @Test
  public void decodesAFrameAtAnOffset()
  {
    byte[] wire = responseBytes(1, MinimalDriverCodec.CMD_PING, 0, 0, 77);
    byte[] buf = new byte[MinimalDriverCodec.BUFFER_LEN];
    System.arraycopy(wire, 0, buf, 5, wire.length);
    assertEquals(77, MinimalDriverCodec.decode(buf, 5, wire.length).value);
  }

  @Test
  public void truncatedFrameIsRejected()
  {
    byte[] wire = responseBytes(1, MinimalDriverCodec.CMD_READ, 0, 1, 1);
    assertEquals("frame length 10, expected 11", reason(wire, 0, wire.length - 1));
  }

  @Test
  public void oversizedDatagramIsRejected()
  {
    byte[] buf = new byte[MinimalDriverCodec.BUFFER_LEN];
    System.arraycopy(responseBytes(1, MinimalDriverCodec.CMD_READ, 0, 1, 1), 0, buf, 0, MinimalDriverCodec.FRAME_LEN);
    assertEquals("frame length 12, expected 11", reason(buf, 0, MinimalDriverCodec.FRAME_LEN + 1));
  }

  @Test
  public void lengthPastTheBufferIsRejectedNotRead()
  {
    byte[] wire = responseBytes(1, MinimalDriverCodec.CMD_READ, 0, 1, 1);
    assertEquals("frame out of bounds", reason(wire, 1, wire.length));
    assertEquals("frame out of bounds", reason(wire, -1, 2));
    assertEquals("frame out of bounds", reason(null, 0, 0));
  }

  @Test
  public void badStartByteIsRejected()
  {
    byte[] wire = responseBytes(1, MinimalDriverCodec.CMD_READ, 0, 1, 1);
    wire[0] = 0x00;
    assertEquals("bad start byte 0x0", reason(wire, 0, wire.length));
  }

  @Test
  public void corruptedPayloadFailsTheChecksum()
  {
    byte[] wire = responseBytes(1, MinimalDriverCodec.CMD_READ, 0, 1, 1);
    wire[MinimalDriverCodec.VALUE + 3] ^= 0x01;
    assertEquals("bad checksum", reason(wire, 0, wire.length));
  }

  @Test
  public void unknownCommandIsRejected()
  {
    byte[] wire = responseBytes(1, MinimalDriverCodec.CMD_READ, 0, 1, 1);
    wire[MinimalDriverCodec.CMD] = (byte)0x8F;
    // keep the checksum valid so only the command check can reject it
    byte sum = 0;
    for (int i = 0; i < MinimalDriverCodec.CHECKSUM; i++) sum ^= wire[i];
    wire[MinimalDriverCodec.CHECKSUM] = sum;
    assertEquals("unknown command 0x8f", reason(wire, 0, wire.length));
  }

  @Test(expected = IllegalArgumentException.class)
  public void requestRejectsARegisterOutOfRange()
  {
    MinimalDriverCodec.request(0, MinimalDriverCodec.CMD_READ, MinimalDriverCodec.MAX_REGISTER + 1, 0);
  }

  @Test(expected = IllegalArgumentException.class)
  public void requestRejectsASequenceOutOfRange()
  {
    MinimalDriverCodec.request(256, MinimalDriverCodec.CMD_READ, 0, 0);
  }

  @Test
  public void correlationNeedsSameSequenceSameCommandAndAResponse()
  {
    Frame req = MinimalDriverCodec.request(9, MinimalDriverCodec.CMD_READ, 4, 0);
    Frame ok = new Frame(9, MinimalDriverCodec.CMD_READ | MinimalDriverCodec.RESPONSE_BIT, 0, 4, 1);
    Frame otherSeq = new Frame(10, MinimalDriverCodec.CMD_READ | MinimalDriverCodec.RESPONSE_BIT, 0, 4, 1);
    Frame otherCmd = new Frame(9, MinimalDriverCodec.CMD_PING | MinimalDriverCodec.RESPONSE_BIT, 0, 0, 1);
    Frame echoedRequest = new Frame(9, MinimalDriverCodec.CMD_READ, 0, 4, 1);
    assertTrue(MinimalDriverCodec.answers(req, ok));
    assertFalse("another transaction's reply", MinimalDriverCodec.answers(req, otherSeq));
    assertFalse("a stale reply to another command on a wrapped sequence", MinimalDriverCodec.answers(req, otherCmd));
    assertFalse("a request is never a response", MinimalDriverCodec.answers(req, echoedRequest));
  }

  @Test
  public void sequenceWrapsAt256()
  {
    assertEquals(1, MinimalDriverCodec.nextSeq(0));
    assertEquals(0, MinimalDriverCodec.nextSeq(255));
  }

  @Test
  public void hostListIsTrimmedAndBlanksDropped()
  {
    assertArrayEquals(new String[] { "10.0.0.1", "10.0.0.2" }, MinimalDriverCodec.splitHosts(" 10.0.0.1, ,10.0.0.2 ,"));
    assertEquals(0, MinimalDriverCodec.splitHosts(null).length);
    assertEquals(0, MinimalDriverCodec.splitHosts("  ").length);
  }

  @Test
  public void wireValueRoundsAndRejectsWhatDoesNotFit()
  {
    assertEquals(3, MinimalDriverCodec.toWireValue(2.6));
    assertEquals(-3, MinimalDriverCodec.toWireValue(-2.6));
    assertEquals(Integer.MAX_VALUE, MinimalDriverCodec.toWireValue(Integer.MAX_VALUE));
    for (double bad : new double[] { Double.NaN, Double.POSITIVE_INFINITY, 1e10, -1e10 })
    {
      try
      {
        MinimalDriverCodec.toWireValue(bad);
        fail("accepted " + bad);
      }
      catch (IllegalArgumentException expected)
      {
        // a value the wire cannot carry must fail the write, never wrap silently
      }
    }
  }

  @Test
  public void percentIsClamped()
  {
    assertEquals(50, MinimalDriverCodec.percent(1, 2));
    assertEquals(100, MinimalDriverCodec.percent(3, 2));
    assertEquals(100, MinimalDriverCodec.percent(0, 0));
    assertEquals(0, MinimalDriverCodec.percent(-1, 2));
  }
}
