import { useQuery } from "@tanstack/react-query";

import { fetchDashboard, fetchDepartments, type AnalyticsQuery } from "@/services/analytics";

export const analyticsKeys = {
  all: ["analytics"] as const,
  dashboard: (query: AnalyticsQuery) => [...analyticsKeys.all, "dashboard", query] as const,
  departments: () => [...analyticsKeys.all, "departments"] as const,
};

export function useDashboard(query: AnalyticsQuery) {
  return useQuery({
    queryKey: analyticsKeys.dashboard(query),
    queryFn: () => fetchDashboard(query),
  });
}

export function useDepartments(enabled = true) {
  return useQuery({
    queryKey: analyticsKeys.departments(),
    queryFn: fetchDepartments,
    enabled,
  });
}
