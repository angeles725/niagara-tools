package demo;
import javax.baja.workbench.*;
/** WBT1d twin: doInvoke -> helperA[invokeLater] -> helperB -> getNavChildren (guarded, NOT WARN). */
public class DeepGuardedTraversal extends BWbComponent {
  public void doInvoke(Context cx) {
    helperA();
  }
  private void helperA() {
    // off-loads helperB to a non-Swing thread -- getNavChildren is not on the EDT
    invokeLater(() -> helperB());
  }
  private void helperB() {
    for (Object child : getNavChildren()) { process(child); }
  }
  void process(Object o) {}
}
