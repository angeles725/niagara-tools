package demo;

public class Panel extends BWebServlet {
  public void doPost(WebOp op) {
    BUser user = cx.getUser();
    user.checkInvoke(this, restart);
    post(restart, null, cx);
  }
}
