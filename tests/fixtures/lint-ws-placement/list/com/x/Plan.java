package com.x;

// Not slot adds of a new component: collection adds, value-typed properties, addAll.
// One shape per method, and no method mentions the placement slot, so none is silenced by the guard.
public class Plan
{
  void collect(List<BComponent> out)
  {
    out.add(new BFolder());
    out.add(0, new BFolder());
  }

  void values(BComponent c)
  {
    c.add("status", new BStatusNumeric());
    c.add("since", new BAbsTime());
  }

  void bulk(BComponent c)
  {
    c.addAll(items, new BFolder());
  }
}
