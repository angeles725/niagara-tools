/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.model;

import org.junit.Test;
import static org.junit.Assert.*;

/** Pure JUnit test for {@link ModuleAExport}; run it with {@code toolbelt/run-pure-test.sh}. */
public class ModuleAExportTest
{
  @Test
  public void splitTrimsAndDropsEmptyEntries()
  {
    assertArrayEquals(new String[] { "a", "b c" }, ModuleAExport.split(" a, ,b c ,"));
    assertEquals(0, ModuleAExport.split(null).length);
    assertEquals(0, ModuleAExport.split("").length);
  }

  @Test
  public void lineQuotesTheField()
  {
    assertEquals("\"say \"\"hi\"\"\"", ModuleAExport.line("say \"hi\""));
  }

  @Test
  public void plainFileNamesPassAndPathsAreRefused()
  {
    assertTrue(ModuleAExport.isPlainFileName("export-1.csv"));
    for (String bad : new String[] { null, "", ".hidden", "../x.csv", "a/b.csv", "a b.csv", "x|y" })
      assertFalse("refused: " + bad, ModuleAExport.isPlainFileName(bad));
  }
}
