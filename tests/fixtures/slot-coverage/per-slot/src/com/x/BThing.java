package com.x;
// setpoint: OPERATOR, NO lexicon key      -> MISSING (renders raw camelCase in operator views)
@NiagaraProperty(name = "setpoint", flags = Flags.SUMMARY | Flags.OPERATOR)
// fanMode: OPERATOR, HAS a lexicon key     -> covered
@NiagaraProperty(name = "fanMode", flags = Flags.SUMMARY | Flags.OPERATOR)
// internalState: READONLY (non-operator)   -> NOT operator-facing, must NOT be MISSING (the OPERATOR gate)
@NiagaraProperty(name = "internalState", flags = Flags.READONLY | Flags.TRANSIENT)
public class BThing extends BComponent {}
