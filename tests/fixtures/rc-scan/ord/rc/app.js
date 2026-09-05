// hardcoded ORD literal — should be resolved from a config/servlet, not baked in
const SERVICE = "station:|slot:/Services/DashboardService";
export function boot() { return SERVICE; }
