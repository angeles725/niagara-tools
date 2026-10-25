/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package test.com.example.ModuleA;

import javax.baja.nre.annotations.NiagaraType;
import javax.baja.sys.BComponent;
import javax.baja.sys.BDouble;
import javax.baja.sys.BLink;
import javax.baja.sys.BValue;
import javax.baja.sys.Flags;
import javax.baja.sys.LinkCheck;
import javax.baja.sys.Property;
import javax.baja.sys.Sys;
import javax.baja.sys.Type;

import org.testng.Assert;
import org.testng.annotations.Test;

import com.example.ModuleA.BModuleA;

/**
 * BModuleALinkTest — the component's link-in slots, checked in a running test station.
 *
 * <p>{@link #LINK_IN_SLOTS} names every slot an integrator links INTO. For each one:
 * Niagara's LinkCheck accepts a link from a plain source ({@code checkLink(...).isValid()}), the slot is
 * not READONLY (LinkCheck refuses a READONLY target, so such a slot shows its default forever), and for a
 * double slot a real BLink carries a source change to it and the value then holds. Add a slot here when you
 * add a link-in slot to the component.</p>
 */
@NiagaraType
@Test(groups = { "ModuleA" })
public class BModuleALinkTest
  extends BModuleAStationTestBase
{
  /*
   * Pre-slotomatic state: the AUTO region is absent. The build runs :slotomatic before :jar.
   */

  /** Every slot of {@link BModuleA} that an external link targets. */
  static final String[] LINK_IN_SLOTS = { "setpoint", "interval" };

  @Test
  public void linkInSlotsAreNotReadonly()
  {
    BModuleA target = new BModuleA();
    for (String name : LINK_IN_SLOTS)
    {
      Property slot = linkIn(target, name);
      Assert.assertFalse(Flags.isReadonly(target, slot),
        "link-in slot '" + name + "' is READONLY: LinkCheck refuses a READONLY target (types/dashboard.md)");
    }
  }

  @Test
  public void linkCheckAcceptsEveryLinkInSlot()
  {
    BModuleA target = mount("target", new BModuleA());
    for (String name : LINK_IN_SLOTS)
    {
      Property slot = linkIn(target, name);
      BComponent source = mount("source", sourceFor(target, slot));
      LinkCheck check = target.checkLink(source, source.getSlot(SOURCE_SLOT), slot, null);
      Assert.assertTrue(check.isValid(), "LinkCheck refused '" + name + "': " + check.getInvalidReason());
    }
  }

  @Test
  public void linkCarriesASourceChangeToEveryDoubleLinkInSlot()
    throws InterruptedException
  {
    BModuleA target = mount("target", new BModuleA());
    for (String name : LINK_IN_SLOTS)
    {
      Property slot = linkIn(target, name);
      if (!(target.get(slot) instanceof BDouble)) continue;
      BComponent source = mount("source", sourceFor(target, slot));
      BLink link = target.makeLink(source, source.getSlot(SOURCE_SLOT), slot, null);
      target.add(null, link);

      double sent = ((BDouble)target.get(slot)).getDouble() + 42.5d;
      source.set(SOURCE_SLOT, BDouble.make(sent));
      Assert.assertTrue(await(() -> valueOf(target, slot) == sent, WAIT_MS),
        "the link never carried " + sent + " to '" + name + "', has " + valueOf(target, slot));
      Assert.assertTrue(holds(() -> valueOf(target, slot) == sent, HOLD_MS),
        "'" + name + "' did not keep the linked value " + sent + ", has " + valueOf(target, slot));
    }
  }

  ////////////////////////////////////////////////////////////////
  // Support
  ////////////////////////////////////////////////////////////////

  /** The named property, failing the test when a rename left LINK_IN_SLOTS stale. */
  private static Property linkIn(BComponent target, String name)
  {
    Property slot = target.getProperty(name);
    Assert.assertNotNull(slot, "LINK_IN_SLOTS names '" + name + "', which " + target.getType() + " does not have");
    return slot;
  }

  /** A plain component whose one property has the target slot's value type. */
  private static BComponent sourceFor(BComponent target, Property slot)
  {
    BComponent source = new BComponent();
    source.add(SOURCE_SLOT, target.get(slot).newCopy());
    return source;
  }

  private static double valueOf(BComponent target, Property slot)
  {
    BValue value = target.get(slot);
    return value instanceof BDouble ? ((BDouble)value).getDouble() : Double.NaN;
  }

  ////////////////////////////////////////////////////////////////
  // Fields
  ////////////////////////////////////////////////////////////////

  private static final String SOURCE_SLOT = "out";
  private static final long WAIT_MS = 5_000L;
  private static final long HOLD_MS = 500L;
}
