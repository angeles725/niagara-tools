/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import com.example.ModuleA.login.model.LoginView.Screen;
import org.junit.Test;

/** The login page keeps Niagara's form contract on every screen, serves only its own assets and escapes what it echoes. */
public class LoginPageTest
{
  private static String page(Screen s) { return LoginPage.render(new LoginPage.Model(s).username("operator")); }

  private static void has(String html, String part) { assertTrue("missing: " + part, html.contains(part)); }

  @Test
  public void everyScreen_usesOurAssetsAndNoRemoteUrl()
  {
    for (Screen s : new Screen[] { Screen.USERNAME, Screen.PASSWORD, Screen.RESET, Screen.EXPIRING })
    {
      String html = page(s);
      has(html, "href=\"/login/modulea-login/login.css\"");
      has(html, "src=\"/login/modulea-login/login.js\"");
      has(html, "name=\"viewport\"");
      assertFalse("no remote assets", html.contains("http://") || html.contains("https://"));
    }
  }

  @Test
  public void usernameStep_postsJUsernameToLogin()
  {
    String html = page(Screen.USERNAME);
    has(html, "action=\"/login\"");
    has(html, "name=\"j_username\"");
  }

  @Test
  public void passwordStep_runsScramThenJSecurityCheck()
  {
    String html = page(Screen.PASSWORD);
    has(html, "action=\"/j_security_check\"");
    has(html, "id=\"login-form\"");
    has(html, "name=\"j_password\"");
    has(html, "value=\"operator\"");
    has(html, "src=\"/login/core/auth.min.js\"");
  }

  @Test
  public void resetStep_sendsOnlyAResetToken()
  {
    String html = LoginPage.render(new LoginPage.Model(Screen.RESET).username("operator").strengthJson("{\"minimumLength\":8}"));
    has(html, "id=\"reset-form\"");
    has(html, "name=\"resetToken\"");
    has(html, "data-rules=\"{&quot;minimumLength&quot;:8}\"");
    assertFalse("the new password must not have a name", html.contains("name=\"new-password\""));
  }

  @Test
  public void expiringStep_offersYesResetAndNoReset()
  {
    String html = LoginPage.render(new LoginPage.Model(Screen.EXPIRING).expires("25 Nov 2026"));
    has(html, "name=\"yes-reset\"");
    has(html, "name=\"no-reset\"");
    has(html, "25 Nov 2026");
  }

  @Test
  public void echoedText_isEscaped()
  {
    String html = LoginPage.render(new LoginPage.Model(Screen.PASSWORD).username("\"><script>x</script>").error("<b>bad</b>"));
    assertFalse(html.contains("<script>x</script>"));
    assertFalse(html.contains("<b>bad</b>"));
    has(html, "&lt;b&gt;bad&lt;/b&gt;");
    assertTrue(Html.esc(null).isEmpty());
    assertTrue(Html.esc("a&b'c").equals("a&amp;b&#39;c"));
  }
}
