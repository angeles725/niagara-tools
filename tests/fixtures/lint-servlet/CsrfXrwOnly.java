package demo;
import javax.baja.web.*;
public class CsrfXrwOnly extends BWebServlet {
  public void doPost(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    // CSRF guard uses X-Requested-With only — no CsrfUtil / x-niagara-csrfToken
    String xhr = op.getRequest().getHeader("X-Requested-With");
    if (!"XMLHttpRequest".equals(xhr)) { op.getResponse().sendError(400); return; }
    ((BComponent) resolve(op)).setDouble("setpoint", 22.0);
  }
}
