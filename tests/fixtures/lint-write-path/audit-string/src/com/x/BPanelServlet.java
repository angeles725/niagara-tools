package com.x;
public class BPanelServlet extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    op.setContentType("*/*");   // the "/*" inside the literal does not open a block comment
    BComponent target = resolve(op); String base = "http://local/"; BValue old = target.get("setpoint");
    target.set("setpoint", BDouble.make(parse(op)), cx(op));
    svc.appendAudit(buildAuditEntry(user(op), "setpoint", old, target.get("setpoint"), "HMI"));
    respond(op, "{\"ok\":true}");
  }
}
