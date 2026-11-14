/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan.model;

/**
 * ConfigSaveDecision — what {@link ConfigSaveScheduler} tells its adapter to do when a save ticket fires.
 *
 * <p>Zero-Baja value object: an action and, for {@link Action#REARM}, the delay of the next ticket.</p>
 */
public final class ConfigSaveDecision
{
  /** The three things a fired ticket can do, plus "nothing is pending". */
  public enum Action
  {
    /** No change is waiting: the ticket was stale. Do nothing. */
    NONE,
    /** Request the station save now. */
    SAVE,
    /** A station save that started after the covered change already holds it: request none. */
    SKIP,
    /** Not yet: schedule one more ticket after {@link #delayMillis()}. */
    REARM
  }

  static final ConfigSaveDecision NONE = new ConfigSaveDecision(Action.NONE, 0L);
  static final ConfigSaveDecision SAVE = new ConfigSaveDecision(Action.SAVE, 0L);
  static final ConfigSaveDecision SKIP = new ConfigSaveDecision(Action.SKIP, 0L);

  private final Action action;
  private final long delayMillis;

  private ConfigSaveDecision(Action action, long delayMillis)
  {
    this.action = action;
    this.delayMillis = delayMillis;
  }

  static ConfigSaveDecision rearm(long delayMillis)
  {
    return new ConfigSaveDecision(Action.REARM, delayMillis);
  }

  public Action action()
  {
    return action;
  }

  /** Delay of the next ticket in milliseconds; meaningful only for {@link Action#REARM}. */
  public long delayMillis()
  {
    return delayMillis;
  }
}
