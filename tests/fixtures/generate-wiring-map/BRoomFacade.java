package demo;
import javax.baja.sys.*;

// BRoomFacade: one OPERATOR config slot (facade->control) + one SUMMARY display slot (control->facade).
// Used by generate-wiring-map.bats to verify Table 1 and Table 2 extraction.
@NiagaraType
public final class BRoomFacade extends BComponent {

  // OPERATOR config slot — facade → control direction
  @NiagaraProperty(
    name = "temperatureSetpoint",
    flags = "Flags.SUMMARY|Flags.OPERATOR",
    defaultValue = "BDouble.make(-5.0)"
  )
  private BDouble temperatureSetpoint;

  // SUMMARY display slot — control → facade direction (no OPERATOR)
  @NiagaraProperty(
    name = "roomTemperature",
    flags = "Flags.SUMMARY",
    defaultValue = "BStatusNumeric.make(0.0)"
  )
  private BStatusNumeric roomTemperature;
}
