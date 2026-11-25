package demo;

public class Panel extends BWebServlet {
  public void doPost(WebOp op) {
    post(restart, null, cx);
  }
  // The check comes after the post, so it guards nothing.
  public void doPut(WebOp op) {
    post(restart, null, cx);
    cx.getUser().checkInvoke(this, restart);
  }
  // A guard named only in a comment or a string is not a guard.
  public void doGet(WebOp op) {
    // cx.getUser().checkInvoke(this, restart);
    String note = "checkInvoke(this, restart)";
    post(restart, null, cx);
  }
}
