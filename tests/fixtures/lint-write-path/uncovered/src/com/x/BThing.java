package com.x;
@NiagaraProperty(name="setpoint", flags=Flags.SUMMARY | Flags.OPERATOR)
@NiagaraProperty(name="fanMode", flags=Flags.SUMMARY | Flags.OPERATOR)
@NiagaraProperty(name="hoaMode", flags=Flags.SUMMARY | Flags.OPERATOR)
@NiagaraProperty(name="internalState", flags=Flags.READONLY)
public class BThing extends BComponent {}
