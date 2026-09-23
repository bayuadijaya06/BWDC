/**
 * Menyimpan berkas dari memori.
 *
 * Unduhan dokumen **tidak** dapat berupa `<a href="/api/v1/documents/:id/download/:versionId">`:
 * endpoint itu menuntut header `Authorization: Bearer`, dan tautan biasa tidak
 * membawa header apa pun — akibatnya pengguna menerima `401` di tab baru.
 * Karena itu berkasnya diambil lebih dulu sebagai blob (`GET` ber-token), lalu
 * diserahkan ke peramban lewat objek URL.
 *
 * Nama berkas datang dari `Content-Disposition` server
 * (`42-API.md` §4), bukan dikarang dari judul dokumen: server sudah mengirim
 * nama asli yang diunggah, termasuk `filename*=UTF-8''…` untuk nama non-ASCII.
 */
export function saveBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.rel = "noopener";
  anchor.style.display = "none";
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  // Objek URL baru dilepas setelah peramban sempat memulai unduhan; melepasnya
  // seketika membatalkan berkas yang belum selesai dibaca (Safari paling
  // sensitif soal ini).
  window.setTimeout(() => URL.revokeObjectURL(url), 10_000);
}

/**
 * Membaca nama berkas dari header `Content-Disposition`.
 *
 * Urutannya mengikuti RFC 5987: bentuk `filename*=UTF-8''…` lebih dipercaya
 * daripada `filename="…"` karena yang kedua hanya cadangan ASCII dan pada nama
 * berkarakter non-ASCII berisi versi yang sudah dilucuti. Bila keduanya tidak
 * ada, nama cadangan dari pemanggil dipakai apa adanya — tidak ada nama berkas
 * yang dikarang.
 */
export function filenameFromDisposition(
  header: string | null | undefined,
  fallback: string,
): string {
  if (!header) return fallback;

  const extended = /filename\*\s*=\s*utf-8''([^;]+)/i.exec(header);
  if (extended?.[1]) {
    const raw = extended[1].trim().replace(/^"|"$/g, "");
    try {
      return decodeURIComponent(raw);
    } catch {
      return raw;
    }
  }

  const plain = /filename\s*=\s*"([^"]*)"/i.exec(header);
  if (plain?.[1]) return plain[1].trim();

  const bare = /filename\s*=\s*([^;]+)/i.exec(header);
  if (bare?.[1]) return bare[1].trim().replace(/^"|"$/g, "");

  return fallback;
}
