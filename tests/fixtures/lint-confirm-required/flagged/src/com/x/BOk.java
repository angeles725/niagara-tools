package com.x;

@NiagaraType
@NiagaraAction(name = "faultReset", flags = Flags.CONFIRM_REQUIRED)
@NiagaraAction(name = "clearHistory", flags = Flags.OPERATOR | Flags.CONFIRM_REQUIRED)
@NiagaraAction(name = "wipeDevice", flags = Flags.CONFIRM_REQUIRED | Flags.HIDDEN | Flags.NO_AUDIT)
@NiagaraAction(name = "tick", flags = Flags.HIDDEN)
public class BOk extends BComponent {}
