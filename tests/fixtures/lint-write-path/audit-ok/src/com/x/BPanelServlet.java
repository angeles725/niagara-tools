package com.x;
public class BPanelServlet extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    BComponent target = resolve(op);
    BValue old = target.get("setpoint");
    target.set("setpoint", BDouble.make(parse(op)), cx(op));
    svc.appendAudit(buildAuditEntry(user(op), "setpoint", old, target.get("setpoint"), "HMI"));
    respond(op, "{\"ok\":true,\"value\":" + target.get("setpoint") + "}");
  }
}
