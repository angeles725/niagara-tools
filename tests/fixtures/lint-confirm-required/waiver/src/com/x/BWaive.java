package com.x;

@NiagaraType
// lint-confirm-required: waived
@NiagaraAction(name = "resetBare")
@NiagaraAction(name = "pingBare") // lint-confirm-required: waived
@NiagaraAction(name = "wipeFlagged", flags = Flags.CONFIRM_REQUIRED) // lint-confirm-required: waived
public class BWaive extends BComponent {}
