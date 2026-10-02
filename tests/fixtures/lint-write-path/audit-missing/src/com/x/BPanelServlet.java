package com.x;
public class BPanelServlet extends BWebServlet {
  // An audit() mention inside a comment is NOT an audit call.
  public void doPost(WebOp op) throws Exception {
    BComponent target = resolve(op);
    target.set("setpoint", BDouble.make(parse(op)), cx(op));
    String trail = getAuditLog();   // reading the audit store is not recording a write
    respond(op, "{\"ok\":true}");
  }
}
