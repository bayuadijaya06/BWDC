/** Pemformat tanggal dan identitas. Tidak ada string yang dikarang di sini. */

/**
 * Teks pengganti nilai yang belum ada, dipakai seluruh halaman dari satu tempat
 * supaya tidak ada dua cara mengatakan hal yang sama. Tanda pisah bukan pilihan:
 * em dash dilarang R-02 pada teks yang dibaca pengguna, dan tanda pisah juga
 * tidak menjelaskan apa pun (R-27).
 */
export const EMPTY_VALUE = "Belum diisi";
export const EMPTY_DATE = "Belum ditetapkan";

/**
 * Inisial untuk avatar. Sumbernya `username` atau `name` nyata dari API, bukan
 * foto stok: antislop R-23 melarang mengarang aset identitas.
 */
export function initials(name: string): string {
  const parts = name
    .trim()
    .split(/[\s._-]+/)
    .filter(Boolean);
  if (parts.length === 0) return "?";
  const first = parts[0]?.[0] ?? "?";
  const second = parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? "") : "";
  return (first + second).toUpperCase();
}

/**
 * Stempel waktu RFC 3339 dari API menjadi tanggal-waktu yang dapat dibaca.
 * Zona waktu mengikuti perangkat pembaca, karena `created_at`/`joined_at`
 * dikirim server sebagai instan (ber-offset), bukan tanggal kalender.
 * Tanggal kalender (`start_date`, `target_end_date`) **tidak** lewat sini: ia
 * dikirim `YYYY-MM-DD` dan ditampilkan apa adanya supaya tidak bergeser hari.
 *
 * Nilai yang tidak ada dikembalikan sebagai kalimat, bukan tanda pisah: R-02
 * melarang em dash di teks yang dibaca pengguna, dan tanda pisah juga tidak
 * memberi tahu apa pun (R-27). "Belum dicatat" mengatakan keadaannya.
 */
export function formatTimestamp(value: string | null | undefined): string {
  if (!value) return "Belum dicatat";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Belum dicatat";
  return new Intl.DateTimeFormat("id-ID", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

/**
 * Ukuran berkas untuk dibaca manusia. Basis 1024 mengikuti cara peramban dan
 * penjelajah berkas sistem menampilkan ukuran, sehingga angka di sini tidak
 * berbeda dengan yang dilihat pengguna di tempat lain. Nilainya **hanya
 * tampilan**: batas 100 MB diperiksa server atas byte yang sebenarnya
 * (`42-API.md` §4), bukan atas string ini.
 */
export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "Ukuran tidak diketahui";
  if (bytes < 1024) return `${bytes} byte`;

  const units = ["KB", "MB", "GB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }

  const rounded = value >= 10 ? Math.round(value) : Math.round(value * 10) / 10;
  const text = new Intl.NumberFormat("id-ID", {
    maximumFractionDigits: 1,
  }).format(rounded);
  return `${text} ${units[unit]}`;
}

/** Durasi detik menjadi kalimat tunggu yang dapat dibaca, mis. `15 menit`. */
export function formatWait(seconds: number): string {
  if (seconds <= 0) return "sebentar lagi";
  if (seconds < 60) return `${seconds} detik`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} menit`;
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  return rest === 0 ? `${hours} jam` : `${hours} jam ${rest} menit`;
}
