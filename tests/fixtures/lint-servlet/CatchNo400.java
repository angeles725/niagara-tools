package demo;
import javax.baja.web.*;
public class CatchNo400 extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    String raw = op.getRequest().getParameter("v");
    double v;
    try { v = Double.parseDouble(raw); }
    catch (NumberFormatException e) { v = 0.0; } // silent default, no 400 -> WARN
    ((BComponent) resolve(op)).setDouble("setpoint", v);
  }
}
