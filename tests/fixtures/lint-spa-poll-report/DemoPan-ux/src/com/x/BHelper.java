package com.x;
import javax.baja.sys.*;
public final class BHelper extends BObject {
    public static final BHelper INSTANCE = new BHelper();
    public Type getType() { return TYPE; }
    public static final Type TYPE = Sys.loadType(BHelper.class);
}
