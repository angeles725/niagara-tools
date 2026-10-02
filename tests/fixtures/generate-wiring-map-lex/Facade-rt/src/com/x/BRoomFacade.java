package com.x;
import javax.baja.sys.*;
// BRoomFacade: lexicon-backed display names for the wiring-map display-name column.
@NiagaraType
@NiagaraProperty(name = "temperatureSetpoint", type = "double", defaultValue = "-5.0", flags = Flags.SUMMARY | Flags.OPERATOR)
@NiagaraProperty(name = "roomTemperature", type = "BStatusNumeric", defaultValue = "new BStatusNumeric()", flags = Flags.SUMMARY)
@NiagaraProperty(name = "doorOpen", type = "boolean", defaultValue = "false", flags = Flags.SUMMARY)
public final class BRoomFacade extends BComponent {
}
