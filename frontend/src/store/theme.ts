import { create } from "zustand";

/**
 * Tema. Pilihannya tiga (`system` ikut preferensi OS) dan yang disimpan hanya
 * pilihan pengguna, bukan hasil resolusinya, supaya pengguna yang memilih
 * `system` tetap ikut berubah saat OS berganti tema.
 *
 * `DESIGN.md` §7 mewajibkan kedua mode benar-benar berfungsi (R-21 & R-34),
 * karena itu nilainya diterapkan ke `data-theme` di <html> dan seluruh warna
 * berasal dari token.
 */

export type ThemeChoice = "system" | "light" | "dark";
export type ResolvedTheme = "light" | "dark";

const STORAGE_KEY = "bwdcs.theme";

function readStoredChoice(): ThemeChoice {
  try {
    const value = window.localStorage.getItem(STORAGE_KEY);
    if (value === "light" || value === "dark" || value === "system")
      return value;
  } catch {
    // Storage tidak tersedia: jatuh ke `system`.
  }
  return "system";
}

function writeStoredChoice(choice: ThemeChoice): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, choice);
  } catch {
    // Diabaikan: preferensi tema bukan data penting.
  }
}

function systemPrefers(): ResolvedTheme {
  if (typeof window.matchMedia !== "function") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function resolve(choice: ThemeChoice): ResolvedTheme {
  return choice === "system" ? systemPrefers() : choice;
}

function apply(theme: ResolvedTheme): void {
  document.documentElement.dataset.theme = theme;
}

interface ThemeState {
  choice: ThemeChoice;
  resolved: ResolvedTheme;
  setChoice: (choice: ThemeChoice) => void;
  /** Dipasang sekali dari `main.tsx`; mengembalikan fungsi pembersih. */
  watchSystem: () => () => void;
}

const initialChoice = readStoredChoice();
const initialResolved = resolve(initialChoice);
apply(initialResolved);

export const useThemeStore = create<ThemeState>((set, get) => ({
  choice: initialChoice,
  resolved: initialResolved,

  setChoice: (choice) => {
    const resolved = resolve(choice);
    writeStoredChoice(choice);
    apply(resolved);
    set({ choice, resolved });
  },

  watchSystem: () => {
    if (typeof window.matchMedia !== "function") return () => {};
    const query = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      if (get().choice !== "system") return;
      const resolved = systemPrefers();
      apply(resolved);
      set({ resolved });
    };
    query.addEventListener("change", onChange);
    return () => query.removeEventListener("change", onChange);
  },
}));
