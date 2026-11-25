/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalPan;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.logging.Level;
import java.util.logging.Logger;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Action;
import javax.baja.sys.BComponent;
import javax.baja.sys.BValue;
import javax.baja.sys.Context;
import javax.baja.sys.Flags;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;
import javax.baja.util.IFuture;
import javax.baja.util.Invocation;

/**
 * BPanOffload — the sanctioned way to run slow work for an ASYNC action: {@code post()} hands the invocation to
 * the module's own single-thread executor instead of the shared {@code Nre:Engine} thread (types/actions.md § 2).
 *
 * <p>Shape of the course solution (Lab 9 {@code BExecutorServiceExecutor.java:46-70}: executor created in
 * {@code started()}, {@code shutdownNow()} in {@code stopped()}, {@code post()} override), with the three
 * corrections the kit requires and the lab does not make:</p>
 * <ul>
 *   <li>the work is {@code new Invocation(this, action, arg, cx)}, run by the executor through
 *       {@code BComponent.doInvoke}, never {@code this::doExecute} (the lab's {@code :66} skips the action pipeline:
 *       corpus B1253 §D6);</li>
 *   <li>a not-running guard: a component that is not running refuses the post (the lab's {@code :66} throws a
 *       NullPointerException before {@code started()} and after {@code stopped()}, which {@code invoke} prints and
 *       swallows: {@code ComponentSlotMap.java:1328-1334});</li>
 *   <li>a named thread factory, so the thread is identifiable in a thread dump.</li>
 * </ul>
 * <p>Creating the thread, {@code setDaemon} and {@code shutdownNow()} need {@code MANAGE_EXECUTION}
 * (module-permissions.xml, type="station"; Lab 9 {@code module-permissions.xml:25-28}).</p>
 *
 * <p>[ev: course Labs/Lab9 Multithreaded/BExecutorServiceExecutor.java:46-70]
 * [ev: code javax/baja/util/Invocation.java:28-48 (docSource vineflower baja)]
 * [ev: code javax/baja/sys/BComponent.java:1255-1259 (post), :1266-1270 (doInvoke)]
 * [ev: code javax/baja/sys/Flags.java:187 (ASYNC = 0x10)]</p>
 */
@NiagaraType
@NiagaraProperty(
  name = "state",
  type = "String",
  defaultValue = "idle",
  flags = Flags.TRANSIENT | Flags.READONLY | Flags.SUMMARY
)
@NiagaraProperty(
  name = "lastThread",
  type = "String",
  defaultValue = "",
  flags = Flags.TRANSIENT | Flags.READONLY | Flags.SUMMARY
)
@NiagaraProperty(
  name = "runCount",
  type = "int",
  defaultValue = "0",
  flags = Flags.TRANSIENT | Flags.READONLY | Flags.SUMMARY
)
@NiagaraAction(name = "execute", flags = Flags.ASYNC | Flags.OPERATOR)
public class BPanOffload
  extends BComponent
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  /** Prefix of the executor thread's name; the test reads it back from the {@code lastThread} slot. */
  public static final String THREAD_PREFIX = "MinimalPan-offload-";

  public BPanOffload() {}

  ////////////////////////////////////////////////////////////////
  // Lifecycle
  ////////////////////////////////////////////////////////////////

  /** Creates the module-owned executor (Lab 9 {@code BExecutorServiceExecutor.java:46-50}). */
  @Override
  public void started() throws Exception
  {
    super.started();
    executor = Executors.newSingleThreadExecutor(new NamedThreadFactory());
  }

  /**
   * Stops the executor so the work thread does not outlive the component (Lab 9 {@code :52-59}).
   * {@code shutdownNow()} needs MANAGE_EXECUTION: without it the denial is only logged and the thread leaks.
   */
  @Override
  public void stopped() throws Exception
  {
    ExecutorService stopping = executor;
    executor = null;
    if (stopping != null) stopping.shutdownNow();
    super.stopped();
  }

  ////////////////////////////////////////////////////////////////
  // Action
  ////////////////////////////////////////////////////////////////

  /**
   * Hands an ASYNC invocation to the module's executor and returns at once (the caller's thread is never held).
   * Always returns null, as {@code BComponent.post} documents.
   */
  @Override
  public IFuture post(Action action, BValue arg, Context cx)
  {
    if (action != execute) return super.post(action, arg, cx);
    ExecutorService running = executor;
    // Not-running guard: before started() and after stopped() there is no executor to take the work.
    if (!isRunning() || running == null) return null;
    try
    {
      running.execute(new Invocation(this, action, arg, cx));
    }
    catch (RejectedExecutionException e)
    {
      // Shut down between the guard and the call: report it, never let it escape from post().
      LOG.log(Level.WARNING, "execute refused: executor is shut down", e);
    }
    return null;
  }

  /**
   * The slow work. Runs on the named executor thread, entered through {@code doInvoke} so the action pipeline
   * (permission check on the Context's user, slot knobs, action event) still runs.
   */
  public void doExecute()
  {
    setState("running");
    setLastThread(Thread.currentThread().getName());
    setRunCount(getRunCount() + 1);
    setState("done");
  }

  ////////////////////////////////////////////////////////////////
  // Thread factory
  ////////////////////////////////////////////////////////////////

  /** Names the thread {@link #THREAD_PREFIX}{@code N} so a thread dump identifies the module. */
  private static final class NamedThreadFactory
    implements ThreadFactory
  {
    @Override
    public Thread newThread(Runnable r)
    {
      Thread thread = new Thread(r, THREAD_PREFIX + COUNTER.incrementAndGet());
      thread.setDaemon(true);
      return thread;
    }
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  private static final Logger LOG = Logger.getLogger("MinimalPan.offload");
  private static final AtomicInteger COUNTER = new AtomicInteger();

  /** The module-owned executor; non-null only between started() and stopped(). */
  private volatile ExecutorService executor;
}
