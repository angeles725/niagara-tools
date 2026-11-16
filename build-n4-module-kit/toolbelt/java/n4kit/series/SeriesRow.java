package n4kit.series;

import java.util.Collections;
import java.util.Map;

/**
 * One step of a series: the source line, the time in ms and the input values by name. A boolean cell
 * ({@code true}/{@code false}) is stored as 1/0, so {@link #num} and {@link #bool} both read any input.
 */
public final class SeriesRow
{
  private final int line;
  private final long timeMs;
  private final Map<String, Double> inputs;

  SeriesRow(int line, long timeMs, Map<String, Double> inputs)
  {
    this.line = line;
    this.timeMs = timeMs;
    this.inputs = Collections.unmodifiableMap(inputs);
  }

  /** The 1-based line of the series source this row came from. */
  public int line() { return line; }

  /** The row's time in ms. */
  public long timeMs() { return timeMs; }

  /** The input called name as a number; an unknown name throws (a model asking for a column the series lacks). */
  public double num(String name)
  {
    Double v = inputs.get(name);
    if (v == null)
      throw new IllegalArgumentException("series has no input column in:" + name + " (line " + line + ")");
    return v.doubleValue();
  }

  /** The input called name as a boolean: any value other than 0 is true. */
  public boolean bool(String name) { return num(name) != 0d; }
}
