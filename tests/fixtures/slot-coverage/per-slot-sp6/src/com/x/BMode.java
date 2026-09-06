package com.x;
// Frozen enum with @Range tags — tags appear as lexicon keys (convention), not @NiagaraProperty slots
@NiagaraEnum
@Range("fast")
@Range("slow")
public final class BMode extends BFrozenEnum {}
