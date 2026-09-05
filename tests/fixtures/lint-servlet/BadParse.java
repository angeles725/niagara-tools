package demo;
import javax.baja.web.*;
public class BadParse extends BWebServlet {
  public void doGet(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    double v = Double.parseDouble(op.getRequest().getParameter("v")); // no try/catch -> 400
    use(v);
  }
  void use(double v) {}
}
