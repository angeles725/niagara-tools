package demo;
import javax.baja.workbench.*;
/** WBT1d fixture: doInvoke -> helperA -> helperB -> nav traversal (depth-3 chain, WARN). */
public class DeepTraversal extends BWbComponent {
  public void doInvoke(Context cx) {
    // real shape BBatchLinkEditor:303 -- three levels of indirection before nav traversal
    helperA();
  }
  private void helperA() { helperB(); }
  private void helperB() {
    for (Object child : getNavChildren()) { process(child); }
  }
  void process(Object o) {}
}
