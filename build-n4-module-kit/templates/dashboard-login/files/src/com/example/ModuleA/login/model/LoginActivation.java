/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import java.util.Collections;
import java.util.List;

/**
 * LoginActivation — whether BModuleAService, when the station starts, sets {@code WebService.loginTemplate} to
 * this module's login template. It changes how every user logs in, so the rule is closed and minimal:
 * <ul>
 * <li>{@code applyLogin} false: {@link Action#OFF}, nothing is touched whatever the slot holds;</li>
 * <li>slot already our template: {@link Action#ALREADY} (no write, no save at every start);</li>
 * <li>slot empty ("null"), Niagara's stock {@value #STOCK}, or a retired type of this module (listed in
 *     {@link #RETIRED}: its class is gone, so Niagara would fail to load it at /login, HTTP 500):
 *     {@link Action#SET};</li>
 * <li>any other type: {@link Action#KEEP_CUSTOM}, never overwritten, only logged.</li>
 * </ul>
 * Type specs are compared as exact, case-sensitive text ("module:Type", BTypeSpec's own form). Pure Java: the
 * caller reads the slot as text and writes only on {@link Decision#sets()}.
 */
public final class LoginActivation
{
  /** Niagara's own login template, as an explicit type spec. */
  public static final String STOCK = "web:DefaultLoginTemplate";

  /**
   * Type specs of login templates THIS module used to ship and no longer does ("ModuleA:OldLoginTemplate").
   * Empty until a login type of the module is renamed or removed: add its spec here in that change.
   */
  public static final List<String> RETIRED = Collections.<String>emptyList();

  /** What to do with the slot. */
  public enum Action { SET, ALREADY, KEEP_CUSTOM, OFF }

  /** The decision, its log line and the spec to write. */
  public static final class Decision
  {
    private final Action action;
    private final String message;
    private final String target;

    Decision(Action action, String message, String target)
    {
      this.action = action;
      this.message = message;
      this.target = target;
    }

    public Action action() { return action; }

    /** True only when the slot must be written (and the station saved). */
    public boolean sets() { return action == Action.SET; }

    /** One line for the station log. */
    public String message() { return message; }

    /** The type spec to write when {@link #sets()}. */
    public String target() { return target; }
  }

  private LoginActivation() {}

  /**
   * @param currentSpec WebService.loginTemplate as "module:Type", or null/blank/"null" when empty
   * @param applyLogin  the BModuleAService.applyLogin switch
   * @param ownSpec     this module's login template type spec ("ModuleA:ModuleALoginTemplate")
   */
  public static Decision decide(String currentSpec, boolean applyLogin, String ownSpec)
  {
    return decide(currentSpec, applyLogin, ownSpec, RETIRED);
  }

  /** As {@link #decide(String, boolean, String)} with the retired types given (a test seam). */
  static Decision decide(String currentSpec, boolean applyLogin, String ownSpec, List<String> retired)
  {
    String current = currentSpec == null ? "" : currentSpec.trim();
    boolean empty = current.isEmpty() || current.equals("null");
    String shown = empty ? "empty" : current;
    if (!applyLogin)
    {
      return new Decision(Action.OFF, "applyLogin is false: WebService.loginTemplate left as " + shown, ownSpec);
    }
    if (current.equals(ownSpec))
    {
      return new Decision(Action.ALREADY, "WebService.loginTemplate is already " + ownSpec, ownSpec);
    }
    if (empty || current.equals(STOCK) || retired.contains(current))
    {
      return new Decision(Action.SET,
          "WebService.loginTemplate set to " + ownSpec + " (was " + shown + "); the station is saved", ownSpec);
    }
    return new Decision(Action.KEEP_CUSTOM,
        "WebService.loginTemplate is " + current + ", a custom template set by someone else: left untouched "
            + "(clear it and restart to let ModuleAService apply the " + ownSpec + " login)", ownSpec);
  }
}
