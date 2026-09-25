/**
 * Penyimpan token. Tidak ada token di storage mana pun.
 *
 * Keputusan dan alasannya (Q-021 diputuskan cookie, ADR-0033, P-086):
 * - **Access token hanya di memori.** Ia tidak pernah masuk `localStorage`
 *   maupun `sessionStorage`, sehingga skrip yang berhasil masuk ke halaman
 *   tidak menemukannya di penyimpanan.
 * - **Refresh token di cookie `HttpOnly` milik server.** Backend memasangnya
 *   (`Set-Cookie`) saat login/refresh/change-password dan membacanya saat
 *   refresh; peramban melampirkannya otomatis (`withCredentials: true`).
 *   Berkas ini tidak pernah menyentuh cookie itu — tidak bisa, dan tidak perlu.
 * - Karena klien tidak dapat melihat cookie, "masih ada sesi tersimpan"
 *   tidak dapat diperiksa lokal: `restore` selalu mencoba refresh sekali,
 *   dan anonim adalah jawaban 401-nya, bukan keadaan awal yang diasumsikan.
 */

let accessToken: string | null = null;

export const session = {
  getAccessToken(): string | null {
    return accessToken;
  },

  setTokens(access: string): void {
    accessToken = access;
  },

  clear(): void {
    accessToken = null;
  },

  hasStoredSession(): boolean {
    return accessToken !== null;
  },
};
