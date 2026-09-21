package com.x;
import javax.baja.sys.*;
public final class BControl extends BComponent {
    @NiagaraProperty(name="setpoint", flags=Flags.OPERATOR)
    private BDouble setpoint = BDouble.make(0.0);
}
