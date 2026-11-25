/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package test.com.angeles.MinimalPan;

import static javax.baja.test.permissions.PermissionsScenario.withPermissions;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.security.PermissionException;
import javax.baja.sys.BDouble;
import javax.baja.sys.BComponent;
import javax.baja.sys.BasicContext;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import org.testng.Assert;
import org.testng.annotations.Test;

import com.angeles.MinimalPan.BMinimalPan;
import com.angeles.MinimalPan.BPanOffload;

/**
 * BMinimalPanPermissionsTest — RBAC against a real station: a user without write/invoke permission is refused,
 * an admin is not (types/moduleTest.md § TEST-G9).
 *
 * <p>{@code userPermissionsScenario(...)} is a {@code protected} method of {@code BTestNgStation}
 * ({@code javax/baja/test/BTestNgStation.java:749}, javadoc examples {@code :719-745}, test-wb vineflower docSource);
 * {@code actAsUser(...)} adds a role and a user for the steps, runs the consumer and removes both in a
 * {@code finally} ({@code javax/baja/test/permissions/PermissionsScenario.java:64-81}). The permission string is
 * the role's: lower-case {@code rwi} is operator read/write/invoke, upper-case {@code RWI} admin ({@code
 * BTestNgStation.java:450-451} defines the stock admin as {@code rwiRWI}). The call is made as that user by
 * passing {@code new BasicContext(user)}: a call without it is not a call as the user
 * (course {@code BUserSchemeTest.java:75-80}).</p>
 *
 * <p>The checks live in the invoke/set path: {@code ComponentSlotMap.invoke} calls {@code user.checkInvoke} before
 * it posts an ASYNC action ({@code com/tridium/sys/schema/ComponentSlotMap.java:1320-1325}), and
 * {@code ComplexSlotMap.set} calls {@code user.checkWrite} ({@code :657-666}); both throw
 * {@code javax.baja.security.PermissionException} ({@code BUser.java:1659-1685}).</p>
 */
@NiagaraType
@Test(groups = { "MinimalPan" })
public class BMinimalPanPermissionsTest
  extends BMinimalPanStationTestBase
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  @Test
  public void userWithoutWriteOrInvokePermissionIsRefused()
    throws InterruptedException
  {
    BMinimalPan pan = mount("pan", new BMinimalPan());
    BPanOffload offload = mount("offload", new BPanOffload());

    userPermissionsScenario(
      withPermissions("r").onOrd(ordOf(pan)),
      withPermissions("r").onOrd(ordOf(offload))
    ).actAsUser(user -> {
      BasicContext cx = new BasicContext(user);

      try
      {
        pan.set("setpoint", BDouble.make(5d), cx);
        Assert.fail("a user with read-only permission wrote the setpoint");
      }
      catch (PermissionException expected)
      {
        // refused: the write never happened
      }
      try
      {
        offload.invoke(offload.getAction("execute"), null, cx);
        Assert.fail("a user with read-only permission invoked execute");
      }
      catch (PermissionException expected)
      {
        // refused before post(): nothing was handed to the executor
      }

      Assert.assertEquals(pan.getSetpoint(), 0d, 0d, "the refused write changed the setpoint");
      Assert.assertTrue(holds(() -> offload.getRunCount() == 0, HOLD_MS), "the refused invoke ran the action");
    });
  }

  @Test
  public void adminCanWriteAndInvoke()
    throws InterruptedException
  {
    BMinimalPan pan = mount("pan", new BMinimalPan());
    BPanOffload offload = mount("offload", new BPanOffload());

    userPermissionsScenario(
      withPermissions("rwiRWI").onOrd(ordOf(pan)),
      withPermissions("rwiRWI").onOrd(ordOf(offload))
    ).actAsUser(user -> {
      BasicContext cx = new BasicContext(user);

      pan.set("setpoint", BDouble.make(5d), cx);
      Assert.assertEquals(pan.getSetpoint(), 5d, 0d);

      offload.invoke(offload.getAction("execute"), null, cx);
      // Wait inside the consumer: the executor thread checks the user's permission again, and actAsUser removes
      // the user when the consumer returns.
      Assert.assertTrue(await(() -> offload.getRunCount() == 1, WAIT_MS),
        "the admin's execute never ran, state=" + offload.getState());
    });
  }

  ////////////////////////////////////////////////////////////////
  // Support
  ////////////////////////////////////////////////////////////////

  /** The ORD of a component mounted directly under the station, in the form the javadoc examples use. */
  private static String ordOf(BComponent mounted)
  {
    return "station:|slot:/" + mounted.getName();
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  private static final long WAIT_MS = 5_000L;
  private static final long HOLD_MS = 300L;
}
