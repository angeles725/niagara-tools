/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash.ux;

import org.junit.Test;
import static org.junit.Assert.*;

import java.util.HashMap;
import java.util.Map;
import java.util.function.Function;

/**
 * Pure-Java unit tests for {@link MinimalDashDispatch#route} — the servlet router.
 *
 * <p>MinimalDashDispatch takes only plain-Java inputs (method, path, header/param
 * lookup functions), so every routing decision is testable in plain JUnit in
 * WSL with zero Niagara station (DUX1 pure-router pattern; mirrors
 * DashboardPan's DashboardDispatchTest).
 * [ev: dashboard.md § DUX1 — ux servlet testable seam]</p>
 *
 * <p>Coverage:</p>
 * <ul>
 *   <li>path-traversal guard (".." / "\\" / NUL) → NotFound (404)</li>
 *   <li>missing X-Requested-With on /api/* → Redirect (302)</li>
 *   <li>unknown /api/* with XHR → NotFound (404)</li>
 *   <li>GET "/" → StaticResource("index.html")</li>
 *   <li>GET /api/status + XHR → Equipment</li>
 *   <li>a static asset path → StaticResource with the stripped path</li>
 * </ul>
 */
public class MinimalDashDispatchTest
{
  private static Function<String, String> xhrHeaders()
  {
    Map<String, String> h = new HashMap<String, String>();
    h.put("X-Requested-With", "XMLHttpRequest");
    return h::get;
  }

  private static Function<String, String> noHeaders() { return k -> null; }
  private static Function<String, String> noParams()  { return k -> null; }

  // --- Path-traversal guard → NotFound (404) --------------------------------

  @Test
  public void route_traversalDotDot_returnsNotFound()
  {
    MinimalDashDispatch.RouteAction a =
        MinimalDashDispatch.route("GET", "/../../etc/passwd", noHeaders(), noParams());
    assertTrue("'..' path must be NotFound",
        a instanceof MinimalDashDispatch.RouteAction.NotFound);
  }

  @Test
  public void route_traversalBackslash_returnsNotFound()
  {
    MinimalDashDispatch.RouteAction a =
        MinimalDashDispatch.route("GET", "/foo\\bar", noHeaders(), noParams());
    assertTrue("backslash path must be NotFound",
        a instanceof MinimalDashDispatch.RouteAction.NotFound);
  }

  // --- XHR guard on /api/* → Redirect (302) ---------------------------------

  @Test
  public void route_apiWithoutXhr_returnsRedirect()
  {
    MinimalDashDispatch.RouteAction a =
        MinimalDashDispatch.route("GET", "/api/status", noHeaders(), noParams());
    assertTrue("/api/* without XHR must be Redirect",
        a instanceof MinimalDashDispatch.RouteAction.Redirect);
    assertEquals(MinimalDashDispatch.REDIRECT_HOME,
        ((MinimalDashDispatch.RouteAction.Redirect) a).location);
  }

  // --- Unknown /api/* with XHR → NotFound (404) -----------------------------

  @Test
  public void route_unknownApiWithXhr_returnsNotFound()
  {
    MinimalDashDispatch.RouteAction a =
        MinimalDashDispatch.route("GET", "/api/unknown", xhrHeaders(), noParams());
    assertTrue("unknown /api/* must be NotFound",
        a instanceof MinimalDashDispatch.RouteAction.NotFound);
  }

  // --- GET "/" → StaticResource("index.html") --------------------------------

  @Test
  public void route_rootPath_returnsIndexHtml()
  {
    MinimalDashDispatch.RouteAction a =
        MinimalDashDispatch.route("GET", "/", noHeaders(), noParams());
    assertTrue("'/' must be a StaticResource",
        a instanceof MinimalDashDispatch.RouteAction.StaticResource);
    assertEquals("index.html",
        ((MinimalDashDispatch.RouteAction.StaticResource) a).path);
  }

  // --- GET /api/status + XHR → Equipment ------------------------------------

  @Test
  public void route_apiStatusWithXhr_returnsEquipment()
  {
    MinimalDashDispatch.RouteAction a =
        MinimalDashDispatch.route("GET", "/api/status", xhrHeaders(), noParams());
    assertTrue("/api/status + XHR must be Equipment",
        a instanceof MinimalDashDispatch.RouteAction.Equipment);
  }

  // --- Static asset path → StaticResource -----------------------------------

  @Test
  public void route_staticAsset_returnsStrippedPath()
  {
    MinimalDashDispatch.RouteAction a =
        MinimalDashDispatch.route("GET", "/css/main.css", noHeaders(), noParams());
    assertTrue("static path must be StaticResource",
        a instanceof MinimalDashDispatch.RouteAction.StaticResource);
    assertEquals("css/main.css",
        ((MinimalDashDispatch.RouteAction.StaticResource) a).path);
  }
}
