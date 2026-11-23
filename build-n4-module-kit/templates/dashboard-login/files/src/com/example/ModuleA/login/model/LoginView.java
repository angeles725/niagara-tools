/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import java.util.HashMap;
import java.util.Map;

/**
 * LoginView — which screen the login page shows and which error it explains, from what Niagara left in the
 * request and the session (web-rt LoginServlet / NiagaraAuthenticator):
 * - no username yet -> USERNAME (the form posts to {@code login});
 * - session {@code forceReset} -> RESET (new password, posted as {@code resetToken});
 * - session {@code passwordExpires} -> EXPIRING (change it now: {@code yes-reset} / {@code no-reset}),
 *   STOCK for a network user (Niagara's OK-only notice);
 * - otherwise PASSWORD (SCRAM through {@code /login/core/auth.min.js}, then {@code /j_security_check/});
 * - a scheme the page does not speak (not digest) or two-factor -> STOCK: Niagara's own page renders.
 * An error comes from {@code ?auth=fail} or the {@code niagara_failure_cause} cookie (a LoginFailureCause
 * name). Pure Java.
 */
public final class LoginView
{
  /** The screens of the login page. */
  public enum Screen { USERNAME, PASSWORD, RESET, EXPIRING, STOCK }

  static final String WRONG = "Wrong user name or password.";

  private static final Map<String, String> CAUSES = new HashMap<String, String>();
  static
  {
    CAUSES.put("SESSION_TIMEOUT", "Your session closed after a period of inactivity. Sign in again.");
    CAUSES.put("CONCURRENT_SESSION", "This user signed in somewhere else.");
    CAUSES.put("PASSWORD_RESET_REQUIRED", "You have to change your password to continue.");
    CAUSES.put("SECURE_CONNECTION_REQUIRED", "This station only accepts secure connections (https).");
    CAUSES.put("INSECURE_PASSWORD_RESET", "Changing the password needs a secure connection (https).");
    CAUSES.put("ILLEGAL_NETWORK_USER_RESET", "This user's password is changed on their network server, not here.");
    CAUSES.put("LOGIN_INTERFACE_NOT_SUPPORTED", "This user cannot sign in from a browser.");
  }

  private LoginView() {}

  /** What the page knows about this login attempt. */
  public static final class Inputs
  {
    String username;
    boolean digest;
    boolean twoFactor;
    boolean forceReset;
    boolean authFail;
    boolean networkUser;
    Long passwordExpires;
    String failureCause;

    public Inputs username(String v) { username = v; return this; }
    /** The user's scheme is a password (SCRAM digest) scheme the page can speak. */
    public Inputs digest(boolean v) { digest = v; return this; }
    public Inputs twoFactor(boolean v) { twoFactor = v; return this; }
    public Inputs forceReset(boolean v) { forceReset = v; return this; }
    public Inputs passwordExpires(Long v) { passwordExpires = v; return this; }
    public Inputs authFail(boolean v) { authFail = v; return this; }
    /** A network (LDAP/AD) user: their expiry notice is OK-only, so the stock page renders it. */
    public Inputs networkUser(boolean v) { networkUser = v; return this; }
    public Inputs failureCause(String v) { failureCause = v; return this; }
    public Long passwordExpires() { return passwordExpires; }
  }

  /** The screen to render. */
  public static Screen screen(Inputs in)
  {
    if (in.username == null || in.username.trim().isEmpty()) return Screen.USERNAME;
    if (!in.digest || in.twoFactor) return Screen.STOCK;
    if (in.forceReset) return Screen.RESET;
    if (in.passwordExpires != null) return in.networkUser ? Screen.STOCK : Screen.EXPIRING;
    return Screen.PASSWORD;
  }

  /** The error to show, or null when there is none. */
  public static String error(Inputs in)
  {
    if (in.failureCause != null)
    {
      String text = CAUSES.get(in.failureCause);
      return text != null ? text : WRONG;
    }
    return in.authFail ? WRONG : null;
  }
}
