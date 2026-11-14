/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan.model;

/**
 * ConfigSaveScheduler — the kit's ONE reference for the debounced station save of operator config
 * (types/logic.md § Restart &amp; persistence). Zero-Baja: it only decides; the adapter owns the ticket,
 * the clock and the save request.
 *
 * <p>The rule is per STATION: one operator change yields at most one station save however many modules
 * observe it. Only the component that OWNS the persisted slot has a scheduler; a facade that forwards or
 * mirrors the write has none. Where two owners can still fire for one change, the later ticket skips its
 * save because a station save started after the change it covers.</p>
 *
 * <p>Fails toward saving: a save is skipped only on positive evidence (a save that started strictly after
 * the last covered change and not ahead of the clock). Equal stamps, an unknown save time and a clock that
 * stepped back all save.</p>
 *
 * <p>Adapter wiring, in the owning BComponent (it cannot be instantiated offline, so it is not shipped as
 * code; pin its order with a source-structural test):</p>
 * <pre>
 *   // field:      private final ConfigSaveScheduler saver = new ConfigSaveScheduler(DEBOUNCE_MILLIS);
 *   //             private Clock.Ticket saveTicket;          // cancelled in stopped()
 *   // changed():  if (ConfigSavePolicy says "operator, persisted, running, steady state")
 *   //               arm(saver.onChange(Clock.millis()));
 *   // ticket:     ConfigSaveDecision d = saver.onTicket(Clock.millis(), lastSaveStarted(), saveRunning());
 *   //             SAVE  -> Sys.getStation().save();          // the ONLY call site of the module
 *   //             REARM -> arm(d.delayMillis());
 *   //             SKIP, NONE -> nothing
 *   // arm(ms):    cancel saveTicket, then Clock.schedule(this, BRelTime.make(ms), saveTick, null)
 *   // lastSaveStarted() / saveRunning(): startTime and jobState of the newest StationSaveJob* child of
 *   //             Services/JobService (station-wide, so it sees a save another module requested);
 *   //             NO_SAVE_YET and false when there is none (after a restart, or no JobService).
 * </pre>
 *
 * [ev: retro double-station-save Δ2]
 */
public final class ConfigSaveScheduler
{
  /** {@code lastSaveStarted} when the station shows no save: nothing is known, so the ticket saves. */
  public static final long NO_SAVE_YET = Long.MIN_VALUE;

  private final long debounceMillis;
  private boolean pending;
  private long changeAt;

  /**
   * @param debounceMillis quiet time after the last write before the save is requested; a value below 1
   *                       is taken as 1 so a ticket is never scheduled with a zero delay
   */
  public ConfigSaveScheduler(long debounceMillis)
  {
    this.debounceMillis = Math.max(1L, debounceMillis);
  }

  /**
   * An operator, persisted config slot changed at {@code now}. The change joins the pending ticket (a
   * burst is one save).
   *
   * @return the delay after which the adapter fires the ticket, replacing any ticket already armed
   */
  public long onChange(long now)
  {
    // The covered change is the latest stamp seen: if the clock stepped back between two writes the
    // older, larger stamp stays, so a save between them is never taken as covering both.
    if (!pending || now > changeAt)
    {
      changeAt = now;
    }
    pending = true;
    return debounceMillis;
  }

  /**
   * The ticket fired at {@code now}.
   *
   * @param lastSaveStarted when the station's last save started, or {@link #NO_SAVE_YET}
   * @param saveRunning     whether that save is still running (a save request during one is a no-op)
   */
  public ConfigSaveDecision onTicket(long now, long lastSaveStarted, boolean saveRunning)
  {
    if (!pending)
    {
      return ConfigSaveDecision.NONE;
    }
    long quiet = now - changeAt;
    if (quiet >= 0L && quiet < debounceMillis)
    {
      return ConfigSaveDecision.rearm(debounceMillis - quiet);
    }
    if (covers(lastSaveStarted, now))
    {
      pending = false;
      return ConfigSaveDecision.SKIP;
    }
    if (saveRunning)
    {
      return ConfigSaveDecision.rearm(debounceMillis);
    }
    pending = false;
    return ConfigSaveDecision.SAVE;
  }

  /** Whether a save that started at {@code saveStarted} holds every pending change. */
  private boolean covers(long saveStarted, long now)
  {
    return saveStarted != NO_SAVE_YET && saveStarted > changeAt && saveStarted <= now;
  }
}
