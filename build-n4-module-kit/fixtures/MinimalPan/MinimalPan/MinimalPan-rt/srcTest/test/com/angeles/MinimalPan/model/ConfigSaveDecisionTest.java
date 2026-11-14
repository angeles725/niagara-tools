/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan.model;

import org.junit.Test;
import static org.junit.Assert.*;

import com.angeles.MinimalPan.model.ConfigSaveDecision.Action;

/**
 * Pure JUnit test for {@link ConfigSaveDecision}: each decision carries its action, and only a re-arm
 * carries a delay. Zero-Baja, runs via {@code toolbelt/run-pure-test.sh} in WSL.
 */
public class ConfigSaveDecisionTest
{
  @Test
  public void fixedDecisionsCarryTheirActionAndNoDelay()
  {
    assertEquals(Action.NONE, ConfigSaveDecision.NONE.action());
    assertEquals(Action.SAVE, ConfigSaveDecision.SAVE.action());
    assertEquals(Action.SKIP, ConfigSaveDecision.SKIP.action());
    assertEquals(0L, ConfigSaveDecision.NONE.delayMillis());
    assertEquals(0L, ConfigSaveDecision.SAVE.delayMillis());
    assertEquals(0L, ConfigSaveDecision.SKIP.delayMillis());
  }

  @Test
  public void rearmCarriesTheDelayOfTheNextTicket()
  {
    ConfigSaveDecision d = ConfigSaveDecision.rearm(1500L);
    assertEquals(Action.REARM, d.action());
    assertEquals(1500L, d.delayMillis());
  }
}
