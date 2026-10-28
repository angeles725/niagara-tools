/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package test.com.angeles.MinimalDriver;

import java.io.IOException;
import java.net.DatagramPacket;
import java.net.DatagramSocket;
import java.net.InetAddress;
import java.net.SocketException;
import java.net.SocketTimeoutException;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import com.angeles.MinimalDriver.model.MinimalDriverCodec;
import com.angeles.MinimalDriver.model.MinimalDriverCodec.Frame;

/**
 * MinimalDriverLoopbackDevice — a mock MinimalDriver device for station tests: a UDP socket on
 * 127.0.0.1, ephemeral port, that answers requests with the driver's own codec.
 *
 * <p>It never leaves the loopback interface and binds port 0, so two suites can run at once and no real
 * device or site address is ever touched (types/driver-authoring.md § 12). Plain Java, no Baja type:
 * it is a fixture, not a component. Every request it receives is recorded, answered or not, so a test
 * can assert what the driver sent while the device was silent.</p>
 */
public final class MinimalDriverLoopbackDevice
  implements AutoCloseable
{
  private MinimalDriverLoopbackDevice(int deviceId)
    throws SocketException
  {
    this.deviceId = deviceId;
    this.socket = new DatagramSocket(0, InetAddress.getLoopbackAddress());
    this.socket.setSoTimeout(RECEIVE_POLL_MS);
    this.thread = new Thread(this::serve, "MinimalDriverLoopbackDevice:" + socket.getLocalPort());
    this.thread.setDaemon(true);
  }

  /** Bind 127.0.0.1 on an ephemeral port and start answering. */
  public static MinimalDriverLoopbackDevice start(int deviceId)
    throws SocketException
  {
    MinimalDriverLoopbackDevice d = new MinimalDriverLoopbackDevice(deviceId);
    d.thread.start();
    return d;
  }

  public int port()
  {
    return socket.getLocalPort();
  }

  /** false: keep receiving and recording, answer nothing (a device that is down or unplugged). */
  public void setAnswering(boolean answering)
  {
    this.answering = answering;
  }

  public void setRegister(int register, int value)
  {
    registers.put(register, value);
  }

  /** The register's value, or null when nothing set or wrote it. */
  public Integer register(int register)
  {
    return registers.get(register);
  }

  /** How many requests carrying {@code cmd} arrived so far, answered or not. */
  public int received(int cmd)
  {
    synchronized (log)
    {
      int n = 0;
      for (Frame f : log) if (f.command() == cmd) n++;
      return n;
    }
  }

  /** Commands of every request received after the first {@code from} ones, in arrival order. */
  public List<Integer> commandsSince(int from)
  {
    synchronized (log)
    {
      List<Integer> out = new ArrayList<>();
      for (int i = from; i < log.size(); i++) out.add(log.get(i).command());
      return out;
    }
  }

  public int receivedCount()
  {
    synchronized (log)
    {
      return log.size();
    }
  }

  @Override
  public void close()
  {
    running = false;
    socket.close();
    try
    {
      thread.join(JOIN_MS);
    }
    catch (InterruptedException e)
    {
      Thread.currentThread().interrupt();
    }
  }

  private void serve()
  {
    byte[] buf = new byte[MinimalDriverCodec.BUFFER_LEN];
    while (running)
    {
      DatagramPacket in = new DatagramPacket(buf, buf.length);
      try
      {
        socket.receive(in);
      }
      catch (SocketTimeoutException e)
      {
        continue;   // wake up to re-check running
      }
      catch (IOException e)
      {
        return;     // closed
      }

      Frame req;
      try
      {
        req = MinimalDriverCodec.decode(in.getData(), in.getOffset(), in.getLength());
      }
      catch (IllegalArgumentException e)
      {
        continue;   // not a frame: a real device would ignore it too
      }
      if (req.isResponse()) continue;
      synchronized (log)
      {
        log.add(req);
      }
      if (!answering) continue;

      byte[] out = MinimalDriverCodec.encode(answer(req));
      try
      {
        socket.send(new DatagramPacket(out, out.length, in.getAddress(), in.getPort()));
      }
      catch (IOException e)
      {
        return;
      }
    }
  }

  private Frame answer(Frame req)
  {
    int value;
    switch (req.command())
    {
      case MinimalDriverCodec.CMD_PING:
        value = deviceId;
        break;
      case MinimalDriverCodec.CMD_READ:
        Integer v = registers.get(req.register);
        value = v == null ? 0 : v;
        break;
      case MinimalDriverCodec.CMD_WRITE:
        registers.put(req.register, req.value);
        value = req.value;
        break;
      default:
        value = registers.size();
        break;
    }
    return new Frame(req.seq, req.command() | MinimalDriverCodec.RESPONSE_BIT, MinimalDriverCodec.STATUS_OK,
      req.register, value);
  }

  private static final int RECEIVE_POLL_MS = 100;
  private static final int JOIN_MS = 2000;

  private final int deviceId;
  private final DatagramSocket socket;
  private final Thread thread;
  private final Map<Integer, Integer> registers = new ConcurrentHashMap<>();
  private final List<Frame> log = new ArrayList<>();
  private volatile boolean answering = true;
  private volatile boolean running = true;
}
