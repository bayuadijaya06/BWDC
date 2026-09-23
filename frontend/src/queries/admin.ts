import { useQuery } from "@tanstack/react-query";

import { listAdminUsers, type AdminUserListQuery } from "@/services/admin";

export function useAdminUsers(query: AdminUserListQuery, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ["admin", "users", query],
    queryFn: () => listAdminUsers(query),
    enabled: options?.enabled ?? true,
  });
}
