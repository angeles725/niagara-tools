package demo;
import javax.baja.sys.*;
// WithUnconditional: resistanceOut is written OFF in execute() (guarded) AND in started() (unconditional).
// Both writes present -> no FAIL (the unconditional reset satisfies the gate).
public final class WithUnconditional extends BComponent {
  private boolean inDefrost = false;

  @Override
  public void started() {
    getResistanceOut().setValue(false);  // UNCONDITIONAL reset in started() -- satisfies LRP gate
  }

  public void execute() {
    if (inDefrost) return;
    getResistanceOut().setValue(false);  // also written in transient method
  }

  public BBoolean getResistanceOut() { return (BBoolean) get("resistanceOut"); }
}
