/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

/** Every login lands on the dashboard; a page of the dashboard is kept; nothing leaves the station. */
public class LoginRedirectTest
{
  private static final String HOME = LoginRedirect.DASHBOARD;

  @Test
  public void anyOtherOrigin_landsOnTheDashboard()
  {
    assertEquals(HOME, LoginRedirect.target(null));
    assertEquals(HOME, LoginRedirect.target(""));
    assertEquals(HOME, LoginRedirect.target("/ord?station:|slot:/"));
    assertEquals(HOME, LoginRedirect.target("http://elsewhere.invalid/modulea/"));
  }

  @Test
  public void aDashboardPage_isKept()
  {
    assertEquals("/modulea/index.html", LoginRedirect.target("/modulea/index.html"));
    assertEquals(HOME, LoginRedirect.target("/modulea"));
    assertEquals(HOME, LoginRedirect.target("/modulea/"));
  }

  @Test
  public void aLookAlikePrefix_isNotTheDashboard()
  {
    assertEquals(HOME, LoginRedirect.target("/moduleaother/page"));
  }

  @Test
  public void theTargetIsAlwaysStationRelative()
  {
    assertEquals('/', HOME.charAt(0));
  }

  @Test
  public void dotSegmentsSlashesAndControlChars_landOnTheDashboard()
  {
    for (String o : new String[] {"/modulea/../login", "/modulea//evil.invalid", "/modulea/\\evil.invalid",
        "/modulea/%2e%2e/x", "/modulea/a\r\nSet-Cookie: x=1", "/modulea/a b"})
    {
      assertEquals(o, HOME, LoginRedirect.target(o));
    }
    assertEquals("/modulea/index.html?demo=1", LoginRedirect.target("/modulea/index.html?demo=1"));
  }
}
