// A "/*" inside a string before the wizard stub must not open a block comment.
package com.example.literals;

public class BMarkerDevice extends BNDevice
{
  private static final String START = "/*";

  public void doPing()
  {
    pingOk();
  }
}
