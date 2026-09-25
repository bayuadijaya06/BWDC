import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AuthSession, UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import { session } from "@/services/session";

const mocks = vi.hoisted(() => ({
  login: vi.fn<(username: string, password: string) => Promise<unknown>>(),
  fetchProfile: vi.fn<() => Promise<unknown>>(),
  logout: vi.fn<(all?: boolean) => Promise<void>>(),
  refreshSession: vi.fn<() => Promise<unknown>>(),
}));

vi.mock("@/services/auth", () => ({
  login: mocks.login,
  fetchProfile: mocks.fetchProfile,
  logout: mocks.logout,
  refreshSession: mocks.refreshSession,
}));

const { useAuthStore } = await import("./auth");

const authSession: AuthSession = {
  token: "access-1",
  expires_at: "2026-09-21T10:15:00Z",
  user: {
    id: "u1",
    username: "admin",
    email: "admin@example.test",
    roles: ["administrator"],
  },
};

const profile: UserProfile = {
  ...authSession.user,
  organization_id: "org-1",
  is_active: true,
  permissions: ["project:read"],
};

beforeEach(() => {
  mocks.login.mockReset();
  mocks.fetchProfile.mockReset();
  mocks.logout.mockReset();
  mocks.refreshSession.mockReset();
  session.clear();
  useAuthStore.setState({
    status: "unknown",
    profile: null,
    error: null,
    pending: false,
  });
});

describe("store auth", () => {
  it("menyimpan access token dan membaca profil saat login berhasil", async () => {
    mocks.login.mockResolvedValue(authSession);
    mocks.fetchProfile.mockResolvedValue(profile);

    const ok = await useAuthStore.getState().signIn("admin", "rahasia");

    expect(ok).toBe(true);
    expect(mocks.login).toHaveBeenCalledWith("admin", "rahasia");
    expect(session.getAccessToken()).toBe("access-1");
    expect(useAuthStore.getState().status).toBe("authenticated");
    expect(useAuthStore.getState().profile?.permissions).toEqual([
      "project:read",
    ]);
  });

  it("kembali ke keadaan anonim dan tidak menyimpan token saat kredensial salah", async () => {
    mocks.login.mockRejectedValue(
      new ApiError({
        status: 401,
        code: "INVALID_CREDENTIALS",
        message: "kredensial tidak sah",
      }),
    );

    const ok = await useAuthStore.getState().signIn("admin", "salah");

    expect(ok).toBe(false);
    expect(session.getAccessToken()).toBeNull();
    expect(useAuthStore.getState().status).toBe("anonymous");
    expect(useAuthStore.getState().error?.code).toBe("INVALID_CREDENTIALS");
  });

  it("membersihkan sesi lokal walau permintaan logout gagal", async () => {
    session.setTokens("access-1");
    useAuthStore.setState({ status: "authenticated", profile });
    mocks.logout.mockRejectedValue(
      ApiError.network("tidak dapat menghubungi server"),
    );

    await useAuthStore.getState().signOut(true);

    expect(mocks.logout).toHaveBeenCalledWith(true);
    expect(session.getAccessToken()).toBeNull();
    expect(useAuthStore.getState().status).toBe("anonymous");
    expect(useAuthStore.getState().profile).toBeNull();
  });

  it("anonim tanpa memanggil server saat refresh cookie gagal", async () => {
    mocks.refreshSession.mockRejectedValue(
      new ApiError({ status: 401, code: "UNAUTHORIZED", message: "tanpa sesi" }),
    );

    await useAuthStore.getState().restore();

    expect(mocks.refreshSession).toHaveBeenCalledOnce();
    expect(mocks.fetchProfile).not.toHaveBeenCalled();
    expect(useAuthStore.getState().status).toBe("anonymous");
  });

  it("menukar cookie menjadi sesi saat halaman dimuat ulang", async () => {
    mocks.refreshSession.mockResolvedValue({ token: "access-baru" });
    mocks.fetchProfile.mockResolvedValue(profile);

    await useAuthStore.getState().restore();

    expect(mocks.fetchProfile).toHaveBeenCalledOnce();
    expect(session.getAccessToken()).toBe("access-baru");
    expect(useAuthStore.getState().status).toBe("authenticated");
  });

  it("memperlakukan refresh yang dicabut sebagai anonim, bukan galat", async () => {
    mocks.refreshSession.mockRejectedValue(
      new ApiError({
        status: 401,
        code: "TOKEN_REVOKED",
        message: "sesi dicabut",
      }),
    );

    await useAuthStore.getState().restore();

    expect(useAuthStore.getState().status).toBe("anonymous");
    expect(useAuthStore.getState().error).toBeNull();
    expect(session.getAccessToken()).toBeNull();
  });

  it("memeriksa izin dari daftar server, bukan dari nama role", () => {
    useAuthStore.setState({ status: "authenticated", profile });
    expect(useAuthStore.getState().has("project:read")).toBe(true);
    expect(useAuthStore.getState().has("administrator")).toBe(false);
    expect(useAuthStore.getState().hasAny(["user:read", "project:read"])).toBe(
      true,
    );
  });
});
