package com.angeles.wsplacement;

/**
 * Baja-free wire-sheet grid: maps a stable child index to the {@code p,q} cell (in wixels) of its
 * {@code wsAnnotation}, row-major, clamped inside the sheet. Copy it into the module's rt package
 * (rename the package) and test it with the module's pure tests; it needs no Niagara class.
 *
 * <p>Why: a component added from code with no {@code wsAnnotation} gets Workbench's in-memory
 * diagonal stagger ({@code +4}, wrap to 2, width 8), so every code-created sibling overlaps
 * (corpus B1172 §1172.3; {@code WsController.java:398-405}). The sheet clamps {@code p+w} and
 * {@code q+h} to {@code BWsOptions} maxWidth / maxHeight, default 600 (B1172 §1172.3 item 2).
 *
 * <p>Index rule: pass the child's position in the STABLE data order (the inventory row, the
 * seed-table row), never the add order, so a re-sync that creates one missing child puts it in
 * its own cell instead of on top of a sibling.
 *
 * <p>Baja recipe (add if absent, READONLY like Workbench; never re-place an existing one, the
 * user may have moved it — B1172 §1172.2, {@code WsController.java:189-200}):
 * <pre>
 *   if (child.getProperty("wsAnnotation") == null)
 *     child.add("wsAnnotation", BWsAnnotation.make(GRID.p(i), GRID.q(i), GRID.width()), Flags.READONLY);
 *   if (parent.getProperty(name) == null) parent.add(name, child);
 * </pre>
 * ({@code javax.baja.util.BWsAnnotation}, {@code javax.baja.sys.Flags}). Annotating before the
 * parent {@code add} stores the child with its position in one add (B1172 §1172.5 item 1).
 */
public final class WsGrid
{
  /** {@code BWsOptions} maxWidth / maxHeight property default, in wixels (B1172 §1172.3). */
  public static final int SHEET_MAX = 600;
  /** The width Workbench itself assigns to a standard component glyph (B1172 §1172.5). */
  public static final int GLYPH_WIDTH = 8;
  /** Smallest glyph height: 2 + 0 visible slot bars + 1 (B1194 §1194.8). */
  private static final int MIN_GLYPH_HEIGHT = 3;
  /** Row spacing of {@link #standard()}: clears a glyph with up to 4 SUMMARY bars by {@link #ROW_GAP}. */
  private static final int STANDARD_ROW_SPACING = 10;
  /** Free wixels between a glyph's bottom edge and the next row (standard(): 10 - (2 + 4 + 1)). */
  private static final int ROW_GAP = 3;
  /** Left and top margin of {@link #standard()}, in wixels. */
  private static final int MARGIN = 3;
  /** Largest SUMMARY count whose row spacing still fits one row under the top margin: 600 - 3 - 3 - 3 = 591. */
  public static final int MAX_SUMMARY_COUNT = SHEET_MAX - MARGIN - MIN_GLYPH_HEIGHT - ROW_GAP;

  private final int cols;
  private final int colSpacing;
  private final int rowSpacing;
  private final int pOffset;
  private final int qOffset;
  private final int width;
  private final int maxWidth;
  private final int maxHeight;

  /**
   * @param cols       cells per row (at least 1)
   * @param colSpacing wixels between the left edges of neighbouring cells (wider than {@code width})
   * @param rowSpacing wixels between the top edges of neighbouring rows: the tallest expected glyph
   *                   (2 + visible SUMMARY slot bars + 1) plus a gap; at least 4
   * @param pOffset    left margin (at least 0)
   * @param qOffset    top margin (at least 0)
   * @param width      glyph width in wixels (at least 2; Workbench raises a smaller one)
   * @param maxWidth   sheet width ({@link #SHEET_MAX} unless the sheet's BWsOptions changed it)
   * @param maxHeight  sheet height ({@link #SHEET_MAX} unless the sheet's BWsOptions changed it)
   */
  public WsGrid(int cols, int colSpacing, int rowSpacing, int pOffset, int qOffset,
                int width, int maxWidth, int maxHeight)
  {
    if (cols < 1) throw new IllegalArgumentException("cols < 1: " + cols);
    if (width < 2) throw new IllegalArgumentException("width < 2: " + width);
    if (colSpacing <= width) throw new IllegalArgumentException("colSpacing <= width: " + colSpacing);
    if (rowSpacing <= MIN_GLYPH_HEIGHT) throw new IllegalArgumentException("rowSpacing <= 3: " + rowSpacing);
    if (pOffset < 0 || qOffset < 0) throw new IllegalArgumentException("negative offset");
    if (pOffset + width > maxWidth || qOffset + rowSpacing > maxHeight)
      throw new IllegalArgumentException("the first cell does not fit a " + maxWidth + "x" + maxHeight + " sheet");
    this.cols = cols;
    this.colSpacing = colSpacing;
    this.rowSpacing = rowSpacing;
    this.pOffset = pOffset;
    this.qOffset = qOffset;
    this.width = width;
    this.maxWidth = maxWidth;
    this.maxHeight = maxHeight;
  }

  /**
   * 8 columns, 14 x 10 wixel cells, 3 wixel margins, width 8, 600 x 600 sheet (chihuahua's grid). Fits
   * children of up to 4 SUMMARY slots; a taller facade glyph takes {@link #forSummaryCount(int)}.
   */
  public static WsGrid standard()
  {
    return forRowSpacing(STANDARD_ROW_SPACING);
  }

  /**
   * Row spacing that clears a glyph showing {@code summaryCount} SUMMARY slot bars:
   * {@code max(10, 2 + summaryCount + 1 + 3)} — the glyph height (B1194 §1194.8) plus the 3-wixel gap
   * {@link #standard()} leaves under a 4-bar glyph. A 22-SUMMARY facade (25 wixels tall) gets 28; the fixed
   * 10 of {@link #standard()} would stack its rows on top of each other (issue #259). A count past
   * {@link #MAX_SUMMARY_COUNT} is rejected: its glyph cannot fit the sheet (and near
   * {@code Integer.MAX_VALUE} the sum would overflow into a silent 10).
   */
  public static int rowSpacingFor(int summaryCount)
  {
    if (summaryCount < 0) throw new IllegalArgumentException("summaryCount < 0: " + summaryCount);
    if (summaryCount > MAX_SUMMARY_COUNT)
      throw new IllegalArgumentException("summaryCount > " + MAX_SUMMARY_COUNT + " does not fit the sheet: " + summaryCount);
    return Math.max(STANDARD_ROW_SPACING, MIN_GLYPH_HEIGHT + summaryCount + ROW_GAP);
  }

  /**
   * {@link #standard()} with the row spacing derived from the tallest child's SUMMARY slot count
   * ({@link #rowSpacingFor(int)}); a count of 4 or fewer gives exactly {@link #standard()}.
   */
  public static WsGrid forSummaryCount(int summaryCount)
  {
    return forRowSpacing(rowSpacingFor(summaryCount));
  }

  private static WsGrid forRowSpacing(int rowSpacing)
  {
    return new WsGrid(8, 14, rowSpacing, MARGIN, MARGIN, GLYPH_WIDTH, SHEET_MAX, SHEET_MAX);
  }

  /** Column coordinate of cell {@code index}, clamped so {@code p + width <= maxWidth}. */
  public int p(int index)
  {
    return Math.min(rawP(index), maxWidth - width);
  }

  /** Row coordinate of cell {@code index}, clamped so {@code q + rowSpacing <= maxHeight}. */
  public int q(int index)
  {
    return Math.min(rawQ(index), maxHeight - rowSpacing);
  }

  /** Glyph width to store in the annotation. */
  public int width()
  {
    return width;
  }

  /** True when cell {@code index} lies inside the sheet unclamped (a clamped cell may overlap). */
  public boolean fits(int index)
  {
    return rawP(index) + width <= maxWidth && rawQ(index) + rowSpacing <= maxHeight;
  }

  /** The {@code "p,q,w"} string form of the annotation (BOG {@code v=} / BOX value). */
  public String annotation(int index)
  {
    return p(index) + "," + q(index) + "," + width;
  }

  private int rawP(int index)
  {
    return pOffset + col(index) * colSpacing;
  }

  private int rawQ(int index)
  {
    checkIndex(index);
    long q = qOffset + (long) (index / cols) * rowSpacing;
    return q > Integer.MAX_VALUE ? Integer.MAX_VALUE : (int) q;
  }

  private int col(int index)
  {
    checkIndex(index);
    return index % cols;
  }

  private static void checkIndex(int index)
  {
    if (index < 0) throw new IllegalArgumentException("index < 0: " + index);
  }
}
