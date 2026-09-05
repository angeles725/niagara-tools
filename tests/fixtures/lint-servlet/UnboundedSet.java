package demo;
import javax.baja.web.*;
public class UnboundedSet extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    BComponent c = (BComponent) resolve(op);
    c.setDouble("setpoint", 999.0); // no MIN/MAX clamp before the write -> WARN
  }
}
