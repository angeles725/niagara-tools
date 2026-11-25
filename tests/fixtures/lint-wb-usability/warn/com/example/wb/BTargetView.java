package com.example.wb;

import javax.baja.naming.BOrd;
import javax.baja.workbench.view.BWbComponentView;

/** A config view whose only way to set the target is to type its ORD. */
public class BTargetView extends BWbComponentView {
  private BOrd target = BOrd.NULL;

  public void doLoadValue() {
    // load the target into a text field
  }
}
