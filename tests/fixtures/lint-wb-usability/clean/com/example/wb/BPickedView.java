package com.example.wb;

import javax.baja.naming.BOrd;
import javax.baja.workbench.mgr.BComponentChooser;
import javax.baja.workbench.view.BWbComponentView;

/** The same view, with the station-component picker in its Add flow. */
public class BPickedView extends BWbComponentView {
  private BOrd target = BOrd.NULL;

  void pick() {
    BComponentChooser.prompt(root, path, display, select);
  }
}
