/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash;

/**
 * MinimalDashLogic — zero-Baja, zero-external-dependency pure decision class.
 *
 * <p>Provides the setpoint-validity guard so the contract is testable in plain
 * JUnit without a running station or dev license. This is the dashboard type's
 * zero-Baja test seam: extract each decision into a pure class, test it standalone.
 * [ev: build-verify.md §Unit tests; dashboard.md § rt — the facade (no logic)]</p>
 */
final class MinimalDashLogic
{
  private MinimalDashLogic() {}

  /**
   * Returns {@code true} when {@code value} is a finite, writable setpoint.
   *
   * <p>NaN or infinite values must be rejected before writing to a facade slot —
   * they would propagate to the control module and may cause unexpected behavior.
   * This guard is testable without Baja.</p>
   *
   * @param value setpoint candidate (from operator input or POST /api/setpoint)
   * @return true when value is finite (not NaN, not +/-infinity)
   */
  static boolean isValidSetpoint(double value)
  {
    return Double.isFinite(value);
  }
}
