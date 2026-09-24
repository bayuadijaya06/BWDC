import { useEffect, useRef, useState, type Ref } from "react";

import type { UserProfile } from "@/services/auth";
import { useThemeStore, type ThemeChoice } from "@/store/theme";
import { initials } from "@/utils/format";

import { NotificationBell } from "./NotificationBell";

/**
 * Header. Berisi wordmark teks (BWDCS tidak punya berkas logo, dan aset tidak
 * boleh dikarang: `DESIGN.md` §1), tombol menu untuk layar sempit, pengalih
 * tema, dan menu pengguna.
 *
 * Tidak ada kolom pencarian global di sini: `51-UX.md` §2.2 menandainya
 * opsional, dan tombol yang belum bisa mencari apa pun akan melanggar R-26.
 */

const themeLabels: Record<ThemeChoice, string> = {
  system: "Tema: ikut sistem",
  light: "Tema: terang",
  dark: "Tema: gelap",
};

/**
 * Label pendek untuk layar sempit, tempat label penuh akan mendesak nama akun
 * keluar dari viewport. Teks panjang dan pendek sama-sama dirender: yang satu
 * disembunyikan CSS, dan pembaca layar memakai `aria-label` yang penuh. Label
 * pendek sengaja tetap menjadi **kata dari** label penuh, supaya nama yang
 * terbaca tidak berbeda dari yang terlihat.
 */
const themeShortLabels: Record<ThemeChoice, string> = {
  system: "Sistem",
  light: "Terang",
  dark: "Gelap",
};

function nextChoice(choice: ThemeChoice): ThemeChoice {
  if (choice === "system") return "light";
  if (choice === "light") return "dark";
  return "system";
}

function ThemeToggle() {
  const choice = useThemeStore((state) => state.choice);
  const setChoice = useThemeStore((state) => state.setChoice);

  return (
    <button
      type="button"
      onClick={() => setChoice(nextChoice(choice))}
      aria-label={themeLabels[choice]}
      className="tap-target rounded-control border border-line-strong px-2 text-12 text-text-soft hover:bg-surface-hover"
    >
      <span className="sm:hidden">{themeShortLabels[choice]}</span>
      <span className="hidden sm:inline">{themeLabels[choice]}</span>
    </button>
  );
}

function UserMenu({
  profile,
  onSignOut,
  pending,
}: {
  profile: UserProfile;
  onSignOut: (all: boolean) => void;
  pending: boolean;
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

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
  }, [open]);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        className="tap-target flex items-center gap-2 rounded-control border border-line-strong px-2 text-12 text-text hover:bg-surface-hover"
      >
        <span
          aria-hidden="true"
          className="flex size-5 shrink-0 items-center justify-center rounded-control bg-surface-sunken text-12 font-medium"
        >
          {initials(profile.username)}
        </span>
        {/* Nama akun tetap terlihat (nama yang terbaca tidak boleh berbeda dari
            yang terlihat), tetapi dipotong di layar sempit supaya header tidak
            melebar melewati viewport. */}
        <span className="mono max-w-[6rem] truncate sm:max-w-none">
          {profile.username}
        </span>
      </button>

      {open ? (
        <div
          role="menu"
          aria-label="Menu pengguna"
          className="absolute right-0 z-20 mt-1.5 w-64 rounded-panel border border-line bg-surface-raised py-1.5 text-13"
        >
          <div className="border-b border-line px-3 pb-2">
            <p className="mono text-13 text-text">{profile.username}</p>
            <p className="text-12 text-text-muted">{profile.email}</p>
            <p className="pt-1 text-12 text-text-muted">
              {profile.roles.length > 0
                ? profile.roles.join(", ")
                : "tanpa role"}
            </p>
          </div>
          <button
            type="button"
            role="menuitem"
            disabled={pending}
            onClick={() => onSignOut(false)}
            className="tap-target block w-full px-3 text-left hover:bg-surface-hover disabled:opacity-55"
          >
            Keluar
          </button>
          <button
            type="button"
            role="menuitem"
            disabled={pending}
            onClick={() => onSignOut(true)}
            className="tap-target block w-full px-3 text-left hover:bg-surface-hover disabled:opacity-55"
          >
            Keluar dari semua perangkat
          </button>
        </div>
      ) : null}
    </div>
  );
}

export function Header({
  profile,
  onSignOut,
  signingOut,
  onToggleSidebar,
  sidebarOpen,
  toggleRef,
}: {
  profile: UserProfile | null;
  onSignOut: (all: boolean) => void;
  signingOut: boolean;
  onToggleSidebar: () => void;
  sidebarOpen: boolean;
  /** Tombol Menu: dipakai laci untuk mengembalikan fokus saat ditutup. */
  toggleRef?: Ref<HTMLButtonElement>;
}) {
  return (
    <header className="sticky top-0 z-30 flex h-12 items-center justify-between gap-2 border-b border-line bg-surface-raised px-3">
      <div className="flex min-w-0 items-center gap-2">
        <button
          type="button"
          ref={toggleRef}
          onClick={onToggleSidebar}
          aria-expanded={sidebarOpen}
          aria-controls="sidebar-panel"
          className="tap-target shrink-0 rounded-control border border-line-strong px-2.5 text-12 text-text hover:bg-surface-hover md:hidden"
        >
          Menu
        </button>
        <span className="text-14 font-semibold tracking-tight">BWDCS</span>
        <span className="hidden text-12 text-text-muted sm:inline">
          Kendali dokumen dan alur kerja
        </span>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        <ThemeToggle />
        {profile && profile.permissions.includes("notification:read") ? (
          <NotificationBell />
        ) : null}
        {profile ? (
          <UserMenu
            profile={profile}
            onSignOut={onSignOut}
            pending={signingOut}
          />
        ) : null}
      </div>
    </header>
  );
}
