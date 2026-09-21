package com.x;
import javax.baja.sys.*;
public final class BConfig extends BComponent {
    @NiagaraProperty(name="cycleInterval", flags=Flags.SUMMARY,
        defaultValue="make(BRelTime.class, BRelTime.makeSeconds(300))")
    private BRelTime cycleInterval;
    @NiagaraProperty(name="defrostDuration", flags=Flags.SUMMARY,
        defaultValue="make(BRelTime.class, BRelTime.makeSeconds(600))")
    private BRelTime defrostDuration;
}
