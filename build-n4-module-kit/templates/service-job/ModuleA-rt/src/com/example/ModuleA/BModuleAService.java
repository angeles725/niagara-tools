/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Action;
import javax.baja.sys.BAbstractService;
import javax.baja.sys.Flags;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;
import javax.baja.util.BIRestrictedComponent;

/**
 * BModuleAService — a station service that exports a list of names to a file in a background job
 * (types/service.md, types/logic-authoring.md § Background jobs).
 *
 * <p>Registered under {@code Services} by placement; BIRestrictedComponent keeps it to one instance, directly
 * under Services (its default check, a framework diagnostic on a mis-drop). {@code exportNow} is OPERATOR, so an
 * operator or a link can start it; the work runs in {@link BModuleAExportJob}, never on the thread that invoked
 * the action.</p>
 */
@NiagaraType
@NiagaraProperty(name = "fileName", type = "String", defaultValue = "ModuleA-export.csv", flags = Flags.OPERATOR)
@NiagaraProperty(name = "pointNames", type = "String", defaultValue = "", flags = Flags.OPERATOR)
@NiagaraAction(name = "exportNow", flags = Flags.OPERATOR)
public final class BModuleAService
  extends BAbstractService
  implements BIRestrictedComponent
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  public BModuleAService() {}

  /** Mandatory and never empty: the service registers under each type returned, and a lookup finds it by one. */
  @Override
  public Type[] getServiceTypes()
  {
    return new Type[] { TYPE };
  }

  /**
   * Starts one export. The job is submitted with a null Context: {@code submit(cx)} with a user context needs
   * admin invoke on the JobService, which an operator lacks. The OPERATOR flag on this action is the permission
   * check that applies.
   */
  public void doExportNow()
  {
    if (!getEnabled()) return;
    new BModuleAExportJob(this).submit(null);
  }
}
