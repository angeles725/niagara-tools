package com.x;

// Shape of DashboardFord BFordDashboardService.doSyncInventory: creates every missing node, places none.
public class BSyncService extends BAbstractService
{
  public void doSyncInventory()
  {
    for (Op op : plan.ops)
    {
      BComponent parent = resolve(op.parent);
      if (parent == null) continue;
      parent.add(op.name, create(op.type));
    }
  }

  private static BComponent create(String type)
  {
    return (BComponent) Sys.getType(type).getInstance();
  }
}
