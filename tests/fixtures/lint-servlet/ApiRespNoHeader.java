package demo;
import javax.baja.web.*;
import java.io.PrintWriter;
public class ApiRespNoHeader extends BWebServlet {
  public void doGet(WebOp op) throws Exception {
    if (op.getUser() == null) { op.getResponse().sendError(401); return; }
    op.getResponse().setContentType("application/json");
    PrintWriter out = op.getResponse().getWriter(); // no X-Content-Type-Options anywhere -> WARN
    out.print("{\"ok\":true}");
  }
}
