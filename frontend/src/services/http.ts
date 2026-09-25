import axios, {
  AxiosError,
  type AxiosInstance,
  type InternalAxiosRequestConfig,
} from "axios";

import {
  isLockedDetails,
  isValidationDetails,
  type ApiErrorBody,
  type ApiErrorCode,
  type ApiFailure,
  type ApiSuccess,
} from "@/types/api";

import { session } from "./session";

/** Kesalahan API yang sudah dinormalkan, siap dipetakan ke UI. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode | string;
  readonly details: ApiErrorBody["details"];
  readonly retryAfterSeconds: number | null;
  /** Terisi untuk `422`: satu entri per field yang bermasalah. */
  readonly fieldErrors: Record<string, string>;

  constructor(init: {
    status: number;
    code: ApiErrorCode | string;
    message: string;
    details?: ApiErrorBody["details"];
    retryAfterSeconds?: number | null;
  }) {
    super(init.message);
    this.name = "ApiError";
    this.status = init.status;
    this.code = init.code;
    this.details = init.details ?? null;
    this.retryAfterSeconds = init.retryAfterSeconds ?? null;
    this.fieldErrors = {};
    if (isValidationDetails(this.details)) {
      for (const detail of this.details) {
        this.fieldErrors[detail.field] = detail.error;
      }
    }
  }

  /** Kesalahan jaringan atau server yang tidak menjawab amplop API. */
  static network(message: string): ApiError {
    return new ApiError({ status: 0, code: "NETWORK_ERROR", message });
  }

  get isSessionLost(): boolean {
    return this.code === "TOKEN_REVOKED";
  }

  get lockedForSeconds(): number | null {
    if (this.code !== "LOCKED") return null;
    if (isLockedDetails(this.details)) return this.details.retry_after_seconds;
    return this.retryAfterSeconds;
  }
}

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

export const http: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  // Cookie refresh token (ADR-0033) harus ikut pada setiap permintaan ke API —
  // termasuk refresh itu sendiri. Tanpa ini peramban menahan cookie dan sesi
  // tidak pernah bertahan melewati umur access token.
  withCredentials: true,
  timeout: 30_000,
});

type SessionLostListener = () => void;
const sessionLostListeners = new Set<SessionLostListener>();

/** Dipakai store auth untuk membersihkan keadaan saat sesi benar-benar mati. */
export function onSessionLost(listener: SessionLostListener): () => void {
  sessionLostListeners.add(listener);
  return () => sessionLostListeners.delete(listener);
}

function notifySessionLost(): void {
  for (const listener of sessionLostListeners) listener();
}

http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = session.getAccessToken();
  if (token !== null) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});

// Single-flight: beberapa permintaan yang gagal bersamaan hanya memicu SATU
// penukaran refresh token. Tanpa ini, sepuluh permintaan paralel berarti
// sepuluh penukaran, dan backend tidak mencabut token lama saat rotasi
// (ADR-0023), sehingga penukaran berulang tidak menambah keamanan.
let refreshInFlight: Promise<string> | null = null;

async function exchangeRefreshToken(): Promise<string> {
  // Cookie dilampirkan peramban; tidak ada token yang dibaca dari storage
  // (ADR-0033). Kegagalan di sini berarti tidak ada sesi (cookie hilang,
  // kedaluwarsa, atau dicabut) — pemanggil yang memutuskan anonim.
  const response = await axios.post<ApiSuccess<{ token: string }>>(
    `${BASE_URL}/auth/refresh`,
    {},
    { timeout: 15_000, withCredentials: true },
  );

  session.setTokens(response.data.data.token);
  return response.data.data.token;
}

function refreshAccessToken(): Promise<string> {
  refreshInFlight ??= exchangeRefreshToken().finally(() => {
    refreshInFlight = null;
  });
  return refreshInFlight;
}

interface RetryableConfig extends InternalAxiosRequestConfig {
  _retriedAfterRefresh?: boolean;
  _skipRefresh?: boolean;
}

function normalise(error: AxiosError): ApiError {
  const response = error.response;
  if (response === undefined) {
    return ApiError.network(
      error.code === "ECONNABORTED"
        ? "server tidak menjawab tepat waktu"
        : "tidak dapat menghubungi server",
    );
  }

  const body = response.data as ApiFailure | undefined;
  const retryAfterHeader = response.headers?.["retry-after"];
  const retryAfterSeconds =
    typeof retryAfterHeader === "string" || typeof retryAfterHeader === "number"
      ? Number(retryAfterHeader)
      : null;

  if (body && typeof body === "object" && "error" in body && body.error) {
    return new ApiError({
      status: response.status,
      code: body.error.code,
      message: body.error.message,
      details: body.error.details ?? null,
      retryAfterSeconds: Number.isFinite(retryAfterSeconds)
        ? retryAfterSeconds
        : null,
    });
  }

  return new ApiError({
    status: response.status,
    code: "INTERNAL_ERROR",
    message: "respons server tidak dikenali",
    retryAfterSeconds: Number.isFinite(retryAfterSeconds)
      ? retryAfterSeconds
      : null,
  });
}

/**
 * Membaca badan galat yang datang sebagai blob.
 *
 * Permintaan unduhan memakai `responseType: "blob"`, jadi amplop JSON dari
 * `404`/`500` ikut terunduh sebagai blob dan pesan server akan hilang tanpa
 * langkah ini. Isi yang bukan JSON dikembalikan apa adanya; keputusan berikutnya
 * tetap milik `normalise`.
 */
async function readBlobBody(data: Blob): Promise<unknown> {
  try {
    return JSON.parse(await data.text());
  } catch {
    return data;
  }
}

http.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    if (error.response?.data instanceof Blob) {
      error.response.data = await readBlobBody(error.response.data);
    }
    if (!error.response) throw normalise(error);

    const apiError = normalise(error);
    const config = error.config as RetryableConfig | undefined;
    const isRefreshCall = config?.url?.includes("/auth/refresh") ?? false;

    if (apiError.isSessionLost) {
      session.clear();
      notifySessionLost();
      throw apiError;
    }

    if (
      apiError.status === 401 &&
      apiError.code === "UNAUTHORIZED" &&
      config !== undefined &&
      !isRefreshCall &&
      config._retriedAfterRefresh !== true
    ) {
      config._retriedAfterRefresh = true;
      try {
        const token = await refreshAccessToken();
        config.headers.set("Authorization", `Bearer ${token}`);
        return await http.request(config);
      } catch {
        session.clear();
        notifySessionLost();
        throw apiError;
      }
    }

    throw apiError;
  },
);
