package demo;
import javax.baja.web.*;
import java.io.PrintWriter;
public class ApiRespWithHeader extends BWebServlet {
  public void doGet(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    op.getResponse().setHeader("X-Content-Type-Options", "nosniff");
    op.getResponse().setContentType("application/json");
    PrintWriter out = op.getResponse().getWriter();
    out.print("{\"ok\":true}");
  }
}
