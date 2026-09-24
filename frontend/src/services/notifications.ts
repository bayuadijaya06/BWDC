import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";

import { http } from "./http";

/**
 * Lapisan API modul notification. Sumber kontrak: `docs/design/42-API.md` §8.
 *
 * Dua hal yang sengaja **tidak** dikerjakan di sini:
 *
 * - **Cakupan tidak dihitung klien.** `GET /notifications` hanya mengembalikan
 *   baris `user_id = user` (`44-SECURITY.md` §3.1.3) — bahkan Administrator
 *   tidak dapat membaca notifikasi orang lain, dan `?is_read=` tidak mengubah
 *   cakupan itu.
 * - **ID orang lain tidak dibedakan dari tidak ada.** `PATCH /:id/read` atas
 *   milik user lain dibalas `404`, sama seperti baris yang tidak ada.
 */

export interface NotificationItem {
  id: string;
  type: string;
  title: string;
  message: string;
  entity_id: string | null;
  entity_type: string | null;
  is_read: boolean;
  created_at: string;
}

export interface NotificationListQuery {
  page?: number;
  limit?: number;
  /** `undefined` berarti tanpa penyaring (semua). */
  is_read?: boolean;
}

export interface NotificationListResult {
  items: NotificationItem[];
  meta: ApiMeta;
}

export async function listNotifications(
  query: NotificationListQuery = {},
): Promise<NotificationListResult> {
  const params: Record<string, string | number | boolean> = {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  };
  if (query.is_read !== undefined) params.is_read = query.is_read;

  const response = await http.get<ApiSuccess<NotificationItem[]>>(
    "/notifications",
    { params },
  );
  return {
    items: response.data.data,
    meta: fallbackMeta(response.data.meta),
  };
}

export async function markNotificationRead(id: string): Promise<void> {
  await http.patch(`/notifications/${encodeURIComponent(id)}/read`);
}

export async function markAllNotificationsRead(): Promise<number> {
  const response = await http.post<ApiSuccess<{ updated: number }>>(
    "/notifications/read-all",
  );
  return response.data.data.updated;
}

/**
 * Tujuan navigasi "Click → Navigate to related entity" (`50-FSD.md` §8.2).
 * Hanya entitas yang punya halaman yang dipetakan; `comment` dan yang tidak
 * dikenal mengembalikan `null` — mengekliknya hanya menandai dibaca, bukan
 * menavigasi ke halaman yang salah.
 */
export function notificationTarget(item: NotificationItem): string | null {
  if (!item.entity_id || !item.entity_type) return null;
  const id = encodeURIComponent(item.entity_id);
  switch (item.entity_type) {
    case "document":
      return `/documents/${id}`;
    case "task":
      return `/tasks/${id}`;
    case "project":
      return `/projects/${id}`;
    case "workflow":
    case "workflow_instance":
      return `/approvals/${id}`;
    default:
      return null;
  }
}
