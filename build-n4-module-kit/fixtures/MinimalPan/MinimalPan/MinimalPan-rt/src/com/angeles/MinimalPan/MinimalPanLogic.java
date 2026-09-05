/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan;

/**
 * MinimalPanLogic — zero-Baja, zero-external-dependency pure decision class.
 *
 * <p>Extracts the interval-validity guard from {@link BMinimalPan#arm()} so the
 * rule can be tested standalone with JUnit without a running station or dev license.
 * This is the B790 §build-verify.md §Unit-tests pattern:
 * "extract the control decision into a ZERO-Baja pure class and run it standalone."</p>
 *
 * [ev: build-verify.md §Unit tests; retro rt-hardening #5]
 */
final class MinimalPanLogic
{
  private MinimalPanLogic() {}

  /**
   * Returns {@code true} when {@code millis} is a positive, scheduleable interval.
   *
   * <p>Zero or negative values would cause {@code Clock.schedulePeriodically} to fire
   * immediately and/or loop at CPU speed — both are defects. The guard in
   * {@link BMinimalPan#arm()} delegates here so this contract is testable without Baja.</p>
   *
   * @param millis interval in milliseconds (from {@code BRelTime.getMillis()})
   * @return true when millis &gt; 0
   */
  static boolean isValidInterval(long millis)
  {
    return millis > 0L;
  }
}
