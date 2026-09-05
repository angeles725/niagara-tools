/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Action;
import javax.baja.sys.BComponent;
import javax.baja.sys.BRelTime;
import javax.baja.sys.Clock;
import javax.baja.sys.Context;
import javax.baja.sys.Flags;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

/**
 * BMinimalPan — B790 minimal correct N4 module skeleton.
 *
 * <p>One OPERATOR-writable setpoint, a configurable periodic timer, and a HIDDEN
 * tick action. Demonstrates the minimal viable Baja component: property lifecycle,
 * Clock.Ticket management, and the annotation/slotomatic pattern.</p>
 *
 * <p>All control logic is in {@link MinimalPanLogic} (zero-Baja, unit-testable).</p>
 *
 * [ev: B790 §790.1 / §790.2 — minimal N4 module skeleton]
 */
@NiagaraType
@NiagaraProperty(
  name = "setpoint",
  type = "double",
  defaultValue = "0d",
  flags = Flags.SUMMARY | Flags.OPERATOR
)
@NiagaraProperty(
  name = "interval",
  type = "BRelTime",
  defaultValue = "BRelTime.makeSeconds(30)",
  flags = Flags.SUMMARY | Flags.OPERATOR
)
@NiagaraAction(name = "tickExpired", flags = Flags.HIDDEN)
public class BMinimalPan
  extends BComponent
{
//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/
//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/

  public BMinimalPan() {}

  ////////////////////////////////////////////////////////////////
  // Lifecycle
  ////////////////////////////////////////////////////////////////

  /**
   * Called when the component is started.
   * If the station is already at steady state (component added at runtime), arm the
   * periodic timer immediately — atSteadyState() only fires once at boot. [ev: B729]
   */
  @Override
  public void started() throws Exception
  {
    super.started();
    if (!Sys.atSteadyState()) return;
    arm();
  }

  /**
   * Called once when the station reaches steady state (normal boot path).
   * Arms the periodic timer. [ev: B729]
   */
  @Override
  public void atSteadyState()
  {
    arm();
  }

  /**
   * Called when the component is stopped.
   * Cancels the periodic timer so no callbacks fire after the component is disabled.
   * [ev: B787 — ticket-without-stopped-cancel biting check]
   */
  @Override
  public void stopped() throws Exception
  {
    if (ticket != null) { ticket.cancel(); ticket = null; }
    super.stopped();
  }

  /**
   * Called when any slot changes.
   * Re-arms the timer when the {@code interval} slot changes so the new period takes
   * effect immediately (no wait for the current ticket to expire). [ev: B775]
   */
  @Override
  public void changed(Property p, Context cx)
  {
    super.changed(p, cx);
    if (p == interval && isRunning()) arm();
  }

  ////////////////////////////////////////////////////////////////
  // Timer
  ////////////////////////////////////////////////////////////////

  /**
   * Arms (or re-arms) the periodic timer.
   * Cancels any existing ticket first to prevent double-firing on interval changes.
   * Uses {@code Clock.schedulePeriodically} so the action fires repeatedly without
   * manual re-arming in the callback (one ticket per arm call). [ev: B790 §790.2]
   */
  private void arm()
  {
    if (ticket != null) { ticket.cancel(); ticket = null; }
    if (!MinimalPanLogic.isValidInterval(getInterval().getMillis())) return;
    ticket = Clock.schedulePeriodically(this, getInterval(), tickExpired, null);
  }

  /**
   * Timer callback — called every {@code interval} by the Baja engine.
   * In this skeleton there are no output slots to update; it demonstrates the
   * periodic callback hook pattern. [ev: B790 §790.2]
   */
  public void doTickExpired()
  {
    // Minimal skeleton — no output slots to update.
    // A real module would run its control logic here.
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  /** Active timer ticket; null when the component is stopped or interval is invalid. */
  private Clock.Ticket ticket;
}
