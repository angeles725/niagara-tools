package demo;
import javax.baja.workbench.*;
public class GuardedTraversal extends BWbComponent {
  public void doInvoke(Context cx) {
    invokeLater(() -> { for (Object child : getNavChildren()) walk(child); }); // off the UI thread
  }
  void walk(Object o) {}
}
