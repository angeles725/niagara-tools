/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDriver;

import javax.baja.control.BControlPoint;
import javax.baja.control.BNumericPoint;
import javax.baja.control.BNumericWritable;
import javax.baja.nre.annotations.NiagaraProperty;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.registry.TypeInfo;
import javax.baja.sys.BComponent;
import javax.baja.sys.Flags;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.tridium.ndriver.discover.BNPointDiscoveryLeaf;

/**
 * BMinimalDriverPointDiscoveryLeaf — one device register offered by point discovery. The manager adds
 * it as a numeric point (writable first: the best type is the first entry, § 1.11) whose proxy
 * extension reads this register.
 */
@NiagaraType
@NiagaraProperty(
  name = "register",
  type = "int",
  defaultValue = "0",
  flags = Flags.READONLY
)
public class BMinimalDriverPointDiscoveryLeaf
  extends BNPointDiscoveryLeaf
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. build.sh runs :slotomatic before :jar.
   */

  public BMinimalDriverPointDiscoveryLeaf() {}

  public BMinimalDriverPointDiscoveryLeaf(int register)
  {
    setRegister(register);
  }

  @Override
  public TypeInfo[] getValidDatabaseTypes()
  {
    return new TypeInfo[] { BNumericWritable.TYPE.getTypeInfo(), BNumericPoint.TYPE.getTypeInfo() };
  }

  @Override
  public String getDiscoveryName()
  {
    return "Register" + getRegister();
  }

  @Override
  public void updateTarget(BComponent target)
  {
    BMinimalDriverProxyExt ext = proxyExtOf(target);
    if (ext != null) ext.setRegister(getRegister());
  }

  /** A point whose proxy already reads this register is greyed out. */
  @Override
  public boolean isExisting(BComponent target)
  {
    BMinimalDriverProxyExt ext = proxyExtOf(target);
    return ext != null && ext.getRegister() == getRegister();
  }

  private static BMinimalDriverProxyExt proxyExtOf(BComponent target)
  {
    if (!(target instanceof BControlPoint)) return null;
    Object ext = ((BControlPoint)target).getProxyExt();
    return ext instanceof BMinimalDriverProxyExt ? (BMinimalDriverProxyExt)ext : null;
  }
}
