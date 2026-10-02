package com.x;
public class BPanelServlet extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    BComponent target = resolve(op);
    target.set("setpoint", BDouble.make(parse(op)), cx(op));
    log("TODO: appendAudit( once the store exists");   // a string mention is not an audit call
    respond(op, "{\"ok\":true}");
  }
}
