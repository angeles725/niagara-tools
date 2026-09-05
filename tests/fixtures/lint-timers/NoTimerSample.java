package fixtures;
// Lint fixture: a class with no timers — should produce no FAIL from lint-timers.sh.
public class NoTimerSample {
  private int count;
  public void increment() { count++; }
  public int getCount() { return count; }
}
