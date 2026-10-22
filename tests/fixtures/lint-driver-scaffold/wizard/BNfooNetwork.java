// Shape of the NDriver wizard output (devkit-wb.jar!gradle/ndriver/BNfooNetwork.java.vm:200-205).
package com.example.nfoo;

public class BNfooNetwork extends BNNetwork
{
  public BOrd doSubmitDiscoveryJob(BNfooDeviceDiscoveryPreferences prefs, Context cx)
  {
    BNfooLearnDevicesJob job = new BNfooLearnDevicesJob(this, prefs);
    return job.submit(null);
  }
}
