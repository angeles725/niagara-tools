/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import static org.junit.Assert.assertTrue;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import org.junit.Test;

/**
 * Sys.loadType reads a type's static INSTANCE field: a class that declares INSTANCE after TYPE fails to load on
 * a station ("INSTANCE field is null; insure getType() comes last") and the login template never activates,
 * while the pure tests and the build pass. This reads the sources (zero-Baja, so it runs under run-pure-test.sh
 * from the -rt directory): every class under src that declares INSTANCE must declare it before TYPE. Slot-o-Matic
 * writes TYPE into its generated region: after the first :slotomatic run the region must sit below INSTANCE.
 */
public class TypeInitOrderTest
{
  private static final Pattern INSTANCE = Pattern.compile("static\\s+final\\s+\\w+\\s+INSTANCE\\s*=");
  private static final Pattern TYPE = Pattern.compile("static\\s+final\\s+Type\\s+TYPE\\s*=\\s*Sys\\.loadType");

  @Test
  public void instanceIsDeclaredBeforeType() throws IOException
  {
    List<String> wrong = new ArrayList<String>();
    int withInstance = 0;
    for (File f : javaFiles(new File("src")))
    {
      String s = new String(Files.readAllBytes(f.toPath()), StandardCharsets.UTF_8);
      Matcher i = INSTANCE.matcher(s);
      if (!i.find()) continue;
      withInstance++;
      Matcher t = TYPE.matcher(s);
      if (t.find() && i.start() > t.start()) wrong.add(f.getPath());
    }
    assertTrue("no class under src declares INSTANCE; the check would be vacuous", withInstance > 0);
    assertTrue("INSTANCE declared after TYPE in " + wrong, wrong.isEmpty());
  }

  private static List<File> javaFiles(File dir)
  {
    List<File> out = new ArrayList<File>();
    File[] kids = dir.listFiles();
    if (kids == null) return out;
    for (File k : kids)
    {
      if (k.isDirectory()) out.addAll(javaFiles(k));
      else if (k.getName().endsWith(".java")) out.add(k);
    }
    return out;
  }
}
