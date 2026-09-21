package com.x;
import javax.baja.sys.*;
public final class BSub extends BComponent {
    public void started() { link.subscribe(this); }
}
