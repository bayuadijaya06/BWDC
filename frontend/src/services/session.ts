/**
 * Penyimpan token. Satu-satunya berkas yang menyentuh `sessionStorage`.
 *
 * Keputusan dan alasannya:
 * - **Access token hanya di memori.** Ia tidak pernah masuk `localStorage`
 *   maupun `sessionStorage`, sehingga skrip yang berhasil masuk ke halaman
 *   tidak menemukannya di penyimpanan.
 * - **Refresh token di `sessionStorage` sebagai jembatan.** Backend
 *   mengembalikan refresh token di body (`42-API.md` §2, ADR-0023) dan tidak
 *   memasang cookie, jadi klien belum punya tempat yang lebih baik. Praktik
 *   yang dituju adalah refresh token di cookie `HttpOnly` + `Secure` +
 *   `SameSite`, dan itu perubahan kontrak backend yang dicatat sebagai
 *   pertanyaan terbuka di `docs/progress/OPEN-QUESTIONS.md` (Q-021).
 *   `sessionStorage` dipilih daripada `localStorage` supaya sesinya berakhir
 *   saat tab ditutup.
 * - Berkas ini satu-satunya tempat yang perlu diubah bila kelak backend
 *   memasang cookie.
 */

const REFRESH_KEY = "bwdcs.refresh_token";

let accessToken: string | null = null;
let refreshToken: string | null = null;

function readStoredRefreshToken(): string | null {
  try {
    return window.sessionStorage.getItem(REFRESH_KEY);
  } catch {
    // Mode privat atau storage diblokir: sesi tetap jalan sampai reload.
    return null;
  }
}

function writeStoredRefreshToken(value: string | null): void {
  try {
    if (value === null) {
      window.sessionStorage.removeItem(REFRESH_KEY);
      return;
    }
    window.sessionStorage.setItem(REFRESH_KEY, value);
  } catch {
    // Diabaikan dengan sengaja: kegagalan storage tidak boleh mematikan login.
  }
}

export const session = {
  /** Dipanggil sekali saat modul dimuat. */
  init(): void {
    refreshToken = readStoredRefreshToken();
  },

  getAccessToken(): string | null {
    return accessToken;
  },

  getRefreshToken(): string | null {
    return refreshToken;
  },

  setTokens(access: string, refresh: string): void {
    accessToken = access;
    refreshToken = refresh;
    writeStoredRefreshToken(refresh);
  },

  clear(): void {
    accessToken = null;
    refreshToken = null;
    writeStoredRefreshToken(null);
  },

  hasStoredSession(): boolean {
    return accessToken !== null || refreshToken !== null;
  },
};

session.init();
