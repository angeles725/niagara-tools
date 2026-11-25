package demo;

public class Panel extends BWebServlet {
  public void doPost(WebOp op) {
    // post-permission: ok the route is admin-only, enforced by the servlet filter
    post(restart, null, cx);
  }
  public void doPut(WebOp op) {
    post(restart, null, cx); // post-permission: ok
  }
}
