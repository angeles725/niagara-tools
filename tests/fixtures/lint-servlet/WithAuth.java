package demo;
import javax.baja.web.*;
public class WithAuth extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    ((BComponent) resolve(op)).set("setpoint", op.getRequest().getParameter("v"));
  }
}
