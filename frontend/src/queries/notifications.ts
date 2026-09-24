import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type NotificationListQuery,
} from "@/services/notifications";

export const notificationKeys = {
  all: ["notifications"] as const,
  lists: () => [...notificationKeys.all, "list"] as const,
  list: (query: NotificationListQuery) =>
    [...notificationKeys.lists(), query] as const,
  unreadCount: () => [...notificationKeys.all, "unread-count"] as const,
};

export function useNotificationList(
  query: NotificationListQuery,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: notificationKeys.list(query),
    queryFn: () => listNotifications(query),
    enabled: options.enabled ?? true,
  });
}

/**
 * Badge bell (`50-FSD.md` §8.2): jumlah yang belum dibaca. `limit: 1` karena
 * yang dipakai hanya `meta.total`, bukan barisnya.
 */
export function useUnreadNotificationCount(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: notificationKeys.unreadCount(),
    queryFn: () => listNotifications({ is_read: false, limit: 1 }),
    enabled: options.enabled ?? true,
    refetchInterval: 60_000,
  });
}

export function useMarkNotificationRead() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}

export function useMarkAllNotificationsRead() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => markAllNotificationsRead(),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: notificationKeys.all });
    },
  });
}
