package com.example.thermo;

import static org.junit.Assert.assertTrue;

import n4kit.series.LogicSeries;
import n4kit.series.SeriesModel;
import n4kit.series.SeriesRow;
import org.junit.Test;

/**
 * Worked example of a logic-series test: the adapter maps series columns to the pure model's step
 * arguments and fields. Run it with
 * {@code build-n4-module-kit/toolbelt/run-pure-test.sh <this-rt-dir> com.example.thermo.ThermostatSeriesTest}.
 */
public class ThermostatSeriesTest
{
  /** Adapter: in:temp, in:setpoint, in:enable → step(); out:cooling, out:demand ← fields. */
  static SeriesModel adapter(final Thermostat t)
  {
    return new SeriesModel() {
      @Override public void step(SeriesRow r)
      {
        t.step(r.timeMs(), r.num("temp"), r.num("setpoint"), r.bool("enable"));
      }

      @Override public double output(String name)
      {
        if ("cooling".equals(name)) return t.cooling ? 1d : 0d;
        if ("demand".equals(name)) return t.demand;
        throw new IllegalArgumentException("Thermostat has no output " + name);
      }
    };
  }

  @Test public void thermostat_followsTheScriptedSeries()
  {
    int checked = LogicSeries.fromFile("srcTest/series/thermostat.csv").run(adapter(new Thermostat()));
    System.out.println("series checked " + checked + " values");
    assertTrue("a series that checks nothing proves nothing", checked > 0);
  }
}
