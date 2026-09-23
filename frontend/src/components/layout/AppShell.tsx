import { useRef, useState, type ReactNode } from "react";
import { Outlet } from "react-router";

import {
  TABLET_QUERY,
  useMediaQuery,
  useMediaQueryEnter,
} from "@/hooks/useMediaQuery";
import { useModalLayer } from "@/hooks/useModalLayer";
import { useAuthStore } from "@/store/auth";

import { Header } from "./Header";
import { Sidebar } from "./Sidebar";

/**
 * Kerangka aplikasi (`51-UX.md` §2 dan §8). Tiga state lebar, bukan dua:
 *
 * | Lebar | Yang berlaku |
 * |---|---|
 * | < 768px | sidebar menjadi **laci** yang menutupi konten, satu kolom |
 * | 768-1023px | sidebar **kompak**: label tetap, kolom 12rem |
 * | ≥ 1024px | sidebar penuh (15rem), tetap saat halaman digulir |
 *
 * Laci itu lapisan modal yang sungguhan: ada latar penutup, gulir halaman
 * dikunci, fokus masuk ke laci lalu **kembali** ke tombol Menu, dan `inert`
 * dipasang di konten di belakangnya. Sebelumnya panelnya hidup di dalam baris
 * flex, sehingga membuka menu **menyempitkan** halaman alih-alih menutupinya.
 *
 * State lebar dibaca `useSyncExternalStore` (`@/hooks/useMediaQuery`), bukan
 * disalin ke state lewat effect; yang perlu reaksi terhadap perubahan hanya
 * satunya: melupakan permintaan laci saat jendela melewati titik henti.
 *
 * `51-UX.md` §8 menuliskan state tengah sebagai "sidebar collapsed (icon-only)".
 * Yang diimplementasikan adalah **kompak, label tetap**: ikon-ikon generik
 * dilarang `DESIGN.md` §1 dan R-04, dan rail tanpa label menuntut bahasa ikon
 * yang belum ada padahal label-lah navigasinya. Penyimpangan itu dicatat sebagai
 * temuan **C-067** dan dokumennya diselaraskan, bukan dibiarkan.
 */
export function AppShell({ children }: { children?: ReactNode }) {
  const profile = useAuthStore((state) => state.profile);
  const signOut = useAuthStore((state) => state.signOut);
  const pending = useAuthStore((state) => state.pending);

  const [sidebarOpen, setSidebarOpen] = useState(false);
  const isTablet = useMediaQuery(TABLET_QUERY);
  const panelRef = useRef<HTMLDivElement>(null);
  const menuButtonRef = useRef<HTMLButtonElement>(null);

  // Laci hanya berlaku di bawah 768px: dari 768px ke atas sidebar sudah menjadi
  // kolom tetap.
  const drawerOpen = sidebarOpen && !isTablet;

  // Jendela yang dilebarkan saat laci terbuka menutup lacinya **dan
  // melupakan** permintaannya — kalau hanya yang pertama, menu akan muncul
  // kembali sendiri begitu jendela dipersempit lagi.
  useMediaQueryEnter(TABLET_QUERY, () => setSidebarOpen(false));

  useModalLayer({
    active: drawerOpen,
    containerRef: panelRef,
    onClose: () => setSidebarOpen(false),
    restoreFocusTo: menuButtonRef,
  });

  return (
    <div className="min-h-dvh bg-surface">
      <a
        href="#konten-utama"
        className="sr-only focus:not-sr-only focus:absolute focus:top-1 focus:left-1 focus:z-40 focus:rounded-control focus:border focus:border-line-strong focus:bg-surface-raised focus:px-2 focus:py-1 focus:text-13"
      >
        Lewati ke konten utama
      </a>

      <Header
        profile={profile}
        onSignOut={(all) => void signOut(all)}
        signingOut={pending}
        sidebarOpen={drawerOpen}
        onToggleSidebar={() => setSidebarOpen((value) => !value)}
        toggleRef={menuButtonRef}
      />

      <div className="flex">
        {/* `div`, bukan `aside`: `role="dialog"` tidak diizinkan di atas elemen
            yang sudah punya peran implisit `complementary` (axe:
            `aria-allowed-role`). Saat menjadi kolom, yang penting adalah
            landmark `nav` di dalamnya; saat menjadi laci, panel ini memang
            lapisan modal dan perannya harus benar-benar `dialog`. */}
        <div
          id="sidebar-panel"
          ref={panelRef}
          role={drawerOpen ? "dialog" : undefined}
          aria-modal={drawerOpen ? true : undefined}
          aria-label={drawerOpen ? "Menu utama" : undefined}
          tabIndex={-1}
          className={[
            "border-line bg-surface-raised",
            // 768px ke atas: kolom tetap yang lebih sempit dari versi penuh.
            "md:block md:w-48 md:shrink-0 md:overflow-y-auto md:border-r",
            // 1024px ke atas: penuh, dan ikut menempel saat halaman digulir.
            "lg:sticky lg:top-12 lg:h-[calc(100dvh-3rem)] lg:w-60",
            drawerOpen
              ? "fixed inset-y-0 left-0 z-40 block w-[min(18rem,85vw)] overflow-y-auto border-r shadow-lift"
              : "hidden",
          ].join(" ")}
        >
          {drawerOpen ? (
            <div className="flex items-center justify-between border-b border-line px-3 py-2 md:hidden">
              <span className="text-13 font-medium text-text">Menu</span>
              <button
                type="button"
                onClick={() => setSidebarOpen(false)}
                className="tap-target rounded-control border border-line-strong px-2.5 text-12 text-text hover:bg-surface-hover"
              >
                Tutup
              </button>
            </div>
          ) : null}

          <Sidebar
            profile={profile}
            onNavigate={() => setSidebarOpen(false)}
            autoFocusFirst={drawerOpen}
          />
        </div>

        {drawerOpen ? (
          <div
            // Penanda eksplisit, bukan posisi di DOM: test jsdom dan skrip bukti
            // tata letak (`scripts/responsive-evidence.mjs`) mencari latar ini
            // dengan namanya, sehingga urutan elemen dapat berubah tanpa
            // membuat kedua pemeriksa itu mengukur elemen yang salah.
            data-drawer-backdrop="true"
            aria-hidden="true"
            onClick={() => setSidebarOpen(false)}
            className="fixed inset-0 z-30 bg-ink-900/35 md:hidden"
          />
        ) : null}

        {/* `inert` menahan fokus dan pembaca layar keluar dari konten yang
            sedang tertutup laci; jebakan fokus di `useModalLayer` menjaga sisi
            papan ketik. */}
        <main
          id="konten-utama"
          inert={drawerOpen ? true : undefined}
          className="min-w-0 flex-1 px-4 py-4 lg:px-6"
        >
          <div className="mx-auto flex max-w-[1100px] flex-col gap-5">
            {children ?? <Outlet />}
          </div>
        </main>
      </div>
    </div>
  );
}
