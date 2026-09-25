import { create } from "zustand";

import { ApiError, onSessionLost } from "@/services/http";
import { session } from "@/services/session";
import { fetchProfile, login, logout, refreshSession, type UserProfile } from "@/services/auth";

export type AuthStatus = "unknown" | "anonymous" | "authenticated";

interface AuthState {
  status: AuthStatus;
  profile: UserProfile | null;
  error: ApiError | null;
  pending: boolean;

  /** Dipanggil sekali saat aplikasi start. */
  restore: () => Promise<void>;
  signIn: (username: string, password: string) => Promise<boolean>;
  signOut: (all?: boolean) => Promise<void>;
  clearError: () => void;
  has: (permission: string) => boolean;
  hasAny: (permissions: string[]) => boolean;
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error;
  return ApiError.network("terjadi kesalahan yang tidak terduga");
}

export const useAuthStore = create<AuthState>((set, get) => ({
  status: "unknown",
  profile: null,
  error: null,
  pending: false,

  restore: async () => {
    if (!session.hasStoredSession()) {
      // Tanpa access token di memori, satu-satunya harapan adalah cookie
      // refresh (ADR-0033). Coba tukar dulu tanpa memanggil `GET /auth/me`
      // yang pasti 401 dan memenuhi console dengan error XHR — perilaku yang
      // sama dengan sesi sebelumnya, hanya sumber tokennya yang berpindah dari
      // `sessionStorage` ke cookie. Gagal berarti anonim, bukan error halaman.
      try {
        const refreshed = await refreshSession();
        session.setTokens(refreshed.token);
      } catch {
        session.clear();
        set({ status: "anonymous", profile: null, error: null });
        return;
      }
    }
    try {
      const profile = await fetchProfile();
      set({ status: "authenticated", profile, error: null });
    } catch (error) {
      // Sesi tersimpan yang tidak dapat dipakai berarti anonim, bukan error
      // halaman: pengguna cukup login lagi.
      const apiError = toApiError(error);
      if (apiError.isSessionLost || apiError.status === 401) {
        session.clear();
        set({ status: "anonymous", profile: null, error: null });
        return;
      }
      set({ status: "anonymous", profile: null, error: apiError });
    }
  },

  signIn: async (username, password) => {
    set({ pending: true, error: null });
    try {
      const authSession = await login(username, password);
      session.setTokens(authSession.token);
      const profile = await fetchProfile();
      set({ status: "authenticated", profile, pending: false, error: null });
      return true;
    } catch (error) {
      session.clear();
      set({
        status: "anonymous",
        profile: null,
        pending: false,
        error: toApiError(error),
      });
      return false;
    }
  },

  signOut: async (all = false) => {
    set({ pending: true });
    try {
      await logout(all);
    } catch {
      // Sesi lokal dibersihkan walau permintaan logout gagal: pengguna yang
      // menekan "keluar" tidak boleh tetap masuk karena jaringan bermasalah.
    } finally {
      session.clear();
      set({ status: "anonymous", profile: null, pending: false, error: null });
    }
  },

  clearError: () => set({ error: null }),

  has: (permission) => get().profile?.permissions.includes(permission) ?? false,

  hasAny: (permissions) => {
    const granted = get().profile?.permissions ?? [];
    return permissions.some((permission) => granted.includes(permission));
  },
}));

// Sesi yang dicabut dari sisi server (logout di perangkat lain, ganti password,
// atau logout semua) harus memindahkan UI ke keadaan anonim, bukan membiarkan
// layar yang sudah basi.
onSessionLost(() => {
  useAuthStore.setState({ status: "anonymous", profile: null, error: null });
});
