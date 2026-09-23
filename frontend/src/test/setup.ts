import "@testing-library/jest-dom/vitest";

import { cleanup, configure } from "@testing-library/react";
import { afterEach, beforeEach } from "vitest";

/**
 * Jendela tunggu `findBy*`/`waitFor` dinaikkan dari bawaan 1000ms menjadi 5 detik.
 *
 * Bawaan itu **klaim tentang kecepatan mesin**, bukan tentang aplikasi: dengan 25
 * berkas test berjalan paralel, sebuah promise yang selesai dalam mikrotask tetap
 * dapat tersaji lewat 1 detik bila prosesornya berebut, dan kegagalannya muncul
 * sebagai `Unable to find role="link"` — pesan yang menuduh halaman tidak memuat
 * data padahal yang habis adalah anggaran waktunya. Terukur pada mesin ini: tiga
 * test gagal berpindah-pindah antar-berkas pada `npm run test:run` berulang,
 * sementara `--maxWorkers=2` dan `--maxWorkers=1` hijau sempurna pada kode yang
 * sama. "Coba jalankan ulang" adalah cara kegagalan sungguhan diabaikan, jadi
 * anggarannya dinaikkan alih-alih testnya dilonggarkan: asersinya tidak berubah
 * sama sekali, hanya lamanya ia boleh menunggu (temuan **C-082**).
 */
configure({ asyncUtilTimeout: 5000 });

/**
 * Persiapan lingkungan jsdom.
 *
 * `matchMedia` tidak ada di jsdom, sedangkan store tema memakainya untuk
 * pilihan `system`. Stub di sini membuat jalur itu benar-benar dilatih, bukan
 * dilewati lewat penjaga `typeof`.
 */
beforeEach(() => {
  if (typeof window.matchMedia !== "function") {
    window.matchMedia = ((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    })) as unknown as typeof window.matchMedia;
  }
  window.sessionStorage.clear();
});

afterEach(() => {
  cleanup();
});
