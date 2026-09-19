/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash;

import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BComponent;
import javax.baja.sys.Flags;
import javax.baja.sys.Property;
import javax.baja.sys.Type;

/**
 * BMinimalDash — minimal dashboard facade component.
 *
 * <p>A pure slot container: one OPERATOR-writable config slot ({@code setpoint})
 * and one READONLY display slot ({@code status}) that the integrator links from
 * a control module's output point. No control logic lives here.</p>
 *
 * <p>The HTTP servlet ({@link com.angeles.MinimalDash.ux.BMinimalDashServlet})
 * walks these slots to serve the browser dashboard. The integrator links control
 * points into {@code status} (display) and the servlet writes back to
 * {@code setpoint} (config) on operator input.</p>
 *
 * <p>All decision logic is in {@link MinimalDashLogic} (zero-Baja, unit-testable).
 * [ev: dashboard.md § rt — the facade (no logic)]</p>
 */
@NiagaraType
@NiagaraProperty(
  name = "setpoint",
  type = "double",
  defaultValue = "0d",
  flags = Flags.SUMMARY | Flags.OPERATOR
)
@NiagaraProperty(
  name = "status",
  type = "double",
  defaultValue = "0d",
  flags = Flags.SUMMARY | Flags.READONLY
)
public class BMinimalDash
  extends BComponent
{
  /*
   * Pre-slotomatic state: the AUTO region is absent.
   * Run build.sh — it invokes :slotomatic before :jar and generates the
   * //region … //endregion block (slot fields, accessors, Type) from the
   * @NiagaraType / @NiagaraProperty annotations above.
   * [ev: B793 §793.3 C1 — pre-slotomatic scaffold convention]
   */

  public BMinimalDash() {}

  // Pure facade — no timer, no control logic.
  // MinimalDashLogic.isValidSetpoint() is the zero-Baja test seam.
}
