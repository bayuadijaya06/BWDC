import { useCallback, useEffect, useRef, useSyncExternalStore } from "react";

/**
 * Titik henti bernama, dipakai bersama oleh CSS (kelas `md:`/`lg:`) dan oleh
 * JavaScript yang perlu tahu state lebar yang sedang berlaku.
 *
 * Nilainya adalah **skala yang sama** dengan Tailwind (`48rem` = `md`,
 * `64rem` = `lg`), karena kalau angka di JS dan di CSS berbeda, tata letaknya
 * menyimpang tepat di sekitar titik henti — dan itu gagal R-03 tanpa terlihat
 * di satu pun dari dua lebar contoh. `51-UX.md` §8 menulis angkanya dalam px
 * (768 / 1024) karena itu bahasa yang dipakai dokumen.
 */
export const TABLET_QUERY = "(min-width: 48rem)";
export const DESKTOP_QUERY = "(min-width: 64rem)";

/**
 * Membaca satu media query dan ikut berubah saat nilainya berubah.
 *
 * Dibaca lewat `useSyncExternalStore`, bukan `useEffect` + `useState`: media
 * query adalah **sumber luar**, dan React menyediakan cara resmi membacanya
 * pada render yang sama. Versi lama menyalin nilainya ke state di dalam effect,
 * sehingga render pertama selalu memakai tebakan lalu render kedua memperbaiki
 * — persis efek berantai yang diperingatkan `react-hooks/set-state-in-effect`.
 *
 * Dipakai untuk hal yang **tidak** dapat diselesaikan CSS murni: perilaku modal
 * (kunci gulir, jebakan fokus, `inert`) hanya berlaku saat sidebar sedang
 * menjadi laci, bukan saat ia kolom tetap. Tanpa hook ini, laci yang ditinggal
 * terbuka sambil jendela dilebarkan akan tetap mengunci gulir halaman padahal
 * laci sudah menjadi kolom biasa.
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (onChange: () => void) => subscribeQuery(query, onChange),
    [query],
  );
  const getSnapshot = useCallback(() => readMatch(query), [query]);

  return useSyncExternalStore(subscribe, getSnapshot, () => false);
}

/**
 * Menjalankan `onEnter` ketika lebar **masuk** ke query ini (dari tidak cocok
 * menjadi cocok) — bukan saat nilainya berubah dua arah.
 *
 * Dipakai untuk membereskan state yang hanya berlaku di layar sempit: laci menu
 * yang ditinggal terbuka sambil jendela dilebarkan harus ditutup, supaya saat
 * jendela dipersempit lagi menunya tidak muncul kembali tanpa diminta. State
 * ditulis di dalam callback langganan (sumber luar), sehingga tidak ada
 * `setState` di badan effect.
 */
export function useMediaQueryEnter(query: string, onEnter: () => void): void {
  const onEnterRef = useRef(onEnter);
  useEffect(() => {
    onEnterRef.current = onEnter;
  });

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function")
      return;

    const list = window.matchMedia(query);
    const onChange = (event: MediaQueryListEvent) => {
      if (event.matches) onEnterRef.current();
    };
    list.addEventListener("change", onChange);
    return () => list.removeEventListener("change", onChange);
  }, [query]);
}

function subscribeQuery(query: string, onChange: () => void): () => void {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function")
    return () => {};

  const list = window.matchMedia(query);
  list.addEventListener("change", onChange);
  return () => list.removeEventListener("change", onChange);
}

function readMatch(query: string): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function")
    return false;
  return window.matchMedia(query).matches;
}
