package n4kit.series;

import java.util.ArrayList;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Reads the series text format (comma-separated; blank lines and lines starting with # are skipped):
 * <pre>
 * time_ms, in:temp, in:enable, out:cooling, out:demand~0.5
 * 0,       20,      true,      false,       0
 * 1000,    26,      true,      true,        -
 * </pre>
 * The header starts with {@code time_ms}; every other column is {@code in:<name>} or {@code out:<name>},
 * an output optionally with a tolerance {@code ~<tol>} (default 0: exact). A cell is a number, {@code true}
 * or {@code false}; an output cell may be {@code -} (not checked at that step) or {@code NaN}. Times never
 * decrease. Anything else is refused with the source name and line, never skipped.
 */
final class SeriesParser
{
  private static final Pattern NUMBER = Pattern.compile("[-+]?(\\d+(\\.\\d*)?|\\.\\d+)([eE][-+]?\\d+)?");
  private static final Pattern NAME = Pattern.compile("[A-Za-z_][A-Za-z0-9_.]*");
  private static final Pattern TIME = Pattern.compile("[-+]?\\d+");
  static final String TIME_COLUMN = "time_ms";
  static final String SKIP = "-";

  private final String source;
  private final List<String> inNames = new ArrayList<String>();
  private final List<String> outNames = new ArrayList<String>();
  private final List<Double> outTols = new ArrayList<Double>();
  private final List<Boolean> isOut = new ArrayList<Boolean>(); // per header column after time_ms
  private final List<SeriesRow> rows = new ArrayList<SeriesRow>();
  private final List<Double[]> expected = new ArrayList<Double[]>();

  private SeriesParser(String source) { this.source = source; }

  static LogicSeries parse(String source, List<String> lines)
  {
    SeriesParser p = new SeriesParser(source);
    boolean header = false;
    for (int i = 0; i < lines.size(); i++) {
      String text = lines.get(i).trim();
      if (text.isEmpty() || text.startsWith("#")) continue;
      String[] cells = text.split(",", -1);
      for (int c = 0; c < cells.length; c++) cells[c] = cells[c].trim();
      if (!header) { p.header(i + 1, cells); header = true; }
      else p.row(i + 1, cells);
    }
    if (!header) throw p.bad(lines.size(), "no header line (expected " + TIME_COLUMN + ", in:…, out:…)");
    if (p.rows.isEmpty()) throw p.bad(lines.size(), "no data rows after the header");
    return new LogicSeries(source, p.rows, p.outNames, p.outTols, p.expected);
  }

  private void header(int line, String[] cells)
  {
    if (!TIME_COLUMN.equals(cells[0]))
      throw bad(line, "first column must be " + TIME_COLUMN + ", got '" + cells[0] + "'");
    Set<String> seen = new HashSet<String>();
    for (int c = 1; c < cells.length; c++) {
      String col = cells[c];
      boolean out = col.startsWith("out:");
      if (!out && !col.startsWith("in:"))
        throw bad(line, "column '" + col + "' must start with in: or out:");
      String name = col.substring(out ? 4 : 3);
      double tol = 0d;
      int tilde = name.indexOf('~');
      if (tilde >= 0) {
        if (!out) throw bad(line, "column '" + col + "': a tolerance applies to an out: column only");
        String t = name.substring(tilde + 1);
        if (!NUMBER.matcher(t).matches() || Double.parseDouble(t) < 0d)
          throw bad(line, "column '" + col + "': tolerance must be a number >= 0");
        tol = Double.parseDouble(t);
        name = name.substring(0, tilde);
      }
      if (!NAME.matcher(name).matches()) throw bad(line, "column '" + col + "': bad name");
      if (!seen.add(col.substring(0, out ? 4 : 3) + name)) throw bad(line, "duplicate column '" + col + "'");
      isOut.add(out);
      if (out) { outNames.add(name); outTols.add(tol); } else inNames.add(name);
    }
    if (outNames.isEmpty()) throw bad(line, "no out: column — a series must expect at least one output");
  }

  private void row(int line, String[] cells)
  {
    if (cells.length != isOut.size() + 1)
      throw bad(line, cells.length + " cells, header has " + (isOut.size() + 1));
    if (!TIME.matcher(cells[0]).matches()) throw bad(line, "time '" + cells[0] + "' is not an integer (ms)");
    long time = Long.parseLong(cells[0].startsWith("+") ? cells[0].substring(1) : cells[0]);
    if (!rows.isEmpty() && time < rows.get(rows.size() - 1).timeMs())
      throw bad(line, "time " + time + " goes back (previous " + rows.get(rows.size() - 1).timeMs() + ")");
    Map<String, Double> in = new LinkedHashMap<String, Double>();
    Double[] exp = new Double[outNames.size()];
    int ii = 0, oi = 0;
    for (int c = 1; c < cells.length; c++) {
      if (isOut.get(c - 1)) {
        exp[oi] = SKIP.equals(cells[c]) ? null : value(line, cells[c], true);
        oi++;
      } else {
        in.put(inNames.get(ii++), value(line, cells[c], false));
      }
    }
    rows.add(new SeriesRow(line, time, in));
    expected.add(exp);
  }

  private Double value(int line, String cell, boolean out)
  {
    if ("true".equals(cell)) return 1d;
    if ("false".equals(cell)) return 0d;
    if (out && "NaN".equals(cell)) return Double.NaN;
    if (NUMBER.matcher(cell).matches()) return Double.parseDouble(cell);
    throw bad(line, "cell '" + cell + "' is not a number, true or false" + (out ? " (or - / NaN)" : ""));
  }

  private IllegalArgumentException bad(int line, String why)
  {
    return new IllegalArgumentException("bad series " + source + ":" + line + ": " + why);
  }
}
