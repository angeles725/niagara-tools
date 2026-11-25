/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA;

import java.io.OutputStream;
import java.nio.charset.StandardCharsets;

import javax.baja.file.BIFile;
import javax.baja.file.BajaFileUtil;
import javax.baja.job.BSimpleJob;
import javax.baja.job.JobCancelException;
import javax.baja.naming.BOrd;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.Context;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import com.example.ModuleA.model.ModuleAExport;

/**
 * BModuleAExportJob — writes the service's names to a station file, one line per name (types/logic-authoring.md
 * § Job contract). {@code run} executes on the job's own thread; the job bar shows its progress and log.
 */
@NiagaraType
public final class BModuleAExportJob
  extends BSimpleJob
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  /** Type loading needs the public no-arg constructor. */
  public BModuleAExportJob() {}

  /** Copies the service's slots now: the job thread never reads them live, so an edit mid-run changes nothing. */
  public BModuleAExportJob(BModuleAService service)
  {
    this.fileName = service.getFileName();
    this.names = ModuleAExport.split(service.getPointNames());
  }

  @Override
  public void run(Context cx)
    throws Exception
  {
    if (!ModuleAExport.isPlainFileName(fileName))
      throw new IllegalArgumentException("fileName must be a plain file name: " + fileName);
    StringBuilder out = new StringBuilder();
    for (int i = 0; i < names.length; i++)
    {
      if (!isAlive()) throw new JobCancelException();   // cancel is cooperative: check on every pass
      out.append(ModuleAExport.line(names[i])).append('\n');
      log().message(names[i]);
      progress((int)((i + 1) * 100L / names.length));
    }
    BIFile file = BajaFileUtil.createFileToOverwrite(BOrd.make("file:^" + fileName), cx);
    try (OutputStream os = file.getOutputStream())
    {
      os.write(out.toString().getBytes(StandardCharsets.UTF_8));
    }
  }

  private String fileName = "";
  private String[] names = new String[0];
}
