// Synthetic NDriver response for the kit's driver lints (G67); not a working driver.
package com.example.drv.message;

import com.tridium.ndriver.comm.NMessage;

public class AcmeResp extends NMessage
{
  public boolean isResponse() { return true; }
}
