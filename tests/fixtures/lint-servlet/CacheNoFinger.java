package demo;
import javax.baja.web.*;
public class CacheNoFinger extends BWebServlet {
  public void doGet(WebOp op) throws Exception {
    op.getResponse().setHeader("Cache-Control", "max-age=3600"); // rc/ asset, unfingerprinted -> WARN
    serveRc(op);
  }
  void serveRc(WebOp op) {}
}
