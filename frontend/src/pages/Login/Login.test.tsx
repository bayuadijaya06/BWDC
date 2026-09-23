import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AuthSession, UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import { session } from "@/services/session";
import { useAuthStore } from "@/store/auth";

const mocks = vi.hoisted(() => ({
  login: vi.fn<(username: string, password: string) => Promise<unknown>>(),
  fetchProfile: vi.fn<() => Promise<unknown>>(),
  logout: vi.fn<() => Promise<void>>(),
}));

vi.mock("@/services/auth", () => ({
  login: mocks.login,
  fetchProfile: mocks.fetchProfile,
  logout: mocks.logout,
}));

const { LoginPage } = await import("./index");

const authSession: AuthSession = {
  token: "access-1",
  expires_at: "2026-09-21T10:15:00Z",
  refresh_token: "refresh-1",
  refresh_expires_at: "2026-09-28T10:00:00Z",
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

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={["/login"]}>
      <LoginPage />
    </MemoryRouter>,
  );
}

beforeEach(() => {
  mocks.login.mockReset();
  mocks.fetchProfile.mockReset();
  session.clear();
  useAuthStore.setState({
    status: "anonymous",
    profile: null,
    error: null,
    pending: false,
  });
});

describe("halaman login", () => {
  it("menolak isian kosong di klien tanpa menghubungi server", async () => {
    renderLogin();
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }));

    expect(screen.getByText("Username wajib diisi")).toBeInTheDocument();
    expect(screen.getByText("Password wajib diisi")).toBeInTheDocument();
    expect(mocks.login).not.toHaveBeenCalled();
  });

  it("menandai field yang tidak valid dengan aria-invalid", async () => {
    renderLogin();
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }));

    expect(screen.getByLabelText("Username")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(screen.getByLabelText("Username")).toHaveAccessibleDescription(
      "Username wajib diisi",
    );
  });

  it("masuk dengan username yang dirapikan dan menyimpan sesi", async () => {
    mocks.login.mockResolvedValue(authSession);
    mocks.fetchProfile.mockResolvedValue(profile);
    renderLogin();

    await userEvent.type(screen.getByLabelText("Username"), "  admin  ");
    await userEvent.type(screen.getByLabelText("Password"), "rahasia");
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }));

    await waitFor(() =>
      expect(useAuthStore.getState().status).toBe("authenticated"),
    );
    expect(mocks.login).toHaveBeenCalledWith("admin", "rahasia");
    expect(session.getRefreshToken()).toBe("refresh-1");
  });

  it("menampilkan pesan server apa adanya saat kredensial salah", async () => {
    mocks.login.mockRejectedValue(
      new ApiError({
        status: 401,
        code: "INVALID_CREDENTIALS",
        message: "username atau password salah",
      }),
    );
    renderLogin();

    await userEvent.type(screen.getByLabelText("Username"), "admin");
    await userEvent.type(screen.getByLabelText("Password"), "salah");
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("username atau password salah");
    expect(alert).toHaveTextContent("Kode: INVALID_CREDENTIALS");
    // Server tidak membedakan username salah dan password salah (ADR-0022),
    // jadi klien tidak boleh menempelkan kesalahan ke salah satu field.
    expect(screen.queryByText("Username wajib diisi")).not.toBeInTheDocument();
  });

  it("menyebut lama tunggu saat akun terkunci (423 LOCKED)", async () => {
    mocks.login.mockRejectedValue(
      new ApiError({
        status: 423,
        code: "LOCKED",
        message: "akun terkunci sementara",
        details: {
          locked_until: "2026-09-21T10:15:00Z",
          retry_after_seconds: 900,
        },
      }),
    );
    renderLogin();

    await userEvent.type(screen.getByLabelText("Username"), "admin");
    await userEvent.type(screen.getByLabelText("Password"), "salah");
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("akun terkunci sementara");
    expect(alert).toHaveTextContent("Coba lagi dalam 15 menit.");
  });

  it("menonaktifkan tombol selama permintaan berjalan", async () => {
    const pending: { release?: () => void } = {};
    mocks.login.mockImplementation(
      () =>
        new Promise((resolve) => {
          pending.release = () => resolve(authSession);
        }),
    );
    mocks.fetchProfile.mockResolvedValue(profile);
    renderLogin();

    await userEvent.type(screen.getByLabelText("Username"), "admin");
    await userEvent.type(screen.getByLabelText("Password"), "rahasia");
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }));

    const pendingButton = await screen.findByRole("button", {
      name: "Memproses",
    });
    expect(pendingButton).toBeDisabled();
    expect(pendingButton).toHaveAttribute("aria-busy", "true");

    pending.release?.();
    await waitFor(() =>
      expect(useAuthStore.getState().status).toBe("authenticated"),
    );
  });
});
