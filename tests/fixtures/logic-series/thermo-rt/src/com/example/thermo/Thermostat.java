package com.example.thermo;

/**
 * Toy zero-Baja decision core for the logic-series worked example: cooling with a +/-1 degree band around
 * the setpoint, a minimum off time, and a proportional demand (0..100 %, 25 % per degree above setpoint).
 */
final class Thermostat
{
  static final long MIN_OFF_MS = 3000L;

  boolean cooling;
  double demand;
  private long offSince = Long.MIN_VALUE;

  void step(long now, double temp, double setpoint, boolean enable)
  {
    demand = enable ? Math.max(0d, Math.min(100d, (temp - setpoint) * 25d)) : 0d;
    boolean offLongEnough = offSince == Long.MIN_VALUE || now - offSince >= MIN_OFF_MS;
    if (!enable || temp < setpoint - 1d) {
      if (cooling) offSince = now;
      cooling = false;
    }
    else if (!cooling && temp > setpoint + 1d && offLongEnough) {
      cooling = true;
    }
  }
}
