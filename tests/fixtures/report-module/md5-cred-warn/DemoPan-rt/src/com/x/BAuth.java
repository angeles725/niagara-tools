package com.x;
import java.security.MessageDigest;
public final class BAuth extends BComponent {
    BPassword pwd;
    byte[] h(String s) throws Exception { return MessageDigest.getInstance("MD5").digest(s.getBytes()); }
}
