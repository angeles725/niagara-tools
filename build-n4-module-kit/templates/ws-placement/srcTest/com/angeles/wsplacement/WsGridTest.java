package com.angeles.wsplacement;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

import org.junit.Test;

/**
 * Pure test for {@link WsGrid} (Baja-free). Run from the kit root:
 *   toolbelt/run-pure-test.sh templates/ws-placement com.angeles.wsplacement.WsGridTest
 * Evidence for the bounds and the default width: corpus B1172 §1172.1-§1172.3.
 */
public class WsGridTest
{
  @Test
  public void standardGridPlacesRowMajorFromTheOffset()
  {
    WsGrid g = WsGrid.standard();
    assertEquals(3, g.p(0));
    assertEquals(3, g.q(0));
    assertEquals(101, g.p(7));   // last column of row 0: 3 + 7 * 14
    assertEquals(3, g.q(7));
    assertEquals(3, g.p(8));     // wraps to row 1
    assertEquals(13, g.q(8));
    assertEquals(17, g.p(17));
    assertEquals(23, g.q(17));
    assertEquals(8, g.width());
  }

  @Test
  public void annotationIsTheWorkbenchStringForm()
  {
    assertEquals("17,13,8", WsGrid.standard().annotation(9));
  }

  @Test
  public void neighboursNeverShareACell()
  {
    WsGrid g = WsGrid.standard();
    for (int a = 0; a < 200; a++)
      for (int b = a + 1; b < 200; b++)
        assertFalse("cells " + a + " and " + b + " collide",
          g.p(a) == g.p(b) && g.q(a) == g.q(b));
  }

  @Test
  public void rowsPastTheSheetAreClampedInsideItAndReportedAsNotFitting()
  {
    WsGrid g = new WsGrid(1, 14, 10, 3, 3, 8, 600, 600);
    assertTrue(g.fits(58));      // q = 583, 583 + 10 <= 600
    assertFalse(g.fits(59));     // q = 593, 593 + 10 > 600
    assertEquals(590, g.q(100)); // clamped to maxHeight - rowSpacing
    assertEquals(3, g.p(100));
  }

  @Test
  public void columnsPastTheSheetAreClampedInsideIt()
  {
    WsGrid g = new WsGrid(100, 14, 10, 3, 3, 8, 600, 600);
    assertEquals(592, g.p(50));  // 3 + 50 * 14 = 703 -> maxWidth - width
    assertFalse(g.fits(50));
    assertTrue(g.fits(42));      // 3 + 42 * 14 = 591, 591 + 8 <= 600
  }

  @Test
  public void rowSpacingClearsATallFacadeGlyph()
  {
    // DashboardFord BVavPanel: 22 SUMMARY slots -> glyph 2 + 22 + 1 = 25 wixels; 10 overlaps, 28 clears it
    // with the same 3-wixel gap standard() leaves under a 4-SUMMARY glyph (issue #259).
    assertEquals(28, WsGrid.rowSpacingFor(22));
    assertTrue(WsGrid.rowSpacingFor(22) > 2 + 22 + 1);
    WsGrid g = WsGrid.forSummaryCount(22);
    assertEquals(3, g.q(7));     // still 8 columns per row
    assertEquals(31, g.q(8));    // next row starts 28 below: 3 + 28
    assertEquals(3, g.p(8));
    assertEquals(8, g.width());
  }

  @Test
  public void smallGlyphsKeepTheStandardSpacing()
  {
    for (int n = 0; n <= 4; n++)
    {
      assertEquals("summary " + n, 10, WsGrid.rowSpacingFor(n));
      assertEquals("summary " + n, WsGrid.standard().annotation(9), WsGrid.forSummaryCount(n).annotation(9));
    }
    assertEquals(11, WsGrid.rowSpacingFor(5)); // first count past standard(): 2 + 5 + 1 + 3
    try { WsGrid.rowSpacingFor(-1); fail("negative SUMMARY count accepted"); }
    catch (IllegalArgumentException expected) { }
  }

  @Test
  public void summaryCountsWhoseSpacingCannotFitTheSheetAreRejected()
  {
    // 600 - 3 top margin - (2 + n + 1 + 3) >= 0  ->  n <= 591; review R3-001 of #268
    assertEquals(597, WsGrid.rowSpacingFor(591));
    assertEquals(3, WsGrid.forSummaryCount(591).q(8));   // every row clamps onto the only one that fits
    for (int n : new int[] { 592, 1000, Integer.MAX_VALUE - 5, Integer.MAX_VALUE })
    {
      try { WsGrid.rowSpacingFor(n); fail("SUMMARY count " + n + " accepted"); }
      catch (IllegalArgumentException expected) { }
    }
  }

  @Test
  public void invalidShapesAreRejected()
  {
    rejects(0, 14, 10, 3, 3, 8, 600, 600);   // no column
    rejects(8, 8, 10, 3, 3, 8, 600, 600);    // column spacing not wider than the glyph: neighbours touch
    rejects(8, 14, 3, 3, 3, 8, 600, 600);    // row spacing below the smallest glyph (3) plus a gap
    rejects(8, 14, 10, -1, 3, 8, 600, 600);  // negative offset
    rejects(8, 14, 10, 3, 3, 1, 600, 600);   // Workbench raises a width below 2
    rejects(8, 14, 10, 3, 3, 8, 10, 600);    // first cell does not fit the sheet
    try { WsGrid.standard().p(-1); fail("negative index accepted"); }
    catch (IllegalArgumentException expected) { }
  }

  private static void rejects(int cols, int cs, int rs, int po, int qo, int w, int mw, int mh)
  {
    try
    {
      new WsGrid(cols, cs, rs, po, qo, w, mw, mh);
      fail("accepted " + cols + "," + cs + "," + rs + "," + po + "," + qo + "," + w + "," + mw + "," + mh);
    }
    catch (IllegalArgumentException expected) { }
  }
}
