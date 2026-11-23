/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

/**
 * LoginPage — the whole login document: one plain card with the screen's title, the error as an alert and the
 * screen's form. Every asset is served by this module under /login/modulea-login/ (no remote URL or font: the
 * station may have no internet). Echoed text is escaped ({@link Html}). One screen per page. Restyle it in
 * rc/modulea-login/login.css; the form contracts below are Niagara's and must not change. Pure Java.
 */
public final class LoginPage
{
  private LoginPage() {}

  /** What one rendering needs. */
  public static final class Model
  {
    final LoginView.Screen screen;
    String username;
    String error;
    String strengthJson;
    String expires;

    public Model(LoginView.Screen screen) { this.screen = screen; }
    public Model username(String v) { username = v; return this; }
    public Model error(String v) { error = v; return this; }
    public Model strengthJson(String v) { strengthJson = v; return this; }
    public Model expires(String v) { expires = v; return this; }
  }

  /** The HTML of the page (the servlet already set text/html; charset=utf-8). */
  public static String render(Model m)
  {
    StringBuilder b = new StringBuilder(2048);
    b.append("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\">")
        .append("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
        .append("<meta name=\"robots\" content=\"noindex\"><title>ModuleA · Sign in</title>")
        .append("<link rel=\"stylesheet\" href=\"/login/modulea-login/login.css\"></head><body>")
        .append("<main class=\"card\"><section class=\"view\" data-view=\"").append(m.screen.name().toLowerCase())
        .append("\" aria-labelledby=\"login-title\"><h1 id=\"login-title\">").append(title(m.screen)).append("</h1>");
    if (m.error != null)
      b.append("<div class=\"alert\" id=\"login-alert\" role=\"alert\">").append(Html.esc(m.error)).append("</div>");
    b.append(form(m))
        .append("</section><noscript><div class=\"alert\">Turn on JavaScript in the browser to sign in.</div></noscript>")
        .append("</main><script src=\"/login/modulea-login/login.js\"></script></body></html>");
    return b.toString();
  }

  private static String title(LoginView.Screen s)
  {
    switch (s)
    {
      case RESET: return "Change your password";
      case EXPIRING: return "Your password is about to expire";
      default: return "Sign in";
    }
  }

  private static String form(Model m)
  {
    switch (m.screen)
    {
      case PASSWORD: return password(m.username);
      case RESET: return reset(m.username, m.strengthJson);
      case EXPIRING: return expiring(m.expires);
      default: return username();
    }
  }

  // --- the form of each screen: the contract Niagara's login servlets read (web-rt preLoginFormN4.vm,
  // digestLoginFormN4.vm, passwordResetFormN4.vm, passwordExpirationFormN4.vm); field names, ids and form
  // targets must not change.

  /** Step 1: the username, posted to /login, which then renders the password step. */
  static String username()
  {
    return "<form method=\"post\" action=\"/login\" class=\"form\">"
        + "<label for=\"username\">User name</label>"
        + "<input id=\"username\" name=\"j_username\" type=\"text\" autocomplete=\"username\""
        + " autocapitalize=\"none\" spellcheck=\"false\" required autofocus>"
        + "<button type=\"submit\" class=\"btn\">Continue</button></form>";
  }

  /** Step 2: the password; login.js runs SCRAM (auth.min.js) and then /j_security_check/. */
  static String password(String user)
  {
    return "<form method=\"post\" action=\"/j_security_check\" class=\"form\" id=\"login-form\">"
        + "<label for=\"username\">User name</label>"
        + "<input id=\"username\" name=\"j_username\" value=\"" + Html.esc(user) + "\" readonly autocomplete=\"username\">"
        + "<a href=\"/prelogin?clear=true\">Change user</a>"
        + "<label for=\"password\">Password</label>"
        + "<input id=\"password\" name=\"j_password\" type=\"password\" autocomplete=\"current-password\" required autofocus>"
        + "<button type=\"submit\" id=\"login-submit\" class=\"btn\">Sign in</button></form>"
        + "<script src=\"/login/core/auth.min.js\"></script>";
  }

  /**
   * Forced change: the new password never has a name (it must not travel in clear); login.js puts its base64
   * into resetToken and submits. data-rules holds the station's strength rules; login.js lists them.
   */
  static String reset(String user, String rulesJson)
  {
    return "<p>You have to change the password of <strong>" + Html.esc(user) + "</strong> to continue.</p>"
        + "<form method=\"post\" action=\"/j_security_check\" class=\"form\" id=\"reset-form\" data-rules=\""
        + Html.esc(rulesJson == null ? "{}" : rulesJson) + "\">"
        + "<label for=\"new-password\">New password</label>"
        + "<input id=\"new-password\" type=\"password\" autocomplete=\"new-password\" required autofocus aria-describedby=\"rules\">"
        + "<label for=\"confirm-password\">Repeat the password</label>"
        + "<input id=\"confirm-password\" type=\"password\" autocomplete=\"new-password\" required aria-describedby=\"rules\">"
        + "<ul id=\"rules\" aria-label=\"Password requirements\"></ul>"
        + "<input id=\"resetToken\" name=\"resetToken\" type=\"hidden\">"
        + "<button type=\"submit\" id=\"login-submit\" class=\"btn\" disabled>Save password</button></form>"
        + "<script src=\"/login/core/auth.min.js\"></script>";
  }

  /** The password expires soon: change it now (yes-reset) or continue (no-reset). */
  static String expiring(String expires)
  {
    String when = expires == null ? "" : expires;
    return "<p>Your password expires on <strong>" + Html.esc(when) + "</strong>. Do you want to change it now?</p>"
        + "<form method=\"post\" action=\"/j_security_check\" class=\"form\">"
        + "<button type=\"submit\" name=\"yes-reset\" value=\"yes\" class=\"btn\">Change it now</button>"
        + "<button type=\"submit\" name=\"no-reset\" value=\"no\" class=\"btn secondary\">Later</button></form>";
  }
}
