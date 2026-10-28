/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver.model;

import java.util.ArrayList;
import java.util.List;

/**
 * MinimalDriverCodec — the wire format of the MinimalDriver protocol, with no Baja type in it.
 *
 * <p>The NDriver message classes ({@link MinimalDriverRequest}, {@link MinimalDriverResponse},
 * {@link MinimalDriverMessageFactory}) only delegate here, so every byte rule (length, start byte,
 * checksum, command, request/response correlation) is covered by a pure JUnit test that runs in WSL
 * with {@code toolbelt/run-pure-test.sh}. [ev: kit types/driver-authoring.md § 12]</p>
 *
 * <p>The protocol is a placeholder you replace with your device's: one fixed-length frame per UDP
 * datagram (UDP never calls {@code LinkMessage.receive}, so one datagram is one frame, § 1.9):</p>
 * <pre>
 *   0      start byte 0xA5
 *   1      sequence number 0..255 (the transaction tag, § 1.7)
 *   2      command; bit 7 set on a response
 *   3      status: 0 ok, anything else is a device error code (responses only)
 *   4..5   register, unsigned 16-bit big-endian
 *   6..9   value, signed 32-bit big-endian
 *   10     checksum: XOR of bytes 0..9
 * </pre>
 */
public final class MinimalDriverCodec
{
  private MinimalDriverCodec() {}

  /** Length of every frame on the wire. */
  public static final int FRAME_LEN = 11;

  /**
   * Receive buffer size. Larger than a frame on purpose: a longer datagram is cut to the buffer
   * (§ 1.9), so a buffer of exactly FRAME_LEN would make an oversized datagram look like a frame.
   */
  public static final int BUFFER_LEN = 64;

  public static final int START = 0xA5;
  public static final int RESPONSE_BIT = 0x80;
  public static final int STATUS_OK = 0;
  public static final int MAX_REGISTER = 0xFFFF;

  /** Ping: the device answers with its device id in {@code value}. */
  public static final int CMD_PING = 0x01;
  /** Read one register: the answer carries its value. */
  public static final int CMD_READ = 0x02;
  /** Write one register: the answer echoes the written value. */
  public static final int CMD_WRITE = 0x03;
  /** Register count: the answer carries the number of registers in {@code value}. */
  public static final int CMD_COUNT = 0x04;

  static final int SEQ = 1;
  static final int CMD = 2;
  static final int STATUS = 3;
  static final int REG = 4;
  static final int VALUE = 6;
  static final int CHECKSUM = 10;

  /** One decoded frame. Immutable. */
  public static final class Frame
  {
    public Frame(int seq, int cmd, int status, int register, int value)
    {
      this.seq = seq;
      this.cmd = cmd;
      this.status = status;
      this.register = register;
      this.value = value;
    }

    /** The command without the response bit. */
    public int command() { return cmd & ~RESPONSE_BIT; }

    public boolean isResponse() { return (cmd & RESPONSE_BIT) != 0; }

    /** A response whose status is not {@link #STATUS_OK}: its value must not be used. */
    public boolean isError() { return status != STATUS_OK; }

    @Override
    public String toString()
    {
      return (isResponse() ? "resp" : "req") + " seq=" + seq + " cmd=" + command() + " status=" + status
        + " reg=" + register + " value=" + value;
    }

    public final int seq;
    public final int cmd;
    public final int status;
    public final int register;
    public final int value;
  }

  /**
   * Build a request frame.
   *
   * @throws IllegalArgumentException for a sequence number, command or register out of range
   */
  public static Frame request(int seq, int cmd, int register, int value)
  {
    if (seq < 0 || seq > 0xFF) throw new IllegalArgumentException("sequence out of range: " + seq);
    if (!isCommand(cmd)) throw new IllegalArgumentException("unknown command: " + cmd);
    if (register < 0 || register > MAX_REGISTER) throw new IllegalArgumentException("register out of range: " + register);
    return new Frame(seq, cmd, STATUS_OK, register, value);
  }

  /** Encode a frame into a new FRAME_LEN array, checksum included. */
  public static byte[] encode(Frame f)
  {
    byte[] b = new byte[FRAME_LEN];
    b[0] = (byte)START;
    b[SEQ] = (byte)f.seq;
    b[CMD] = (byte)f.cmd;
    b[STATUS] = (byte)f.status;
    b[REG] = (byte)(f.register >>> 8);
    b[REG + 1] = (byte)f.register;
    b[VALUE] = (byte)(f.value >>> 24);
    b[VALUE + 1] = (byte)(f.value >>> 16);
    b[VALUE + 2] = (byte)(f.value >>> 8);
    b[VALUE + 3] = (byte)f.value;
    b[CHECKSUM] = checksum(b, 0);
    return b;
  }

  /**
   * Decode {@code len} bytes of {@code buf} from {@code off}.
   *
   * @throws IllegalArgumentException naming the reason when the bytes are not one valid frame:
   *   out of bounds, wrong length, bad start byte, bad checksum or unknown command
   */
  public static Frame decode(byte[] buf, int off, int len)
  {
    if (buf == null || off < 0 || len < 0 || off > buf.length - len)
      throw new IllegalArgumentException("frame out of bounds");
    if (len != FRAME_LEN)
      throw new IllegalArgumentException("frame length " + len + ", expected " + FRAME_LEN);
    if ((buf[off] & 0xFF) != START)
      throw new IllegalArgumentException("bad start byte 0x" + Integer.toHexString(buf[off] & 0xFF));
    byte sum = checksum(buf, off);
    if (sum != buf[off + CHECKSUM]) throw new IllegalArgumentException("bad checksum");
    int cmd = buf[off + CMD] & 0xFF;
    if (!isCommand(cmd & ~RESPONSE_BIT))
      throw new IllegalArgumentException("unknown command 0x" + Integer.toHexString(cmd));
    int register = ((buf[off + REG] & 0xFF) << 8) | (buf[off + REG + 1] & 0xFF);
    int value = ((buf[off + VALUE] & 0xFF) << 24) | ((buf[off + VALUE + 1] & 0xFF) << 16)
      | ((buf[off + VALUE + 2] & 0xFF) << 8) | (buf[off + VALUE + 3] & 0xFF);
    return new Frame(buf[off + SEQ] & 0xFF, cmd, buf[off + STATUS] & 0xFF, register, value);
  }

  /**
   * Correlation rule: {@code response} answers {@code request} when it is a response, carries the
   * request's sequence number and the same command. NComm matches on the tag (the sequence number,
   * § 1.7); this second check makes a stale reply that reuses a wrapped sequence number fail instead
   * of completing the wrong request.
   */
  public static boolean answers(Frame request, Frame response)
  {
    return response.isResponse() && response.seq == request.seq && response.command() == request.command();
  }

  /** The next sequence number, wrapping 255 to 0. */
  public static int nextSeq(int seq)
  {
    return (seq + 1) & 0xFF;
  }

  /**
   * Convert a point value to the wire's signed 32-bit value, rounding to the nearest integer.
   *
   * @throws IllegalArgumentException for NaN, an infinity or a value outside the 32-bit range
   */
  public static int toWireValue(double v)
  {
    if (Double.isNaN(v) || Double.isInfinite(v)) throw new IllegalArgumentException("value is not a number: " + v);
    long r = Math.round(v);
    if (r < Integer.MIN_VALUE || r > Integer.MAX_VALUE) throw new IllegalArgumentException("value out of 32-bit range: " + v);
    return (int)r;
  }

  /** Split a comma-separated host list, trimming blanks and dropping empty entries. */
  public static String[] splitHosts(String hosts)
  {
    List<String> out = new ArrayList<>();
    if (hosts != null)
    {
      for (String h : hosts.split(","))
      {
        String t = h.trim();
        if (!t.isEmpty()) out.add(t);
      }
    }
    return out.toArray(new String[0]);
  }

  /** Whole-percent progress of {@code done} out of {@code total}, clamped to 0..100. */
  public static int percent(int done, int total)
  {
    if (total <= 0) return 100;
    int p = (int)((done * 100L) / total);
    return Math.max(0, Math.min(100, p));
  }

  private static boolean isCommand(int cmd)
  {
    return cmd == CMD_PING || cmd == CMD_READ || cmd == CMD_WRITE || cmd == CMD_COUNT;
  }

  private static byte checksum(byte[] b, int off)
  {
    byte sum = 0;
    for (int i = 0; i < CHECKSUM; i++) sum ^= b[off + i];
    return sum;
  }
}
