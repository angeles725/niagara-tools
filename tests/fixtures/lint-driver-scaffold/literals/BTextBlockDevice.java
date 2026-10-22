// A text block holding braces and an escaped quote before the wizard stub.
package com.example.literals;

public class BTextBlockDevice extends BNDevice
{
  private static final String JSON = """
      { "open": "{", "q": "\"" }
      """;

  public void doPing()
  {
    pingOk();
  }
}
