package com.x;
@NiagaraType
// Commissioning link: written by BLink.
@NiagaraProperty(name = "oldFlag", type = "boolean", defaultValue = "false", flags = Flags.SUMMARY | Flags.READONLY)
@NiagaraProperty(name = "oldMode", type = "int", defaultValue = "0", flags = Flags.OPERATOR | Flags.TRANSIENT)
public final class BOld extends BComponent {}
