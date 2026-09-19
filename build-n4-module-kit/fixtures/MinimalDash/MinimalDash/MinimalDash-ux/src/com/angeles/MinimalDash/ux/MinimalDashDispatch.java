/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash.ux;

import java.util.function.Function;

/**
 * Pure routing logic for {@link BMinimalDashServlet}.
 *
 * <p>This package-private final class is the ONLY place where routing decisions
 * are made. The servlet's {@code doService} calls {@link #route} and then executes
 * the returned {@link RouteAction} against the Niagara WebOp. Because it takes only
 * plain-Java inputs (method, path, header/param lookup functions) it is fully
 * WSL-unit-testable with zero Niagara dependency (DUX1 pure-router pattern).
 * [ev: dashboard.md § DUX1 — ux servlet testable seam]</p>
 *
 * <p>Routes (all under the servlet prefix "/minimaldash/"):</p>
 * <ul>
 *   <li>GET / or /index.html → StaticResource("index.html")</li>
 *   <li>GET /api/status + XHR → Equipment (status JSON)</li>
 *   <li>any /api/* without XHR header → Redirect to index.html</li>
 *   <li>unknown /api/* → NotFound</li>
 *   <li>any other path → StaticResource (rc/ asset)</li>
 * </ul>
 *
 * <p>Guards (in order): path-traversal ("..", "\\", NUL → NotFound); XHR guard
 * on /api/* (require X-Requested-With: XMLHttpRequest, else 302 Redirect);
 * unknown /api/* → NotFound; static fallback (/ → index.html).</p>
 */
final class MinimalDashDispatch
{
  private MinimalDashDispatch() {}

  /** Redirect target for /api/* requests missing the XHR header. */
  static final String REDIRECT_HOME = "/minimaldash/index.html";

  // =========================================================================
  // RouteAction — package-private nested hierarchy; callers instanceof-check.
  // =========================================================================

  abstract static class RouteAction
  {
    private RouteAction() {}

    /** Serve the status JSON (GET /api/status). */
    static final class Equipment extends RouteAction
    {
      static final Equipment INSTANCE = new Equipment();
      private Equipment() {}
    }

    /** Serve a static file from rc/ (index.html and other assets). */
    static final class StaticResource extends RouteAction
    {
      final String path;
      StaticResource(String path) { this.path = path; }
    }

    /** Redirect — used for /api/* requests without the XHR header (302). */
    static final class Redirect extends RouteAction
    {
      final String location;
      Redirect(String location) { this.location = location; }
    }

    /** 404 — path traversal, unknown API endpoint. */
    static final class NotFound extends RouteAction
    {
      final String reason;
      NotFound(String reason) { this.reason = reason; }
    }
  }

  // =========================================================================
  // route() — the single pure decision function
  // =========================================================================

  /**
   * Classifies one HTTP request into a {@link RouteAction}.
   *
   * @param method  HTTP method ("GET", "POST", …)
   * @param path    path relative to servlet mount (from {@code getPathInfo()});
   *                null is treated as "/"
   * @param headers header lookup function (returns null when absent)
   * @param params  query-param lookup function (returns null when absent)
   * @return the action the servlet should execute; never null
   */
  static RouteAction route(String method, String path,
      Function<String, String> headers,
      Function<String, String> params)
  {
    if (path == null || path.isEmpty()) path = "/";

    // Guard 1: path traversal — reject "..", "\\", NUL
    if (path.contains("..") || path.contains("\\") || path.contains("\0")) {
      return new RouteAction.NotFound("traversal");
    }

    // API endpoints
    if (path.startsWith("/api/")) {
      // Guard 2: XHR header required on all /api/* requests
      if (headers.apply("X-Requested-With") == null) {
        return new RouteAction.Redirect(REDIRECT_HOME);
      }
      if ("GET".equals(method) && "/api/status".equals(path)) {
        return RouteAction.Equipment.INSTANCE;
      }
      return new RouteAction.NotFound("unknown-api");
    }

    // Static fallback — serve index.html for bare "/" or "/index.html"
    if ("/".equals(path) || "/index.html".equals(path)) {
      return new RouteAction.StaticResource("index.html");
    }
    // Other static paths: strip leading slash
    String stripped = path.startsWith("/") ? path.substring(1) : path;
    return new RouteAction.StaticResource(stripped);
  }
}
