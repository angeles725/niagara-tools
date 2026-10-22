// An annotation whose array argument holds braces, on the stub's own line: its braces are not a method body.
package com.example.annotated;

public class BArrayAnnotDevice extends BNDevice
{
  @SuppressWarnings({"unused", "rawtypes"}) public void doPing() {
    pingOk();
  }
}
