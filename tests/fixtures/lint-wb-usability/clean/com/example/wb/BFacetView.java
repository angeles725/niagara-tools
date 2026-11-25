package com.example.wb;

import javax.baja.naming.BOrd;
import javax.baja.workbench.view.BWbComponentView;

public class BFacetView extends BWbComponentView {
  private BOrd target = BOrd.NULL;

  BFacets facets = BFacets.make("targetType", BString.make("baja:Component"));
}
