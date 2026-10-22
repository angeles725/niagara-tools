// A same-line annotation with arguments ahead of the wizard stub: the annotation is not the method name.
package com.example.annotated;

public class BSameLineDevice extends BNDevice
{
  @SuppressWarnings("unused") public void doPing()
  {
    pingOk();
  }
}
