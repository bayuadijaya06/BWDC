import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { listAuditLogs, type AuditListQuery } from "@/services/audit";

export const auditKeys = {
  all: ["audit"] as const,
  list: (query: AuditListQuery) => [...auditKeys.all, "list", query] as const,
};

export function useAuditList(query: AuditListQuery, options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: auditKeys.list(query),
    queryFn: () => listAuditLogs(query),
    placeholderData: keepPreviousData,
    enabled: options.enabled ?? true,
  });
}
