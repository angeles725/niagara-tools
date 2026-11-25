/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package test.com.angeles.MinimalPan;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import org.testng.Assert;
import org.testng.annotations.Test;

import com.angeles.MinimalPan.BPanOffload;

/**
 * BMinimalPanOffloadTest — {@link BPanOffload}'s ASYNC action runs on the module's named executor thread, and a
 * component that is not running refuses the post (types/actions.md § 2, the sanctioned offload).
 */
@NiagaraType
@Test(groups = { "MinimalPan" })
public class BMinimalPanOffloadTest
  extends BMinimalPanStationTestBase
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  @Test
  public void executeRunsOnTheNamedExecutorThreadNotTheCaller()
  {
    BPanOffload offload = mount("offload", new BPanOffload());

    offload.invoke(offload.getAction("execute"), null, null);

    Assert.assertTrue(await(() -> offload.getRunCount() == 1, WAIT_MS),
      "execute never ran, runCount=" + offload.getRunCount() + " state=" + offload.getState());
    Assert.assertTrue(offload.getLastThread().startsWith(BPanOffload.THREAD_PREFIX),
      "ran on '" + offload.getLastThread() + "', not on a '" + BPanOffload.THREAD_PREFIX + "N' thread");
    Assert.assertNotEquals(offload.getLastThread(), Thread.currentThread().getName());
    Assert.assertEquals(offload.getState(), "done");
  }

  @Test
  public void aComponentThatIsNotRunningRefusesThePost()
  {
    BPanOffload neverMounted = new BPanOffload();
    Assert.assertNull(neverMounted.post(neverMounted.getAction("execute"), null, null));

    BPanOffload removed = mount("offload", new BPanOffload());
    stationHandler.getStation().remove(removed);
    Assert.assertFalse(removed.isRunning());
    Assert.assertNull(removed.post(removed.getAction("execute"), null, null));

    Assert.assertEquals(neverMounted.getRunCount(), 0, "a never-started component ran the action");
    Assert.assertEquals(removed.getRunCount(), 0, "a stopped component ran the action");
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  private static final long WAIT_MS = 5_000L;
}
