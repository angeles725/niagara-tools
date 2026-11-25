package demo;

public class Nest extends BWebServlet {
  public void doPost(WebOp op) {
    Runnable r = new Runnable() {
      public void run() {
        post(restart, null, cx);
      }
    };
  }
}
