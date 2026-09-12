package demo;
import javax.baja.sys.*;
// FlagCombo: airDefrost is only valid when hasDefrost=true — enforced only in a comment (CS3 WARN).
@NiagaraType
public final class FlagCombo extends BComponent {
  @NiagaraProperty(name = "hasDefrost", flags = Flags.OPERATOR)
  private BBoolean hasDefrost;

  // Only valid when hasDefrost=true; no runtime enforcement present
  @NiagaraProperty(name = "airDefrost", flags = Flags.OPERATOR)
  private BBoolean airDefrost;
}
