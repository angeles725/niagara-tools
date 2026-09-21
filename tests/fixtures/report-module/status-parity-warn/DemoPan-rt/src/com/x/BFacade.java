package com.x;
import javax.baja.sys.*;
public final class BFacade extends BComponent {
    @NiagaraProperty(
        name="setInterval")
    private BRelTime setInterval;
    @NiagaraProperty(
        name="defrostInterval")
    private BRelTime defrostInterval;
}
