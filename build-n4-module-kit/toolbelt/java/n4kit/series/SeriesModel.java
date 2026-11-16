package n4kit.series;

/**
 * The adapter a series test writes around one pure (zero-Baja) model: {@link #step} feeds one row's inputs,
 * {@link #output} reads one named output after that step. A boolean output reads as 1 (true) or 0 (false).
 * A name the adapter does not know should throw {@link IllegalArgumentException}: a typo in the series
 * header is then a loud error, never a silent pass.
 */
public interface SeriesModel
{
  /** Applies one row: its time (ms, never decreasing) and its input values. */
  void step(SeriesRow row);

  /** The current value of the output called name. */
  double output(String name);
}
