package com.x;

@NiagaraType
@NiagaraAction(
  name = "resetGood",
  parameterType = "BInteger",
  defaultValue = "BInteger.make(0)",
  flags = Flags.OPERATOR
    | Flags.CONFIRM_REQUIRED)
@NiagaraAction(
  name = "resetBad",
  parameterType = "BInteger",
  defaultValue = "BInteger.make(0)",
  flags = Flags.OPERATOR
    | Flags.HIDDEN)
public class BWide extends BComponent {}
