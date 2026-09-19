package com.x;
import javax.baja.sys.*;
public final class BQuery extends BComponent {
    void run(String v) { String q = "bql:select * from control:NumericPoint where value = " + v; }
}
