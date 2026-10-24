// Synthetic NDriver device for the kit's driver lints (G65, G66, G67); not a working driver.
package com.example.drv;

import java.util.HashSet;
import java.util.Set;

import com.tridium.ndriver.BNDevice;

public class BAcmeDevice
  extends BNDevice
{
  static final int MAX_RETRIES = 3;

  public void subscribe(String id) { subscribed.add(id); }

  private static final Set<String> subscribed = new HashSet<>();
}
