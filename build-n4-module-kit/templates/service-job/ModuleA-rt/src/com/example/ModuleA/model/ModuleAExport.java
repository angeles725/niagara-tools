/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.model;

import java.util.Arrays;

/**
 * ModuleAExport — the export's decisions, zero-Baja so a plain JUnit test runs them without a station.
 */
public final class ModuleAExport
{
  private ModuleAExport() {}

  /** The names of a comma-separated list: each trimmed, empty entries dropped; null is an empty list. */
  public static String[] split(String csv)
  {
    if (csv == null) return new String[0];
    return Arrays.stream(csv.split(",")).map(String::trim).filter(s -> !s.isEmpty()).toArray(String[]::new);
  }

  /** One output line: the name as a quoted CSV field. */
  public static String line(String name)
  {
    return '"' + name.replace("\"", "\"\"") + '"';
  }

  /**
   * True for a plain file name: letters, digits, '.', '_' and '-', not starting with '.'. The name is an
   * operator-writable slot that becomes a station file ORD, so a path or an ORD separator is refused.
   */
  public static boolean isPlainFileName(String name)
  {
    return name != null && name.matches("[A-Za-z0-9_-][A-Za-z0-9._-]*");
  }
}
