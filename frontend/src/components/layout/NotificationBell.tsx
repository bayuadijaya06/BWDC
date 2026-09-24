import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { ApiError } from "@/services/http";
import {
  notificationTarget,
  type NotificationItem,
} from "@/services/notifications";
import {
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useNotificationList,
  useUnreadNotificationCount,
} from "@/queries/notifications";
import { formatTimestamp } from "@/utils/format";

/**
 * Bell notifikasi (`50-FSD.md` §8.2): badge jumlah belum dibaca + dropdown
 * daftar dengan penyaring `is_read` + tandai dibaca satu/ semua.
 *
 * Pola dropdown mengikuti `UserMenu` di `Header.tsx`: Escape menutup, klik di
 * luar menutup. Daftar hanya diminta saat dropdown dibuka, sedangkan badge
 * dihitung dari kueri ringan (`limit: 1`, yang dipakai `meta.total`-nya).
 */
export function NotificationBell() {
  const [open, setOpen] = useState(false);
  const [unreadOnly, setUnreadOnly] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const unread = useUnreadNotificationCount();
  const unreadTotal = unread.data?.meta.total ?? 0;

  const list = useNotificationList(
    { limit: 20, ...(unreadOnly ? { is_read: false } : {}) },
    { enabled: open },
  );
  const markRead = useMarkNotificationRead();
  const markAllRead = useMarkAllNotificationsRead();

  useEffect(() => {
    if (!open) return;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    const onPointerDown = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
    };

    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("mousedown", onPointerDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("mousedown", onPointerDown);
    };
  }, [open ]);

  async function openItem(item: NotificationItem) {
    if (!item.is_read) {
      try {
        await markRead.mutateAsync(item.id);
      } catch {
        // Gagal menandai tidak menghalangi navigasi: daftar akan menunjukkan
        // keadaan sebenarnya saat kuerinya disegarkan.
      }
    }
    const target = notificationTarget(item);
    if (target !== null) {
      setOpen(false);
      navigate(target);
    }
  }

  const rows = list.data?.items ?? [];

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-expanded={open}
        aria-label={
          unreadTotal > 0
            ? `Notifikasi, ${unreadTotal} belum dibaca`
            : "Notifikasi, tidak ada yang belum dibaca"
        }
        onClick={() => setOpen((value) => !value)}
        className="tap-target relative inline-flex items-center rounded-control border border-line-strong px-2.5 text-12 text-text hover:bg-surface-hover"
      >
        Notifikasi
        {unreadTotal > 0 ? (
          <span
            aria-hidden="true"
            className="ml-1.5 inline-flex min-h-5 min-w-5 items-center justify-center rounded-control border border-text bg-text px-1 text-11 font-semibold text-surface-raised"
          >
            {unreadTotal > 99 ? "99+" : unreadTotal}
          </span>
        ) : null}
      </button>

      {open ? (
        // `region`, bukan `menu`: isinya campuran penyaring, aksi, dan daftar.
        // Peran menu menuntut anak menuitem dan induk menu (temuan axe).
        <div
          role="region"
          aria-label="Notifikasi"
          className="absolute right-0 z-20 mt-1.5 flex max-h-[70vh] w-80 flex-col rounded-panel border border-line bg-surface-raised py-1.5 text-13"
        >
          <div className="flex items-center justify-between gap-2 border-b border-line px-3 pb-2">
            <div className="flex gap-1" role="group" aria-label="Penyaring notifikasi">
              <button
                type="button"
                aria-pressed={!unreadOnly}
                onClick={() => setUnreadOnly(false)}
                className={[
                  "tap-target rounded-control px-2 text-12",
                  !unreadOnly
                    ? "bg-surface-sunken font-medium text-text"
                    : "text-text-soft hover:bg-surface-hover",
                ].join(" ")}
              >
                Semua
              </button>
              <button
                type="button"
                aria-pressed={unreadOnly}
                onClick={() => setUnreadOnly(true)}
                className={[
                  "tap-target rounded-control px-2 text-12",
                  unreadOnly
                    ? "bg-surface-sunken font-medium text-text"
                    : "text-text-soft hover:bg-surface-hover",
                ].join(" ")}
              >
                Belum dibaca
              </button>
            </div>
            <button
              type="button"
              disabled={markAllRead.isPending || unreadTotal === 0}
              onClick={() => void markAllRead.mutateAsync()}
              className="tap-target rounded-control px-2 text-12 text-text-soft hover:bg-surface-hover hover:text-text disabled:opacity-45"
            >
              {markAllRead.isPending ? "Memproses" : "Tandai semua dibaca"}
            </button>
          </div>

          <div className="overflow-y-auto">
            {list.isPending ? (
              <p role="status" className="px-3 py-3 text-13 text-text-muted">
                Memuat notifikasi…
              </p>
            ) : list.error instanceof ApiError ? (
              <div className="flex flex-col gap-2 px-3 py-3">
                <p role="alert" className="text-13 text-danger">
                  {list.error.message}
                </p>
                <div>
                  <button
                    type="button"
                    onClick={() => void list.refetch()}
                    className="tap-target rounded-control border border-line-strong px-2.5 text-12 text-text hover:bg-surface-hover"
                  >
                    Muat ulang
                  </button>
                </div>
              </div>
            ) : rows.length === 0 ? (
              <p role="status" className="px-3 py-3 text-13 text-text-muted">
                {unreadOnly
                  ? "Tidak ada notifikasi yang belum dibaca."
                  : "Belum ada notifikasi untuk Anda."}
              </p>
            ) : (
              <ul className="flex flex-col">
                {rows.map((item) => (
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => void openItem(item)}
                      className="tap-target block w-full px-3 text-left hover:bg-surface-hover"
                    >
                      <span className="flex items-start gap-2">
                        {!item.is_read ? (
                          <span
                            aria-hidden="true"
                            className="mt-1.5 size-2 shrink-0 rounded-full bg-accent"
                          />
                        ) : null}
                        <span className="flex min-w-0 flex-col gap-0.5 py-0.5">
                          <span
                            className={[
                              "text-13",
                              item.is_read ? "text-text-soft" : "font-medium text-text",
                            ].join(" ")}
                          >
                            {!item.is_read ? (
                              <span className="sr-only">belum dibaca: </span>
                            ) : null}
                            {item.title}
                          </span>
                          <span className="text-12 text-text-muted">
                            {item.message}
                          </span>
                          <span className="text-11 text-text-muted">
                            {formatTimestamp(item.created_at)}
                          </span>
                        </span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}
