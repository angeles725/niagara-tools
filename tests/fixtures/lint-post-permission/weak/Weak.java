package demo;

public class Weak extends BWebServlet {
  public void doPost(WebOp op) {
    if (!perms.has(BPermissions.operatorRead)) return;
    post(restart, null, cx);
  }
  public void doPut(WebOp op) {
    check(x);
    post(restart, null, cx);
  }
}
