// The class body never closes: method boundaries cannot be trusted.
package com.example.unbalanced;

public class BTruncatedDevice extends BNDevice
{
  public void doPing()
  {
    sendPing();
  }
