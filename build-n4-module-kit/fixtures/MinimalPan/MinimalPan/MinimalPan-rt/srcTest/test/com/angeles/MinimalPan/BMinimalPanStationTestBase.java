/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package test.com.angeles.MinimalPan;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.BooleanSupplier;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BBoolean;
import javax.baja.sys.BComponent;
import javax.baja.sys.BStation;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;
import javax.baja.test.BTestNgStation;

import org.testng.annotations.AfterMethod;

/**
 * BMinimalPanStationTestBase — the station-test fixture for this module (types/moduleTest.md,
 * types/driver-authoring.md § 12.1).
 *
 * <p>Fox is off: the Windows NRE never binds Fox to loopback only, so the test station sets
 * {@code foxEnabled}, {@code foxsEnabled}, {@code multicastEnabled} and {@code enableAnnouncement} to false.
 * {@link #startStation} then tolerates ONLY BTestNgStation's exact {@value #FOX_NOT_STARTED} failure, and
 * only while the station itself is running; any other setup failure fails the run.</p>
 *
 * <p>Each test mounts its components under a unique name with {@link #mount}; {@link #unmountAll} removes
 * them after every method, so one test never sees another's components. Abstract: the runner skips it
 * (only concrete BTestNg types run), and it is registered in moduleTest-include.xml so its subclasses load.</p>
 */
@NiagaraType
public abstract class BMinimalPanStationTestBase
  extends BTestNgStation
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  /** The exact message of BTestNgStation.startStation when its Fox service is not serving. */
  static final String FOX_NOT_STARTED = "Fox service did not start in time";

  ////////////////////////////////////////////////////////////////
  // Station setup
  ////////////////////////////////////////////////////////////////

  @Override
  protected void configureTestStation(BStation station, String stationName, int webPort, int foxPort)
    throws Exception
  {
    super.configureTestStation(station, stationName, webPort, foxPort);
    BComponent fox = (BComponent)station.getServices().get(FOX_SERVICE);
    if (fox == null) return;
    // By slot name: the test module then needs no compile dependency on fox-rt.
    fox.set("foxEnabled", BBoolean.FALSE);
    fox.set("foxsEnabled", BBoolean.FALSE);
    fox.set("multicastEnabled", BBoolean.FALSE);
    fox.set("enableAnnouncement", BBoolean.FALSE);
  }

  /** Fox is off, so its "not serving" failure is expected; every other failure still fails setup. */
  @Override
  protected void startStation(BStation station)
    throws Exception
  {
    try
    {
      super.startStation(station);
    }
    catch (Exception e)
    {
      if (!FOX_NOT_STARTED.equals(e.getMessage()) || !station.isRunning()) throw e;
    }
  }

  ////////////////////////////////////////////////////////////////
  // Mounting
  ////////////////////////////////////////////////////////////////

  /** Mounts {@code component} on the test station under {@code prefix} plus a name no other test uses. */
  protected <T extends BComponent> T mount(String prefix, T component)
  {
    BStation station = stationHandler.getStation();
    station.add(prefix + '_' + SEQUENCE.incrementAndGet(), component);
    mounted.add(component);
    return component;
  }

  @AfterMethod(alwaysRun = true)
  public void unmountAll()
  {
    BStation station = stationHandler == null ? null : stationHandler.getStation();
    for (int i = mounted.size() - 1; i >= 0; --i)
    {
      BComponent component = mounted.get(i);
      if (station != null && component.isMounted() && component.getParent() == station) station.remove(component);
    }
    mounted.clear();
  }

  ////////////////////////////////////////////////////////////////
  // Waiting
  ////////////////////////////////////////////////////////////////

  /** Polls {@code condition} every 10 ms; true as soon as it holds, false if it never does within {@code timeoutMs}. */
  protected static boolean await(BooleanSupplier condition, long timeoutMs)
    throws InterruptedException
  {
    long deadline = System.currentTimeMillis() + timeoutMs;
    while (true)
    {
      if (condition.getAsBoolean()) return true;
      if (System.currentTimeMillis() >= deadline) return false;
      Thread.sleep(10L);
    }
  }

  /** Samples {@code condition} every 50 ms for {@code windowMs}; true only if every sample holds. */
  protected static boolean holds(BooleanSupplier condition, long windowMs)
    throws InterruptedException
  {
    long deadline = System.currentTimeMillis() + windowMs;
    while (true)
    {
      if (!condition.getAsBoolean()) return false;
      if (System.currentTimeMillis() >= deadline) return true;
      Thread.sleep(50L);
    }
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  private static final AtomicInteger SEQUENCE = new AtomicInteger();
  private final List<BComponent> mounted = new ArrayList<>();
}
