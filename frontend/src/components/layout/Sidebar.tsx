import { NavLink } from "react-router";

import { requiresExactMatch, visibleNavigation } from "@/config/navigation";
import type { UserProfile } from "@/services/auth";

/**
 * Sidebar. Lima hal yang disengaja:
 * - menu mengikuti izin user (`GET /auth/me`), bukan nama role;
 * - modul yang halamannya belum dibangun ditandai teks kecil `belum`, bukan
 *   badge dekoratif (R-09), dan tautannya tetap hidup (R-24);
 * - penanda menu aktif memakai accent tunggal (Signal), sesuai `DESIGN.md` §2.3;
 * - setiap tautan memakai `tap-target`, sehingga di layar sentuh tingginya
 *   44px dan jarak antar item tidak menempel (R-03, `51-UX.md` §9);
 * - **satu entri per modul**, tanpa sub-menu: tab dan penyaring modul ada di
 *   halamannya sendiri (`51-UX.md` §2.1), sehingga daftar menu tidak
 *   memanjang oleh pilihan yang sudah tersedia di tempat yang lebih dekat.
 *
 * `end` dihitung dari daftar path itu sendiri (`requiresExactMatch`), bukan
 * ditulis tangan: kalau tidak, `/reports` ikut menyala saat yang dibuka
 * `/reports/audit`, dan dua menu mengaku sebagai halaman yang sedang dibuka.
 *
 * `autoFocusFirst` dipakai laci menu di layar sempit: fokus harus masuk ke
 * laci saat dibuka, dan item pertama adalah tujuan yang paling mungkin.
 */
export function Sidebar({
  profile,
  onNavigate,
  autoFocusFirst = false,
}: {
  profile: UserProfile | null;
  onNavigate?: () => void;
  autoFocusFirst?: boolean;
}) {
  const items = visibleNavigation(profile?.permissions ?? []);

  return (
    <nav
      aria-label="Menu utama"
      className="flex flex-col gap-1 px-2 py-3 lg:gap-0.5"
    >
      {items.map((item, index) => (
        <NavLink
          key={item.path}
          to={item.path}
          end={requiresExactMatch(item)}
          onClick={onNavigate}
          data-autofocus={autoFocusFirst && index === 0 ? true : undefined}
          className={({ isActive }) =>
            [
              "tap-target flex items-center justify-between rounded-control border-l-2 px-2.5 text-13",
              isActive
                ? "border-accent bg-accent-soft font-medium text-text"
                : "border-transparent text-text-soft hover:bg-surface-hover hover:text-text",
            ].join(" ")
          }
        >
          <span>{item.label}</span>
          {item.status === "pending" ? (
            <span className="pl-2 text-12 text-text-muted">belum</span>
          ) : null}
        </NavLink>
      ))}
    </nav>
  );
}
