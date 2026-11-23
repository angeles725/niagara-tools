/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

/** Everything the login page echoes goes through Html.esc: the five characters that end a text or an attribute. */
public class HtmlTest
{
  @Test
  public void escapesTheFiveSpecialCharacters()
  {
    assertEquals("&amp;&lt;&gt;&quot;&#39;", Html.esc("&<>\"'"));
  }

  @Test
  public void nullAndPlainTextAreKept()
  {
    assertEquals("", Html.esc(null));
    assertEquals("operator 1", Html.esc("operator 1"));
  }
}
