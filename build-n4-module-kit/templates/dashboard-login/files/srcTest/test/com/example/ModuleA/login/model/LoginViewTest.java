/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertNull;

import com.example.ModuleA.login.model.LoginView.Inputs;
import com.example.ModuleA.login.model.LoginView.Screen;
import org.junit.Test;

/** Which screen the page draws and which it leaves to Niagara's own template (STOCK). */
public class LoginViewTest
{
  private static Inputs user() { return new Inputs().username("operator").digest(true); }

  @Test
  public void noUsername_isTheUsernameScreen()
  {
    assertEquals(Screen.USERNAME, LoginView.screen(new Inputs()));
    assertEquals(Screen.USERNAME, LoginView.screen(new Inputs().username("  ")));
  }

  @Test
  public void aDigestUser_getsThePasswordScreen()
  {
    assertEquals(Screen.PASSWORD, LoginView.screen(user()));
  }

  @Test
  public void forcedResetAndExpiryHaveTheirScreens()
  {
    assertEquals(Screen.RESET, LoginView.screen(user().forceReset(true)));
    assertEquals(Screen.EXPIRING, LoginView.screen(user().passwordExpires(1L)));
  }

  @Test
  public void whatThePageCannotDraw_fallsBackToNiagara()
  {
    assertEquals(Screen.STOCK, LoginView.screen(new Inputs().username("operator")));          // not digest
    assertEquals(Screen.STOCK, LoginView.screen(user().twoFactor(true)));
    assertEquals(Screen.STOCK, LoginView.screen(user().passwordExpires(1L).networkUser(true)));
  }

  @Test
  public void errors_comeFromTheCauseOrTheFailFlag()
  {
    assertNull(LoginView.error(user()));
    assertEquals(LoginView.WRONG, LoginView.error(user().authFail(true)));
    assertEquals(LoginView.WRONG, LoginView.error(user().failureCause("NOT_A_CAUSE")));
    assertEquals("This user signed in somewhere else.", LoginView.error(user().failureCause("CONCURRENT_SESSION")));
  }
}
