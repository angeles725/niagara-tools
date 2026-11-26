/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login;

import com.example.ModuleA.login.model.LoginActivation;
import java.util.logging.Level;
import java.util.logging.Logger;
import javax.baja.util.BTypeSpec;
import javax.baja.web.BWebService;

/**
 * LoginActivator — the Baja side of {@link LoginActivation}: reads WebService.loginTemplate as text, asks the
 * pure rule, writes the slot only when it says SET and logs what it did (logger "ModuleA.login"). The caller
 * (BModuleAService) saves the station when this returns true. It never throws: any problem here, an Error
 * included (the first use of BModuleALoginTemplate runs its class initializer; a type that fails to load, or a
 * web-rt class that is missing, is an Error, not an Exception), is logged and the station keeps starting with
 * the login it had. Only a VirtualMachineError is rethrown ({@link LoginActivation#guarded}).
 */
public final class LoginActivator
{
  private static final Logger LOG = Logger.getLogger("ModuleA.login");
  private static final String FAILED = "login template activation failed; WebService.loginTemplate is unchanged";

  private LoginActivator() {}

  /**
   * Apply the rule once, at station start.
   *
   * @return true when WebService.loginTemplate was written (the station needs a save)
   */
  public static boolean apply(boolean applyLogin)
  {
    return LoginActivation.guarded(LOG, FAILED, () ->
    {
      BWebService web = BWebService.getMainService();
      BTypeSpec current = web.getLoginTemplate();
      BTypeSpec own = BTypeSpec.make(BModuleALoginTemplate.TYPE);  // resolved from the type: no typo can name a missing one
      LoginActivation.Decision d = LoginActivation.decide(text(current), applyLogin, text(own));
      if (d.sets()) web.setLoginTemplate(own);  // written before its log line, so a failed write never logs success
      LOG.log(d.action() == LoginActivation.Action.KEEP_CUSTOM ? Level.WARNING : Level.INFO, d.message());
      return d.sets();
    });
  }

  /** "module:Type", or null for the empty spec. */
  private static String text(BTypeSpec spec)
  {
    return spec == null || spec.isNull() ? null : spec.getModuleName() + ":" + spec.getTypeName();
  }
}
