import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, http, onSessionLost } from "./http";
import { session } from "./session";

/**
 * Uji ini memakai adapter axios buatan supaya perjalanan penuh interceptor
 * benar-benar dilewati: header Authorization, penukaran refresh token, dan
 * percobaan ulang. Yang penting di sini adalah perilaku ADR-0023 di sisi klien,
 * khususnya bahwa sepuluh permintaan yang gagal bersamaan hanya memicu **satu**
 * penukaran token.
 */

interface Reply {
  status: number;
  data: unknown;
  headers?: Record<string, string>;
}

interface Call {
  url: string;
  authorization: string;
}

let calls: Call[] = [];
let handler: (config: InternalAxiosRequestConfig) => Reply | Promise<Reply>;

function installAdapter(): void {
  const adapter = async (config: InternalAxiosRequestConfig) => {
    calls.push({
      url: String(config.url),
      authorization: String(config.headers?.Authorization ?? ""),
    });

    const reply = await handler(config);
    if (reply.status >= 400) {
      throw new AxiosError(
        "permintaan gagal",
        "ERR_BAD_REQUEST",
        config,
        null,
        {
          data: reply.data,
          status: reply.status,
          statusText: "",
          headers: reply.headers ?? {},
          config,
        },
      );
    }
    return {
      data: reply.data,
      status: reply.status,
      statusText: "OK",
      headers: reply.headers ?? {},
      config,
    };
  };

  http.defaults.adapter = adapter as never;
  axios.defaults.adapter = adapter as never;
}

function failure(
  status: number,
  code: string,
  details?: unknown,
  headers?: Record<string, string>,
) {
  return {
    status,
    data: { success: false, error: { code, message: code, details } },
    headers,
  };
}

function success(data: unknown): Reply {
  return { status: 200, data: { success: true, data } };
}

beforeEach(() => {
  calls = [];
  session.clear();
  installAdapter();
});

describe("interceptor http", () => {
  it("menyisipkan access token pada setiap permintaan", async () => {
    session.setTokens("access-1", "refresh-1");
    handler = () => success({ ok: true });

    await http.get("/auth/me");

    expect(calls).toHaveLength(1);
    expect(calls[0]?.authorization).toBe("Bearer access-1");
  });

  it("menukar refresh token lalu mengulang permintaan sekali saat 401 UNAUTHORIZED", async () => {
    session.setTokens("access-lama", "refresh-1");
    let protectedAttempts = 0;

    handler = (config) => {
      if (String(config.url).includes("/auth/refresh")) {
        return success({ token: "access-baru", refresh_token: "refresh-2" });
      }
      protectedAttempts += 1;
      if (protectedAttempts === 1) return failure(401, "UNAUTHORIZED");
      return success({ username: "admin" });
    };

    const response = await http.get("/auth/me");

    expect(response.data.data).toEqual({ username: "admin" });
    expect(protectedAttempts).toBe(2);
    expect(session.getAccessToken()).toBe("access-baru");
    expect(session.getRefreshToken()).toBe("refresh-2");
    // Penukaran refresh memakai instance axios polos, jadi ia sengaja **tidak**
    // membawa header Authorization; permintaan yang diulang membawa token baru.
    const refreshCall = calls.find((call) =>
      call.url.includes("/auth/refresh"),
    );
    expect(refreshCall?.authorization).toBe("");
    expect(
      calls
        .filter((call) => !call.url.includes("/auth/refresh"))
        .map((call) => call.authorization),
    ).toEqual(["Bearer access-lama", "Bearer access-baru"]);
  });

  it("tidak menukar token berkali-kali saat banyak permintaan gagal bersamaan", async () => {
    session.setTokens("access-lama", "refresh-1");
    const attempts = new Map<string, number>();
    let refreshes = 0;

    handler = (config) => {
      const url = String(config.url);
      if (url.includes("/auth/refresh")) {
        refreshes += 1;
        return success({ token: "access-baru", refresh_token: "refresh-2" });
      }
      const attempt = (attempts.get(url) ?? 0) + 1;
      attempts.set(url, attempt);
      return attempt === 1 ? failure(401, "UNAUTHORIZED") : success({ url });
    };

    const results = await Promise.all([
      http.get("/projects"),
      http.get("/documents"),
      http.get("/tasks"),
    ]);

    expect(results).toHaveLength(3);
    expect(refreshes).toBe(1);
    expect(session.getRefreshToken()).toBe("refresh-2");
  });

  it("tidak mencoba refresh saat token sesi sudah dicabut (TOKEN_REVOKED)", async () => {
    session.setTokens("access-1", "refresh-1");
    const listener = vi.fn();
    const stop = onSessionLost(listener);
    handler = () => failure(401, "TOKEN_REVOKED");

    await expect(http.get("/auth/me")).rejects.toBeInstanceOf(ApiError);

    expect(listener).toHaveBeenCalledOnce();
    expect(session.getAccessToken()).toBeNull();
    expect(session.getRefreshToken()).toBeNull();
    expect(
      calls.filter((call) => call.url.includes("/auth/refresh")),
    ).toHaveLength(0);
    stop();
  });

  it("meneruskan 401 apa adanya bila tidak ada refresh token tersimpan", async () => {
    session.clear();
    handler = () => failure(401, "UNAUTHORIZED");

    const error = await http.get("/auth/me").catch((caught: unknown) => caught);

    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(401);
    expect(
      calls.filter((call) => call.url.includes("/auth/refresh")),
    ).toHaveLength(0);
  });

  it("membaca lama tunggu akun terkunci dari details 423 (ADR-0022)", async () => {
    handler = () =>
      failure(
        423,
        "LOCKED",
        { locked_until: "2026-09-21T10:15:00Z", retry_after_seconds: 900 },
        { "retry-after": "900" },
      );

    const error = (await http
      .get("/auth/me")
      .catch((caught: unknown) => caught)) as ApiError;

    expect(error.code).toBe("LOCKED");
    expect(error.lockedForSeconds).toBe(900);
  });

  it("menerjemahkan server yang tidak menjawab sebagai galat jaringan", async () => {
    // Adapter yang melempar tanpa respons: inilah bentuk koneksi putus.
    http.defaults.adapter = (async () => {
      throw new AxiosError("koneksi ditolak", "ECONNREFUSED");
    }) as never;

    const error = (await http
      .get("/auth/me")
      .catch((caught: unknown) => caught)) as ApiError;

    expect(error.status).toBe(0);
    expect(error.code).toBe("NETWORK_ERROR");
  });
});
