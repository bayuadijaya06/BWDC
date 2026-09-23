import { useEffect, useRef, type RefObject } from "react";

/** Elemen yang dapat menerima fokus di dalam lapisan modal. */
const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * Perilaku lapisan modal yang dipakai **bersama** oleh `Dialog` dan laci menu
 * di `AppShell`. Ditulis sekali karena inilah bagian yang paling mudah
 * dilupakan, dan versi yang terlupakan selalu yang sama: fokus tidak kembali,
 * Tab keluar dari lapisan, halaman di belakang tetap dapat digulir.
 *
 * Kontraknya:
 * - fokus dipindahkan ke elemen `[data-autofocus]`, lalu kontrol pertama, lalu
 *   ke wadahnya sendiri (R-32, WCAG 2.4.3);
 * - Tab dan Shift+Tab berputar di dalam wadah, sehingga fokus tidak pernah
 *   masuk ke konten di belakangnya;
 * - Escape menutup, dan kejadiannya dihentikan supaya tidak menutup lapisan
 *   lain di belakangnya;
 * - gulir halaman dikunci selama lapisan aktif;
 * - saat ditutup, fokus **dikembalikan** ke elemen yang membukanya.
 */
export function useModalLayer({
  active,
  containerRef,
  onClose,
  restoreFocusTo,
}: {
  active: boolean;
  containerRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  /** Elemen yang menerima kembali fokus saat ditutup; default: yang tadi fokus. */
  restoreFocusTo?: RefObject<HTMLElement | null>;
}): void {
  // Callback sering berupa arrow baru tiap render; ref-nya diperbarui di effect
  // supaya listener tidak dipasang ulang setiap kali komponen render.
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  });

  useEffect(() => {
    if (!active) return;

    const previouslyFocused = document.activeElement as HTMLElement | null;

    const focusables = () =>
      Array.from(containerRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? []);

    const first =
      containerRef.current?.querySelector<HTMLElement>("[data-autofocus]") ??
      focusables()[0] ??
      containerRef.current;
    first?.focus();

    // Tujuan fokus dibaca **di dalam** effect, bukan di pembersihnya: nilai
    // `ref.current` dapat berubah sebelum pembersihan, dan yang benar untuk
    // dikembalikan adalah elemen yang tadi membukanya.
    const restoreTarget = restoreFocusTo?.current ?? previouslyFocused;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.stopPropagation();
        onCloseRef.current();
        return;
      }
      if (event.key !== "Tab") return;

      const items = focusables();
      if (items.length === 0) return;
      const firstItem = items[0]!;
      const lastItem = items[items.length - 1]!;
      const current = document.activeElement;

      if (
        event.shiftKey &&
        (current === firstItem || current === containerRef.current)
      ) {
        event.preventDefault();
        lastItem.focus();
      } else if (!event.shiftKey && current === lastItem) {
        event.preventDefault();
        firstItem.focus();
      }
    };

    document.addEventListener("keydown", onKeyDown, true);
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", onKeyDown, true);
      document.body.style.overflow = previousOverflow;
      restoreTarget?.focus?.();
    };
  }, [active, containerRef, restoreFocusTo]);
}
