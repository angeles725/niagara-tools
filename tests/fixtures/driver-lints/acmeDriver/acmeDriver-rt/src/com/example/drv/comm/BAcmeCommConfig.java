// Synthetic NDriver comm config for the kit's driver lints (G67); not a working driver.
package com.example.drv.comm;

import com.tridium.ndriver.datatypes.BTcpCommConfig;

public class BAcmeCommConfig
  extends BTcpCommConfig
{
  // One request at a time when the protocol carries no transaction tag:
  // public int getMaxOutstandingTransactions() { return 1; }
}
