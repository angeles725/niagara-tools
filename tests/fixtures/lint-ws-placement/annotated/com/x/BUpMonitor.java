package com.x;

// Shape of chihuahua BChiUpMonitor.ensureUps: creates each child into a local and places it on a grid.
public class BUpMonitor extends BComponent
{
  private void ensureUps()
  {
    int seq = 0;
    for (int i = 0; i < UP_DATA.length; i++)
    {
      String slotName = (String) UP_DATA[i][0];
      if (get(slotName) != null) { seq++; continue; }
      BUp up = new BUp();
      add(slotName, up);
      BComponent child = (BComponent) get(slotName);
      child.add("wsAnnotation", BWsAnnotation.make(3 + seq % 8 * 14, 3 + seq / 8 * 10, 8), Flags.READONLY);
      seq++;
    }
  }
}
