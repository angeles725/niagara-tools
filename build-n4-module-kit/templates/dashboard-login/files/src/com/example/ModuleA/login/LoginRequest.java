/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login;

import com.example.ModuleA.login.model.LoginRedirect;
import com.example.ModuleA.login.model.LoginView;
import com.tridium.authn.BDigestAuthenticationScheme;
import com.tridium.web.servlets.LoginSupport;
import com.tridium.web.session.NiagaraWebSession;
import com.tridium.web.session.WebSessionUtil;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Locale;
import javax.baja.authn.BAuthenticationScheme;
import javax.baja.authn.BPasswordAuthenticationScheme;
import javax.baja.sys.Sys;
import javax.baja.user.BPasswordStrength;
import javax.baja.user.BUserService;
import javax.servlet.http.Cookie;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

/**
 * LoginRequest — reads what Niagara's login servlets left in the request, its cookies and the web session
 * (the same sources web-rt BDefaultLoginTemplate reads) into {@link LoginView.Inputs}, and points the
 * session's {@code originURI} at the dashboard ({@link LoginRedirect}), which NiagaraAuthenticator redirects
 * to after the login. One-shot values (the failure-cause cookie, the session's resetError) are consumed here.
 */
final class LoginRequest
{
  static final String FAILURE_COOKIE = "niagara_failure_cause";

  private LoginRequest() {}

  /** Every login lands on the dashboard (a page of the dashboard itself is kept). */
  static void pointAtDashboard(HttpServletRequest req)
  {
    NiagaraWebSession session = WebSessionUtil.getSession(req, true);
    if (session != null) session.setAttribute("originURI", LoginRedirect.target(LoginSupport.getOriginUri(req)));
  }

  /** The inputs of {@link LoginView} for this request; consumes the failure-cause cookie. */
  static LoginView.Inputs inputs(HttpServletRequest req, HttpServletResponse resp, BAuthenticationScheme scheme)
  {
    NiagaraWebSession session = WebSessionUtil.getSession(req, false);
    String query = req.getQueryString();
    Object expires = attr(session, "passwordExpires");
    return new LoginView.Inputs()
        .username(LoginSupport.getRequestedUsername(req))
        // an unknown scheme (no such user, or a failed lookup) renders the password page like a real user's,
        // so the response never tells an anonymous client which usernames exist; SCRAM then fails as stock does
        .digest(scheme == null || scheme.getType().equals(BDigestAuthenticationScheme.TYPE))
        .forceReset(Boolean.TRUE.equals(attr(session, "forceReset")))
        .passwordExpires(expires instanceof Long ? (Long) expires : null)
        .networkUser(Boolean.TRUE.equals(attr(session, "networkUser")))
        .authFail(query != null && query.contains("auth=fail"))
        .failureCause(takeFailureCause(req, resp));
  }

  /** The user's authentication scheme, or null when unknown (no username or no such user). */
  static BAuthenticationScheme schemeOf(String username)
  {
    if (username == null) return null;
    try
    {
      BUserService users = (BUserService) Sys.getService(BUserService.TYPE);
      return users.getAuthenticationSchemeForUser(username);
    }
    catch (Exception e)
    {
      return null;
    }
  }

  /** The station's password strength rules as JSON, for the forced-change screen. */
  static String strengthJson(BAuthenticationScheme scheme)
  {
    if (!(scheme instanceof BPasswordAuthenticationScheme)) return "{}";
    BPasswordStrength s = ((BPasswordAuthenticationScheme) scheme).getGlobalPasswordConfiguration().getPasswordStrength();
    return "{\"minimumLength\":" + s.getMinimumLength() + ",\"minimumUpperCase\":" + s.getMinimumUpperCase()
        + ",\"minimumLowerCase\":" + s.getMinimumLowerCase() + ",\"minimumDigits\":" + s.getMinimumDigits()
        + ",\"minimumSpecial\":" + s.getMinimumSpecial() + ",\"maximumLength\":" + s.getMaximumLength() + "}";
  }

  /** The session's resetError (one-shot, as the stock template does), or null. */
  static String takeResetError(HttpServletRequest req)
  {
    NiagaraWebSession session = WebSessionUtil.getSession(req, false);
    if (session == null) return null;
    Object error = session.removeAttribute("resetError");
    return error == null ? null : "The password could not be changed: " + error;
  }

  /** "25 Nov 2026, 3:40 PM" in the station's time zone. */
  static String expiresText(Long millis)
  {
    if (millis == null) return "";
    return new SimpleDateFormat("d MMM yyyy, h:mm a", Locale.ENGLISH).format(new Date(millis.longValue()));
  }

  private static Object attr(NiagaraWebSession session, String name)
  {
    return session == null ? null : session.getAttribute(name);
  }

  private static String takeFailureCause(HttpServletRequest req, HttpServletResponse resp)
  {
    Cookie[] cookies = req.getCookies();
    if (cookies != null)
      for (Cookie c : cookies)
        if (FAILURE_COOKIE.equals(c.getName()))
        {
          Cookie gone = new Cookie(FAILURE_COOKIE, "");
          gone.setMaxAge(0);
          gone.setPath("/");
          resp.addCookie(gone);
          return c.getValue();
        }
    return req.getParameter("loginFailureCause");
  }
}
