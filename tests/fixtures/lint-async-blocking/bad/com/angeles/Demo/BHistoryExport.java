package com.angeles.Demo;

import javax.baja.nre.annotations.NiagaraAction;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.naming.BOrd;
import javax.baja.sys.BComponent;
import javax.baja.sys.Context;
import javax.baja.sys.Flags;

// Certification-exam shape: an ASYNC export that resolves a history BQL ord and writes a file on the engine thread.
@NiagaraType
@NiagaraAction(name = "export", flags = Flags.ASYNC)
public class BHistoryExport extends BComponent
{
  public void doExport(Context cx) throws Exception
  {
    BOrd ord = BOrd.make("history:/demo/Log|bql:select *");
    Object table = ord.get(this, cx);
    java.io.OutputStream out = file.getOutputStream();
  }
}
