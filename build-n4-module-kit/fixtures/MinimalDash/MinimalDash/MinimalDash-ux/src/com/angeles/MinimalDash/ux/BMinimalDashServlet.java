/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash.ux;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Type;
import javax.baja.web.BWebServlet;
import javax.baja.web.WebOp;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.io.InputStream;
import java.util.function.Function;

/**
 * BMinimalDashServlet — minimal dashboard servlet scaffold.
 *
 * <p>A single BWebServlet that routes every request under "/minimaldash/" by
 * delegating to {@link MinimalDashDispatch#route} (a pure, WSL-testable function).
 * Static assets are served from the {@code rc/} bundle baked into the jar.
 * Drop one instance under station Services — it self-registers via
 * {@link #getServletName()}.</p>
 *
 * <p>Endpoints (under /minimaldash/):</p>
 * <ul>
 *   <li>GET /           → index.html (dashboard SPA)</li>
 *   <li>GET /index.html → index.html</li>
 *   <li>GET /api/status → status JSON (MinimalDashDispatch.Equipment)</li>
 *   <li>other /api/*    → 302 redirect or 404</li>
 *   <li>other paths     → static asset from rc/</li>
 * </ul>
 *
 * <p>All routing decisions are made in {@link MinimalDashDispatch#route};
 * this class is a thin instanceof adapter only (DUX1 pattern).
 * [ev: dashboard.md § ux — servlet + SPA; DUX1]</p>
 */
@NiagaraType
public class BMinimalDashServlet
  extends BWebServlet
{
  /*
   * Pre-slotomatic state: AUTO region absent.
   * Run build.sh to generate the slot fields + Type from @NiagaraType.
   * [ev: B793 §793.3 C1 — pre-slotomatic scaffold convention]
   */

  public BMinimalDashServlet() {}

  /** Mount prefix — servlet is reachable at /{prefix}/. */
  @Override
  public String getServletName() { return "minimaldash"; }

  // -------------------------------------------------------------------------
  // Request dispatch — thin adapter over the pure router (DUX1)
  // -------------------------------------------------------------------------

  @Override
  public void doService(WebOp op) throws Exception
  {
    HttpServletRequest req  = op.getRequest();
    HttpServletResponse rsp = op.getResponse();
    Function<String, String> headers = req::getHeader;
    Function<String, String> params  = req::getParameter;

    MinimalDashDispatch.RouteAction action =
        MinimalDashDispatch.route(req.getMethod(),
                                  req.getPathInfo(),
                                  headers, params);

    if (action instanceof MinimalDashDispatch.RouteAction.Redirect) {
      rsp.sendRedirect(((MinimalDashDispatch.RouteAction.Redirect) action).location);
    } else if (action instanceof MinimalDashDispatch.RouteAction.Equipment) {
      rsp.setContentType("application/json; charset=UTF-8");
      rsp.getWriter().write("{\"status\":\"ok\"}");
    } else if (action instanceof MinimalDashDispatch.RouteAction.StaticResource) {
      String path = ((MinimalDashDispatch.RouteAction.StaticResource) action).path;
      serveStatic(op, path);
    } else {
      // RouteAction.NotFound — 404
      rsp.sendError(HttpServletResponse.SC_NOT_FOUND);
    }
  }

  // -------------------------------------------------------------------------
  // Static resource serving — reads from rc/ inside the jar
  // -------------------------------------------------------------------------

  private void serveStatic(WebOp op, String path) throws IOException
  {
    InputStream in = getClass().getClassLoader().getResourceAsStream("rc/" + path);
    if (in == null) {
      op.getResponse().sendError(HttpServletResponse.SC_NOT_FOUND);
      return;
    }
    try {
      byte[] buf = new byte[8192];
      int n;
      while ((n = in.read(buf)) != -1) {
        op.getResponse().getOutputStream().write(buf, 0, n);
      }
    } finally {
      in.close();
    }
  }
}
