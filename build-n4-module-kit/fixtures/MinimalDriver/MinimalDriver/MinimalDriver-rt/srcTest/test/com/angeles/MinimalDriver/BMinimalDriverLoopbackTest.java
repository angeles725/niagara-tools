/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package test.com.angeles.MinimalDriver;

import java.io.IOException;
import java.net.ServerSocket;
import java.util.List;

import javax.baja.control.BNumericPoint;
import javax.baja.control.BNumericWritable;
import javax.baja.driver.util.BPollFrequency;
import javax.baja.job.BJobState;
import javax.baja.naming.BOrd;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BDouble;
import javax.baja.sys.BasicContext;
import javax.baja.sys.Subscriber;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;
import javax.baja.test.BTestNgStation;
import javax.baja.test.TestHelper;

import org.testng.Assert;
import org.testng.annotations.AfterClass;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.Test;

import com.tridium.ndriver.discover.BINDiscoveryObject;
import com.tridium.ndriver.discover.BNDiscoveryJob;

import com.angeles.MinimalDriver.BMinimalDriverDevice;
import com.angeles.MinimalDriver.BMinimalDriverDeviceDiscoveryLeaf;
import com.angeles.MinimalDriver.BMinimalDriverDeviceDiscoveryPreferences;
import com.angeles.MinimalDriver.BMinimalDriverNetwork;
import com.angeles.MinimalDriver.BMinimalDriverProxyExt;
import com.angeles.MinimalDriver.MinimalDriverCodec;

/**
 * BMinimalDriverLoopbackTest — the driver's comm stack in a running test station against
 * {@link MinimalDriverLoopbackDevice}: ping, read, write, timeout and device discovery, through the real
 * network, device, proxy ext and poll scheduler (types/driver-authoring.md § 12, types/moduleTest.md).
 *
 * <p>Every wait is {@link TestHelper#assertWillBeTrue}, never a fixed sleep. A no-answer wait is bounded
 * by the request timeout: {@code RESPONSE_TIMEOUT_MS} plus the framework's 1000 ms padding, times
 * {@code RETRY_COUNT + 1} sends, so about 4 s per request; {@link #WAIT_MS} leaves room for a poll and a
 * ping in a row. BTestNgStation.startStation waits for its Fox service, so Fox stays on, but on a free
 * ephemeral port: the default 1911 is taken whenever a station or platform runs on the same host.</p>
 *
 * <p>Methods run in priority order and share one network, device and two points; each test sets the
 * mock's mode first, so a failure does not cascade through the mock.</p>
 */
@NiagaraType
@Test(groups = { "MinimalDriver" })
public class BMinimalDriverLoopbackTest
  extends BTestNgStation
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  public BMinimalDriverLoopbackTest()
  {
    foxPort = freePort();
  }

  /** A TCP port nothing listens on right now, so the test station's Fox service can bind it. */
  private static int freePort()
  {
    try (ServerSocket s = new ServerSocket(0))
    {
      return s.getLocalPort();
    }
    catch (IOException e)
    {
      throw new IllegalStateException("no free port for the test station's Fox service", e);
    }
  }

  ////////////////////////////////////////////////////////////////
  // Fixture
  ////////////////////////////////////////////////////////////////

  @BeforeClass
  public void addDriver()
    throws Exception
  {
    mock = MinimalDriverLoopbackDevice.start(DEVICE_ID);
    mock.setRegister(READ_REGISTER, READ_VALUE);

    if (getDrivers().get(NETWORK) != null) getDrivers().remove(NETWORK);   // idempotent per <test> run
    network = new BMinimalDriverNetwork();
    getDrivers().add(NETWORK, network);

    device = new BMinimalDriverDevice();
    device.setIpAddress("127.0.0.1");
    device.setUdpPort(mock.port());
    network.add("loopbackDevice", device);

    readPoint = new BNumericPoint();
    readPoint.setProxyExt(proxy(READ_REGISTER));
    device.getPoints().add("readPoint", readPoint);

    writePoint = new BNumericWritable();
    writePoint.setProxyExt(proxy(WRITE_REGISTER));
    device.getPoints().add("writePoint", writePoint);
  }

  @AfterClass(alwaysRun = true)
  public void removeDriver()
  {
    subscriber.unsubscribeAll();
    if (network != null && network.isMounted()) getDrivers().remove(network);
    if (mock != null) mock.close();
  }

  private static BMinimalDriverProxyExt proxy(int register)
  {
    BMinimalDriverProxyExt ext = new BMinimalDriverProxyExt();
    ext.setRegister(register);
    ext.setPollFrequency(BPollFrequency.fast);
    return ext;
  }

  ////////////////////////////////////////////////////////////////
  // Ping
  ////////////////////////////////////////////////////////////////

  @Test(priority = 1)
  public void pingAnsweredMakesDeviceUp()
  {
    mock.setAnswering(true);
    int pings = mock.received(MinimalDriverCodec.CMD_PING);
    device.ping();
    TestHelper.assertWillBeTrue(() -> mock.received(MinimalDriverCodec.CMD_PING) > pings, WAIT_MS,
      "the ping action never reached the mock");
    TestHelper.assertWillBeTrue(() -> !device.isDown() && !device.getHealth().getDown(), WAIT_MS,
      () -> "device should be up after an answered ping: " + device.getStatus() + " " + device.getHealth().getLastFailCause());
  }

  @Test(priority = 2)
  public void pingUnansweredMakesDeviceDown()
  {
    mock.setAnswering(false);
    device.ping();
    TestHelper.assertWillBeTrue(device::isDown, WAIT_MS,
      () -> "device should be down after an unanswered ping: " + device.getStatus());

    mock.setAnswering(true);
    device.ping();
    TestHelper.assertWillBeTrue(() -> !device.isDown(), WAIT_MS,
      () -> "device should be up again once the mock answers: " + device.getStatus());
  }

  ////////////////////////////////////////////////////////////////
  // Read and write
  ////////////////////////////////////////////////////////////////

  @Test(priority = 3)
  public void subscribedProxyReadsTheRegister()
  {
    mock.setAnswering(true);
    subscriber.subscribe(readPoint);
    TestHelper.assertWillBeTrue(() -> readPoint.getOut().getValue() == READ_VALUE && readPoint.getStatus().isOk(),
      WAIT_MS, () -> "readPoint should read " + READ_VALUE + " with status ok: " + readPoint.getOut());
  }

  @Test(priority = 4)
  public void writeReachesTheDevice()
  {
    mock.setAnswering(true);
    subscriber.subscribe(writePoint);
    writePoint.set(BDouble.make(WRITE_VALUE));
    TestHelper.assertWillBeTrue(() -> Integer.valueOf(WRITE_VALUE).equals(mock.register(WRITE_REGISTER)), WAIT_MS,
      () -> "the mock should hold the written value, has " + mock.register(WRITE_REGISTER));
    TestHelper.assertWillBeTrue(() -> writePoint.getOut().getValue() == WRITE_VALUE && writePoint.getStatus().isOk(),
      WAIT_MS, () -> "writePoint should show the written value with status ok: " + writePoint.getOut());
  }

  ////////////////////////////////////////////////////////////////
  // Timeout
  ////////////////////////////////////////////////////////////////

  /** A poll that times out faults the point and pings the device; answers again bring both back. */
  @Test(priority = 5)
  public void pollTimeoutFaultsPointThenRepings()
  {
    subscriber.subscribe(readPoint);
    mock.setAnswering(true);
    TestHelper.assertWillBeTrue(() -> readPoint.getStatus().isOk(), WAIT_MS, "readPoint should start ok");

    int mark = mock.receivedCount();
    mock.setAnswering(false);
    TestHelper.assertWillBeTrue(() -> readPoint.getStatus().isFault() || readPoint.getStatus().isDown(), WAIT_MS,
      () -> "a timed-out poll should fault the point: " + readPoint.getStatus());
    TestHelper.assertWillBeTrue(() -> pingAfterRead(mock.commandsSince(mark)), WAIT_MS,
      () -> "the timed-out read should be followed by a ping, saw " + mock.commandsSince(mark));
    TestHelper.assertWillBeTrue(device::isDown, WAIT_MS,
      () -> "the unanswered re-ping should mark the device down: " + device.getStatus());

    mock.setAnswering(true);
    device.ping();
    TestHelper.assertWillBeTrue(() -> !device.isDown() && readPoint.getStatus().isOk(), WAIT_MS,
      () -> "device and point should recover: " + device.getStatus() + " / " + readPoint.getStatus());
  }

  private static boolean pingAfterRead(List<Integer> commands)
  {
    int read = commands.indexOf(MinimalDriverCodec.CMD_READ);
    return read >= 0 && commands.subList(read, commands.size()).contains(MinimalDriverCodec.CMD_PING);
  }

  ////////////////////////////////////////////////////////////////
  // Discovery
  ////////////////////////////////////////////////////////////////

  /**
   * The network declares submitDiscoveryJob itself and implements only
   * {@code doSubmitDiscoveryJob(BNDiscoveryPreferences, Context)}. Invoking the action with a user
   * Context must run that method: it returns the job's ord and the job finds the mock.
   */
  @Test(priority = 6)
  public void discoveryActionRunsTheContextForm()
  {
    mock.setAnswering(true);
    BMinimalDriverDeviceDiscoveryPreferences prefs = new BMinimalDriverDeviceDiscoveryPreferences();
    prefs.setHosts("127.0.0.1");
    prefs.setUdpPort(mock.port());

    BOrd ord = (BOrd)network.invoke(BMinimalDriverNetwork.submitDiscoveryJob, prefs,
      new BasicContext(getTestSuperUser()));
    Assert.assertNotNull(ord, "submitDiscoveryJob returned no job ord");
    Assert.assertFalse(ord.isNull(), "submitDiscoveryJob returned a null ord");

    BNDiscoveryJob job = (BNDiscoveryJob)ord.get(network);
    TestHelper.assertWillBeTrue(() -> job.getJobState() == BJobState.success, WAIT_MS,
      () -> "discovery job should succeed, state " + job.getJobState());
    BINDiscoveryObject[] found = job.getRootDiscoveryObjects();
    Assert.assertEquals(found.length, 1, "one device answers on the loopback port");
    Assert.assertEquals(((BMinimalDriverDeviceDiscoveryLeaf)found[0]).getDeviceId(), DEVICE_ID);
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  private static final String NETWORK = "MinimalDriverLoopback";
  private static final int DEVICE_ID = 4711;
  private static final int READ_REGISTER = 1;
  private static final int READ_VALUE = 1234;
  private static final int WRITE_REGISTER = 2;
  private static final int WRITE_VALUE = 42;
  /** Two timed-out requests (a poll, then a ping) at about 4 s each, plus a fast poll cycle. */
  private static final long WAIT_MS = 15_000L;

  private final Subscriber subscriber = Subscriber.make(e -> { });
  private MinimalDriverLoopbackDevice mock;
  private BMinimalDriverNetwork network;
  private BMinimalDriverDevice device;
  private BNumericPoint readPoint;
  private BNumericWritable writePoint;
}
