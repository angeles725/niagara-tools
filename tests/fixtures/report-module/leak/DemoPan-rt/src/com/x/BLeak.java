package com.x;
import javax.baja.sys.*;
public final class BLeak extends BComponent { private Clock.Ticket t; public void arm(){ t = Clock.schedule(this, BRelTime.makeSeconds(5), null, null); } }
