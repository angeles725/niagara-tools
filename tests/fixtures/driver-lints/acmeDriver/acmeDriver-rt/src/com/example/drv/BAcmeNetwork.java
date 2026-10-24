// Synthetic NDriver network for the kit's driver lints (G65, G66, G67); not a working driver.
package com.example.drv;

import com.tridium.ndriver.BNNetwork;
import com.tridium.ndriver.comm.NComm;
import com.tridium.driver.util.DrUtil;

public class BAcmeNetwork
  extends BNNetwork
{
  private final java.util.Map<String, Object> pending = new java.util.HashMap<>();

  public NComm comm() { return (NComm) DrUtil.getParent(this, NComm.class); }
}
