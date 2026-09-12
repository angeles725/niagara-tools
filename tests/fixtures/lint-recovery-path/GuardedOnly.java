package demo;
import javax.baja.sys.*;
// GuardedOnly: resistanceOut is written OFF only inside execute() (not a start/enable method).
// No unconditional reset in started() -> LRP FAIL.
public final class GuardedOnly extends BComponent {
  private boolean inDefrost = false;

  @Override
  public void started() {
    // started() does NOT reset resistanceOut — the reset is missing here
  }

  public void execute() {
    if (inDefrost) return;   // mode gate
    getResistanceOut().setValue(false);  // GUARDED: inside mode-gated execute()
  }

  public BBoolean getResistanceOut() { return (BBoolean) get("resistanceOut"); }
}
