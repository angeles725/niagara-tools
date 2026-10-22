package com.x;
import javax.baja.web.*;
import java.io.PrintWriter;
public class BInfoServlet extends BWebServlet {
  public void doGet(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    op.getResponse().setContentType("application/json");
    PrintWriter out = op.getResponse().getWriter(); // no header calls: the station providers apply -> INFO only
    out.print("{\"ok\":true}");
  }
}
