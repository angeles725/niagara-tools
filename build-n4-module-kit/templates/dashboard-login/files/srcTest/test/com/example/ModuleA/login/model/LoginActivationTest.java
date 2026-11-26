/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertSame;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

import com.example.ModuleA.login.model.LoginActivation.Action;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.logging.Handler;
import java.util.logging.Level;
import java.util.logging.LogRecord;
import java.util.logging.Logger;
import org.junit.Test;

/**
 * The start-time decision of BModuleAService: when WebService.loginTemplate is set to the module's login
 * template and when it is left alone. Every row is a way a station can arrive at start; a wrong SET would
 * replace someone else's login page, a wrong KEEP would leave a retired type (whose class is gone) in the slot.
 */
public class LoginActivationTest
{
  private static final String OWN = "ModuleA:ModuleALoginTemplate";
  private static final String OLD = "ModuleA:ModuleAOldLoginTemplate";
  private static final List<String> RETIRED = Arrays.asList(OLD);

  private static Action decide(String current, boolean applyLogin)
  {
    return LoginActivation.decide(current, applyLogin, OWN, RETIRED).action();
  }

  @Test
  public void emptyTemplate_isSet()
  {
    assertEquals(Action.SET, decide(null, true));
    assertEquals(Action.SET, decide("", true));
    assertEquals(Action.SET, decide("   ", true));
    assertEquals(Action.SET, decide("null", true));   // BTypeSpec.NULL encodes as "null"
  }

  @Test
  public void stockTemplate_isSet()
  {
    assertEquals(Action.SET, decide("web:DefaultLoginTemplate", true));
  }

  @Test
  public void retiredTypeOfThisModule_isSetAndTheMessageNamesIt()
  {
    LoginActivation.Decision d = LoginActivation.decide(OLD, true, OWN, RETIRED);
    assertEquals(Action.SET, d.action());
    assertTrue(d.message(), d.message().contains(OWN));
    assertTrue(d.message(), d.message().contains(OLD));
    assertEquals(OWN, d.target());
  }

  @Test
  public void noRetiredTypeYet_theDefaultListIsEmpty()
  {
    assertTrue(LoginActivation.RETIRED.isEmpty());
    assertEquals(Action.KEEP_CUSTOM, LoginActivation.decide(OLD, true, OWN).action());
  }

  @Test
  public void anotherCustomTemplate_isKeptAndLogged()
  {
    LoginActivation.Decision d = LoginActivation.decide("other:OtherLoginTemplate", true, OWN, RETIRED);
    assertEquals(Action.KEEP_CUSTOM, d.action());
    assertFalse(d.sets());
    assertTrue(d.message(), d.message().contains("other:OtherLoginTemplate"));
    assertTrue(d.message(), d.message().contains("untouched"));
  }

  @Test
  public void aTypeThatOnlyLooksLikeAKnownOne_isKept()
  {
    assertEquals(Action.KEEP_CUSTOM, decide(OLD + "2", true));
    assertEquals(Action.KEEP_CUSTOM, decide(OLD.toLowerCase(), true));
    assertEquals(Action.KEEP_CUSTOM, decide("Other:ModuleAOldLoginTemplate", true));
    assertEquals(Action.KEEP_CUSTOM, decide("web:LoginTemplate", true));
  }

  @Test
  public void ourOwnTemplateAlreadyInPlace_isNotRewritten()
  {
    LoginActivation.Decision d = LoginActivation.decide(OWN, true, OWN, RETIRED);
    assertEquals(Action.ALREADY, d.action());
    assertFalse("no write and no save at every start", d.sets());
  }

  @Test
  public void applyLoginFalse_leavesEverythingUntouched()
  {
    String[] slots = { null, "", "null", "web:DefaultLoginTemplate", OLD, "other:OtherLoginTemplate", OWN };
    for (String slot : slots)
    {
      LoginActivation.Decision d = LoginActivation.decide(slot, false, OWN, RETIRED);
      assertEquals("slot " + slot, Action.OFF, d.action());
      assertFalse("slot " + slot, d.sets());
      assertTrue(d.message(), d.message().contains("applyLogin"));
    }
  }

  @Test
  public void onlyTheSetActionWrites()
  {
    assertTrue(LoginActivation.decide(null, true, OWN, RETIRED).sets());
    assertTrue(LoginActivation.decide(OLD, true, OWN, RETIRED).sets());
    assertFalse(LoginActivation.decide("x:Y", true, OWN, RETIRED).sets());
    assertFalse(LoginActivation.decide(null, false, OWN, Collections.<String>emptyList()).sets());
    assertFalse(LoginActivation.decide(OWN, true, OWN, RETIRED).sets());
  }

  // ---- guarded: nothing the start path throws may escape stationStarted() ----

  private static final String FAILED = "login template activation failed; WebService.loginTemplate is unchanged";

  /** A type whose class initializer fails, as BModuleALoginTemplate does when INSTANCE follows TYPE. */
  private static final class BrokenType
  {
    static final Object TYPE;
    static
    {
      if (true) throw new IllegalStateException("INSTANCE field is null; insure getType() comes last");
      TYPE = null;
    }
  }

  private static final class Captured extends Handler
  {
    final List<LogRecord> records = new java.util.ArrayList<LogRecord>();

    @Override public void publish(LogRecord r) { records.add(r); }
    @Override public void flush() {}
    @Override public void close() {}
  }

  private static Logger logger(Captured into)
  {
    Logger log = Logger.getAnonymousLogger();
    log.setUseParentHandlers(false);
    log.addHandler(into);
    return log;
  }

  private static void assertLoggedOnce(Captured c, Throwable expected)
  {
    assertEquals(1, c.records.size());
    assertEquals(Level.WARNING, c.records.get(0).getLevel());
    assertEquals(FAILED, c.records.get(0).getMessage());
    assertSame(expected, c.records.get(0).getThrown());
  }

  @Test
  public void guarded_aStepThatSucceeds_passesItsAnswerThroughAndLogsNothing()
  {
    Captured c = new Captured();
    assertTrue(LoginActivation.guarded(logger(c), FAILED, () -> true));
    assertFalse(LoginActivation.guarded(logger(c), FAILED, () -> false));
    assertTrue(c.records.isEmpty());
  }

  @Test
  public void guarded_anException_isLoggedAndAnsweredWithFalse()
  {
    Captured c = new Captured();
    final Exception boom = new IllegalStateException("no web service");
    assertFalse(LoginActivation.guarded(logger(c), FAILED, () -> { throw boom; }));
    assertLoggedOnce(c, boom);
  }

  @Test
  public void guarded_anError_isLoggedAndAnsweredWithFalse()
  {
    // the template type's initializer fails (INSTANCE after TYPE, or a web-rt class missing): an Error
    Captured c = new Captured();
    final Error boom = new NoClassDefFoundError("com/tridium/web/BDefaultLoginTemplate");
    assertFalse(LoginActivation.guarded(logger(c), FAILED, () -> { throw boom; }));
    assertLoggedOnce(c, boom);
  }

  @Test
  public void guarded_aTypeWhoseInitializerFails_neverEscapesOnTheFirstOrTheSecondUse()
  {
    Captured c = new Captured();
    Logger log = logger(c);
    assertFalse(LoginActivation.guarded(log, FAILED, () -> BrokenType.TYPE != null));   // ExceptionInInitializerError
    assertFalse(LoginActivation.guarded(log, FAILED, () -> BrokenType.TYPE != null));   // NoClassDefFoundError
    assertEquals(2, c.records.size());
    assertTrue(c.records.get(0).getThrown() instanceof ExceptionInInitializerError);
    assertTrue(c.records.get(1).getThrown() instanceof NoClassDefFoundError);
  }

  @Test
  public void guarded_aVirtualMachineError_isRethrownNotSwallowed()
  {
    Captured c = new Captured();
    final VirtualMachineError boom = new OutOfMemoryError("heap");
    try
    {
      LoginActivation.guarded(logger(c), FAILED, () -> { throw boom; });
      fail("an OutOfMemoryError must propagate");
    }
    catch (OutOfMemoryError e)
    {
      assertSame(boom, e);
    }
    assertTrue(c.records.isEmpty());
  }
}
