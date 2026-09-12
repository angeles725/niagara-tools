package demo;
import javax.baja.sys.*;
// AsymFacade: 3 per-instance Duration config slots, 1 scalar Active status slot.
// N=3 > M=1 -> LSP WARN (or FAIL under --strict).
@NiagaraType
public final class AsymFacade extends BComponent {
  // 3 indexed Duration config slots
  @NiagaraProperty(name = "defrost1Duration")
  private BRelTime defrost1Duration;
  @NiagaraProperty(name = "defrost2Duration")
  private BRelTime defrost2Duration;
  @NiagaraProperty(name = "defrost3Duration")
  private BRelTime defrost3Duration;
  // 1 scalar status slot
  @NiagaraProperty(name = "defrostActive")
  private BBoolean defrostActive;
}
