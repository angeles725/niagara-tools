package demo;
import javax.baja.sys.*;
// ParityFacade: 2 Duration config slots, 2 Active status slots -> N==M, no WARN.
@NiagaraType
public final class ParityFacade extends BComponent {
  @NiagaraProperty(name = "defrost1Duration")
  private BRelTime defrost1Duration;
  @NiagaraProperty(name = "defrost2Duration")
  private BRelTime defrost2Duration;
  @NiagaraProperty(name = "defrost1Active")
  private BBoolean defrost1Active;
  @NiagaraProperty(name = "defrost2Active")
  private BBoolean defrost2Active;
}
