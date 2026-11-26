/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA;

import com.example.ModuleA.login.LoginActivator;
import com.example.ModuleA.login.model.LoginActivation;
import java.util.logging.Logger;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BAbstractService;
import javax.baja.sys.BRelTime;
import javax.baja.sys.BStation;
import javax.baja.sys.Clock;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

/**
 * BModuleAService — the module's service (drop it under the station's Services). At station start it makes the
 * module's login template the station's login page (the rule is LoginActivation: only when
 * WebService.loginTemplate is empty, the stock one or a retired type of this module; never over another custom
 * template) and saves the station. Put the dashboard's other station-wide behaviour here too.
 *
 * <p>Rollback: set {@code applyLogin} false AND clear Services/WebService.loginTemplate (the slot stays set
 * otherwise); then remove the module only after the slot no longer names its type (a station whose slot names a
 * missing type answers HTTP 500 on /login). [ev: types/dashboard.md § The module ships its own login]</p>
 */
@NiagaraType
// Login activation: at station start the service sets WebService.loginTemplate to this module's login template
// when it is empty, the stock one or a retired type of this module. False turns that off; it never changes a
// template that is already set.
@NiagaraProperty(
  name = "applyLogin",
  type = "boolean",
  defaultValue = "true"
)
public class BModuleAService
  extends BAbstractService
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. Run build.sh; it invokes :slotomatic before :jar.
   */

  public BModuleAService() {}

  /**
   * Deliberately empty: the login is applied once, at station start (stationStarted below), never when the
   * service is added to a running station (the next start applies it).
   */
  @Override
  public void started() throws Exception
  {
    super.started();
  }

  /**
   * Station start: make the module's login the station's login page when applyLogin is true (LoginActivation
   * decides), then save the station. Runs once all components started, so WebService exists. A service added
   * to a running station does nothing until the next start.
   */
  @Override
  public void stationStarted() throws Exception
  {
    super.stationStarted();
    if (LoginActivator.apply(getApplyLogin())) scheduleLoginSave();
  }

  /** Settle time before the station save of a login activation, so it runs after the start sequence. */
  private static final long LOGIN_SAVE_DELAY_MS = 5000L;

  private static final Logger LOG = Logger.getLogger("ModuleA.login");

  /** The pending station save of a login activation (expired = none). */
  private volatile Clock.Ticket loginSaveTicket = Clock.expiredTicket;

  /** Save the station (its own save action) so the template survives a restart. */
  private void scheduleLoginSave()
  {
    // guarded: a problem here (an Error from a class that cannot load included) never escapes stationStarted()
    LoginActivation.guarded(LOG,
        "login template set but the station save could not be scheduled; the next start applies it again", () ->
    {
      BStation station = Sys.getStation();
      if (station == null)
      {
        LOG.warning("login template set but the station save was skipped (no station); the next start applies it again");
        return false;
      }
      loginSaveTicket = Clock.schedule(station, BRelTime.make(Math.max(LOGIN_SAVE_DELAY_MS, 1L)), BStation.save, null);
      return true;
    });
  }

  /** Cancel a pending login save when the service stops (a station stop drops it anyway). */
  @Override
  public void stopped() throws Exception
  {
    loginSaveTicket.cancel();
    super.stopped();
  }

  /** BAbstractService contract: the service type(s) this component provides. */
  @Override
  public Type[] getServiceTypes() { return new Type[] { TYPE }; }
}
