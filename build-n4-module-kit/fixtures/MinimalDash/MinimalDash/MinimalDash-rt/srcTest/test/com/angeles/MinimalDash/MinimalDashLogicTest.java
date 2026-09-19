/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash;

import org.junit.Test;
import static org.junit.Assert.*;

/**
 * Pure JUnit test for {@link MinimalDashLogic}.
 *
 * <p>Zero-Baja, zero-station, runs via {@code toolbelt/run-pure-test.sh} in WSL.
 * Tests the setpoint-validity guard that protects {@link BMinimalDash} from
 * accepting NaN or infinite values that would propagate to the control module.
 * [ev: build-verify.md §Unit tests; dashboard.md § rt — facade test seam]</p>
 */
public class MinimalDashLogicTest
{
  @Test
  public void finiteValueIsValid()
  {
    assertTrue("a normal double should be valid", MinimalDashLogic.isValidSetpoint(20.0));
  }

  @Test
  public void zeroIsValid()
  {
    assertTrue("zero is a finite setpoint", MinimalDashLogic.isValidSetpoint(0.0));
  }

  @Test
  public void negativeFiniteIsValid()
  {
    assertTrue("negative setpoint is valid (e.g. -18 C for a freezer)",
        MinimalDashLogic.isValidSetpoint(-18.0));
  }

  @Test
  public void nanIsInvalid()
  {
    assertFalse("NaN must be rejected", MinimalDashLogic.isValidSetpoint(Double.NaN));
  }

  @Test
  public void positiveInfinityIsInvalid()
  {
    assertFalse("+Infinity must be rejected",
        MinimalDashLogic.isValidSetpoint(Double.POSITIVE_INFINITY));
  }

  @Test
  public void negativeInfinityIsInvalid()
  {
    assertFalse("-Infinity must be rejected",
        MinimalDashLogic.isValidSetpoint(Double.NEGATIVE_INFINITY));
  }
}
