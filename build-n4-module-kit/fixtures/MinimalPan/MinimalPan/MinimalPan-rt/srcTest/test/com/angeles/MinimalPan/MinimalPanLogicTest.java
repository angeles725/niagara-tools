/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan;

import org.junit.Test;
import static org.junit.Assert.*;

/**
 * Pure JUnit test for {@link MinimalPanLogic}.
 *
 * <p>Zero-Baja, zero-station, runs via {@code toolbelt/run-pure-test.sh} in WSL.
 * Tests the interval-validity guard that protects {@link BMinimalPan#arm()} from
 * scheduling a zero- or negative-duration periodic ticket (which would loop at
 * CPU speed or never fire). [ev: build-verify.md §Unit tests; B790 §build-sequence]</p>
 */
public class MinimalPanLogicTest
{
  @Test
  public void positiveIntervalIsValid()
  {
    assertTrue("30 s should be valid", MinimalPanLogic.isValidInterval(30_000L));
  }

  @Test
  public void oneMillisIsValid()
  {
    assertTrue("1 ms boundary should be valid", MinimalPanLogic.isValidInterval(1L));
  }

  @Test
  public void zeroIntervalIsInvalid()
  {
    assertFalse("0 ms must be rejected (would loop at CPU speed)", MinimalPanLogic.isValidInterval(0L));
  }

  @Test
  public void negativeIntervalIsInvalid()
  {
    assertFalse("negative ms must be rejected", MinimalPanLogic.isValidInterval(-1L));
  }
}
