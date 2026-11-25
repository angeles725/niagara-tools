package demo;

public class Poller extends BComponent {
  public void started() { post(tick, null, null); }
  public void doTick() { post(tick, null, null); }
  private void kick() { post(tick, null, null); }
}
