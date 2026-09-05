package demo;
import javax.baja.workbench.*;
public class Traversal extends BWbComponent {
  public void doInvoke(Context cx) {
    // real shape BBatchLinkEditor:684-720 — DFS over the station tree ON the Swing thread
    for (Object child : getNavChildren()) { walk(child); }
  }
  void walk(Object o) {}
}
