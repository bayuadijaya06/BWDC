import { useQuery } from "@tanstack/react-query";

import { fetchDashboard, type AnalyticsQuery } from "@/services/analytics";

export const analyticsKeys = {
  all: ["analytics"] as const,
  dashboard: (query: AnalyticsQuery) => [...analyticsKeys.all, "dashboard", query] as const,
};

export function useDashboard(query: AnalyticsQuery) {
  return useQuery({
    queryKey: analyticsKeys.dashboard(query),
    queryFn: () => fetchDashboard(query),
  });
}
