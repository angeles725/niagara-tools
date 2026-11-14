/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan.model;

import org.junit.Test;
import static org.junit.Assert.*;

import com.angeles.MinimalPan.model.ConfigSaveDecision.Action;

/**
 * Pure JUnit test for {@link ConfigSaveScheduler}: one test per rule of the per-STATION debounced save
 * (types/logic.md § Restart &amp; persistence). Zero-Baja, runs via {@code toolbelt/run-pure-test.sh} in WSL.
 *
 * <p>{@link Station} stands for the one thing every owner of a station shares: the time its last save
 * started and whether that save is still running. Each owner has its own scheduler instance.</p>
 */
public class ConfigSaveSchedulerTest
{
  private static final long DEBOUNCE = 2000L;

  /** The station as the adapters see it: one save at a time, its start time visible to every owner. */
  private static final class Station
  {
    long lastSaveStarted = ConfigSaveScheduler.NO_SAVE_YET;
    boolean saveRunning;
    int saves;

    /** Fires owner's ticket at now and applies its decision as the adapter would. */
    Action fire(ConfigSaveScheduler owner, long now)
    {
      ConfigSaveDecision d = owner.onTicket(now, lastSaveStarted, saveRunning);
      if (d.action() == Action.SAVE)
      {
        saves++;
        lastSaveStarted = now;
      }
      return d.action();
    }
  }

  @Test
  public void burstInsideTheWindowCoalescesIntoOneSave()
  {
    Station st = new Station();
    ConfigSaveScheduler owner = new ConfigSaveScheduler(DEBOUNCE);
    assertEquals("a change arms one ticket a debounce away", DEBOUNCE, owner.onChange(1000L));
    assertEquals(DEBOUNCE, owner.onChange(1500L));
    assertEquals(DEBOUNCE, owner.onChange(2500L));

    ConfigSaveDecision early = owner.onTicket(3000L, st.lastSaveStarted, false);
    assertEquals("a ticket armed by the first write is early for the last one", Action.REARM, early.action());
    assertEquals("re-armed for the rest of the window", 1500L, early.delayMillis());
    assertEquals(Action.SAVE, st.fire(owner, 4500L));
    assertEquals("a stale ticket after the save does nothing", Action.NONE, st.fire(owner, 4600L));
    assertEquals(1, st.saves);
  }

  @Test
  public void twoOwnersOneChangeOneSave()
  {
    Station st = new Station();
    ConfigSaveScheduler control = new ConfigSaveScheduler(DEBOUNCE);
    ConfigSaveScheduler dashboard = new ConfigSaveScheduler(DEBOUNCE);
    control.onChange(1000L);
    dashboard.onChange(1001L);

    assertEquals(Action.SAVE, st.fire(control, 3000L));
    assertEquals("a save started after the change it covers", Action.SKIP, st.fire(dashboard, 3001L));
    assertEquals(1, st.saves);
    assertEquals("the skipped ticket is spent", Action.NONE, st.fire(dashboard, 3002L));
  }

  @Test
  public void laterChangeAfterThatSaveStillSaves()
  {
    Station st = new Station();
    ConfigSaveScheduler control = new ConfigSaveScheduler(DEBOUNCE);
    ConfigSaveScheduler dashboard = new ConfigSaveScheduler(DEBOUNCE);
    control.onChange(1000L);
    dashboard.onChange(1001L);
    st.fire(control, 3000L);
    st.fire(dashboard, 3001L);

    dashboard.onChange(9000L);
    assertEquals("the save of 3000 does not hold a change of 9000", Action.SAVE, st.fire(dashboard, 11000L));
    assertEquals(2, st.saves);
  }

  @Test
  public void changeDuringARunningSaveIsNotLost()
  {
    Station st = new Station();
    ConfigSaveScheduler owner = new ConfigSaveScheduler(DEBOUNCE);
    owner.onChange(1000L);
    assertEquals(Action.SAVE, st.fire(owner, 3000L));
    st.saveRunning = true;

    owner.onChange(3500L);
    ConfigSaveDecision busy = owner.onTicket(5500L, st.lastSaveStarted, st.saveRunning);
    assertEquals("a save request during a running save is a no-op, so wait for it", Action.REARM, busy.action());
    assertEquals(DEBOUNCE, busy.delayMillis());
    st.saveRunning = false;
    assertEquals("the change written during the first save is saved after it", Action.SAVE, st.fire(owner, 7500L));
    assertEquals(2, st.saves);
  }

  @Test
  public void equalOrBackwardTimestampsNeverLoseASave()
  {
    ConfigSaveScheduler equal = new ConfigSaveScheduler(DEBOUNCE);
    equal.onChange(1000L);
    assertEquals("a save started in the same millisecond may not hold the change",
      Action.SAVE, equal.onTicket(3000L, 1000L, false).action());

    ConfigSaveScheduler future = new ConfigSaveScheduler(DEBOUNCE);
    future.onChange(1000L);
    assertEquals("a save time ahead of the clock is not evidence",
      Action.SAVE, future.onTicket(3000L, 50000L, false).action());

    ConfigSaveScheduler back = new ConfigSaveScheduler(DEBOUNCE);
    back.onChange(90000L);
    assertEquals("the clock stepped back below the change: due now, never re-armed for hours",
      Action.SAVE, back.onTicket(2000L, 1500L, false).action());

    ConfigSaveScheduler burst = new ConfigSaveScheduler(DEBOUNCE);
    burst.onChange(5000L);
    burst.onChange(4000L);
    assertEquals("a save between two writes with reversed stamps does not cover the later write",
      Action.SAVE, burst.onTicket(7000L, 4500L, false).action());
  }
}
