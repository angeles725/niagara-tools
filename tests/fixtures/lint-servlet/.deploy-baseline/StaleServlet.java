package demo;
import javax.baja.web.*;
public class NoAuth extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    // no getRemoteUser()/op.getUser() gate before a mutating write
    ((BComponent) resolve(op)).set("setpoint", op.getRequest().getParameter("v"));
  }
}
