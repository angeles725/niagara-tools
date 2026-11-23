/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

/**
 * LoginRedirect — where Niagara sends a user after a successful login: always the dashboard, whatever page they
 * came from; a page of the dashboard itself is kept. The login template stores the result as the session's
 * {@code originURI}, which Niagara's authenticator redirects to after the password checks out. Only a
 * station-relative path is ever returned, so the redirect cannot leave the station. Pure Java.
 *
 * <p>DASHBOARD is the mount of the -ux servlet (BModuleAServlet.getServletName()): change both together.</p>
 */
public final class LoginRedirect
{
  /** The dashboard servlet's home. */
  public static final String DASHBOARD = "/modulea/";

  private static final String DASHBOARD_ROOT = "/modulea";

  private LoginRedirect() {}

  /** The page to land on after login, from the page the user was sent to the login from. */
  public static String target(String origin)
  {
    if (origin == null || !origin.startsWith(DASHBOARD_ROOT)) return DASHBOARD;
    String rest = origin.substring(DASHBOARD_ROOT.length());
    if (rest.isEmpty()) return DASHBOARD;
    return SAFE_PAGE.matcher(rest).matches() && !rest.contains("..") && !rest.contains("//") ? origin : DASHBOARD;
  }

  /**
   * A page of the dashboard: a slash, then plain path characters and an optional plain query. Anything else (dot
   * segments, a second slash, a backslash, a percent escape, a space or a control character) lands on the dashboard.
   */
  private static final java.util.regex.Pattern SAFE_PAGE =
      java.util.regex.Pattern.compile("/[A-Za-z0-9._~/-]*(\\?[A-Za-z0-9._~=&-]*)?");
}
