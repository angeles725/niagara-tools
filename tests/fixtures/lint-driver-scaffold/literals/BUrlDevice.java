// A '{' char literal and a "//" inside a string on the line before the wizard stub.
package com.example.literals;

public class BUrlDevice extends BNDevice
{
  private static final char OPEN = '{'; private static final String URL = "tcp://host";

  public void doPing()
  {
    pingOk();
  }
}
