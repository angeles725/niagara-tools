package demo;
import javax.baja.web.*;
import java.util.logging.Logger;
public class LogInHandler extends BWebServlet {
  static final Logger LOG = Logger.getLogger("demo");
  public void doGet(WebOp op) throws Exception {
    LOG.info("request from " + op.getRequest().getRemoteAddr()); // per-request log line -> WARN
  }
}
