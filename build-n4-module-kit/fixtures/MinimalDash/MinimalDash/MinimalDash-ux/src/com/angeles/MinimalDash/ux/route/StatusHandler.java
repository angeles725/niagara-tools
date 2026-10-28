/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash.ux.route;

/**
 * The GET /api/status route group's handler: the servlet routes (MinimalDashDispatch), this class answers.
 *
 * <p>One handler class per route group keeps the servlet a router (types/dashboard.md DUX1, lint-servlet
 * handler-size). The response's fields are the ux-rt contract's ({@code src/rc/js/contract.json}, route
 * {@code GET /api/status}); StatusHandlerTest reads that file, so a change here that drops a contract field
 * fails offline. Zero-Baja: it builds the body from plain values; the servlet writes it.</p>
 */
final class StatusHandler
{
  private StatusHandler() {}

  static final String CONTENT_TYPE = "application/json; charset=UTF-8";

  /** The GET /api/status body. */
  static String respond()
  {
    return "{\"status\":\"ok\"}";
  }
}
