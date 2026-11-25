package com.example.wb;

import javax.baja.naming.BOrd;
import javax.baja.workbench.view.BWbComponentView;

// lint-wb-usability: waived the commissioning wizard sets this ORD, no operator types it
public class BWaivedView extends BWbComponentView {
  private BOrd target = BOrd.NULL;
}
