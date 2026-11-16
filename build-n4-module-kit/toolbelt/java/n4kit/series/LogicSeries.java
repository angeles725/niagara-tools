package n4kit.series;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.Arrays;
import java.util.List;

/**
 * A scripted input series with expected outputs, stepped through a pure model offline (before the lab).
 * Format: {@link SeriesParser}. Use from a JUnit test:
 * <pre>
 * LogicSeries.fromFile("srcTest/series/thermostat.csv").run(new SeriesModel() { … });
 * </pre>
 * {@link #run} stops at the first mismatch with an {@link AssertionError} naming the source line, step,
 * time, output, expected value, tolerance and actual value; a malformed series is an
 * {@link IllegalArgumentException} before any step runs. run-pure-test.sh runs the JVM from the module
 * part directory, so a path relative to it (srcTest/series/…) resolves wherever the runner was called.
 */
public final class LogicSeries
{
  private final String source;
  private final List<SeriesRow> rows;
  private final List<String> outNames;
  private final List<Double> outTols;
  private final List<Double[]> expected;

  LogicSeries(String source, List<SeriesRow> rows, List<String> outNames, List<Double> outTols,
      List<Double[]> expected)
  {
    this.source = source;
    this.rows = rows;
    this.outNames = outNames;
    this.outTols = outTols;
    this.expected = expected;
  }

  /** Reads a series file (UTF-8); the path is relative to the JVM working directory. */
  public static LogicSeries fromFile(String path)
  {
    try {
      return SeriesParser.parse(path, Files.readAllLines(Paths.get(path), StandardCharsets.UTF_8));
    }
    catch (IOException e) {
      throw new IllegalArgumentException("cannot read series " + path + ": " + e, e);
    }
  }

  /** A series written inline in the test, one string per line; name labels it in messages. */
  public static LogicSeries fromLines(String name, String... lines)
  {
    return SeriesParser.parse(name, Arrays.asList(lines));
  }

  /** The number of data rows (steps). */
  public int steps() { return rows.size(); }

  /**
   * Steps the model through every row and checks each expected output after its step.
   * @return the number of output values checked (cells marked - are not counted)
   * @throws AssertionError at the first output outside its tolerance
   */
  public int run(SeriesModel model)
  {
    int checked = 0;
    for (int s = 0; s < rows.size(); s++) {
      SeriesRow row = rows.get(s);
      model.step(row);
      Double[] exp = expected.get(s);
      for (int o = 0; o < exp.length; o++) {
        if (exp[o] == null) continue;
        double want = exp[o].doubleValue(), tol = outTols.get(o).doubleValue();
        double got = model.output(outNames.get(o));
        if (!matches(want, got, tol))
          throw new AssertionError(source + ":" + row.line() + ": step " + (s + 1) + " (t=" + row.timeMs()
              + " ms) out:" + outNames.get(o) + " expected " + want + (tol > 0d ? " ±" + tol : "")
              + ", actual " + got);
        checked++;
      }
    }
    return checked;
  }

  /** NaN matches only NaN; an infinity only itself; anything else within tol. */
  static boolean matches(double want, double got, double tol)
  {
    if (Double.isNaN(want) || Double.isNaN(got)) return Double.isNaN(want) && Double.isNaN(got);
    return want == got || Math.abs(want - got) <= tol;
  }
}
