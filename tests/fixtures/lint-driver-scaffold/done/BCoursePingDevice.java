// Course reference shape: pingOk() is guarded by the protocol's answer, so it is not a stub.
package com.example.course;

public class BCoursePingDevice extends BNDevice
{
  public void doPing() {
    PingResponse pingResponse = sendPing();
    if (pingResponse.getPingSuccessful()) { pingOk(); }
    else { pingFail(pingResponse.getFailureCause()); }
  }
}
