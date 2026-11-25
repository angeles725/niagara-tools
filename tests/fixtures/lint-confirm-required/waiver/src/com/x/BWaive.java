package com.x;

@NiagaraType
@NiagaraAction(name = "resetSame") // lint-confirm-required: waived internal timer callback, never user driven
// lint-confirm-required: waived the commissioning wizard asks before it calls this
@NiagaraAction(name = "resetAbove")
// lint-confirm-required: waived
@NiagaraAction(name = "resetBare")
public class BWaive extends BComponent {}
