package demo;

public class Leak extends BWebServlet {
  public void doPost(WebOp op) {
    post(a, b, cx); // post-permission: ok first only
    post(c, d, cx);
  }
}
