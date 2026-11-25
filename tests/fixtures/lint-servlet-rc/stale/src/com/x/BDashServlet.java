package com.x;
import javax.baja.web.*;
public class BDashServlet extends BWebServlet {
  public String getServletName() { return "dash"; }
  public void doGet(WebOp op) throws Exception {
    op.getResponse().setHeader("Cache-Control", "public, max-age=3600");
  }
  void asset(WebOp op) {
    op.getResponse().setHeader("Cache-Control", op.getRequest().getParameter("v") != null ? "public, max-age=3600" : "no-cache");
  }
}
