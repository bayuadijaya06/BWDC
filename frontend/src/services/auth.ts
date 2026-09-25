import type { ApiSuccess } from "@/types/api";

import { http } from "./http";

/** Ringkasan user pada respons login (`42-API.md` §2). */
export interface UserSummary {
  id: string;
  username: string;
  email: string;
  roles: string[];
}

export interface AuthSession {
  token: string;
  expires_at: string;
  user: UserSummary;
}

/**
 * Profil dari `GET /auth/me`. `permissions` adalah pasangan `resource:action`
 * hasil matriks `44-SECURITY.md` §3.1, dan klien memakainya untuk menyembunyikan
 * menu, bukan untuk menebak isi matriks.
 */
export interface UserProfile extends UserSummary {
  organization_id: string;
  is_active: boolean;
  permissions: string[];
}

export async function login(
  username: string,
  password: string,
): Promise<AuthSession> {
  const response = await http.post<ApiSuccess<AuthSession>>("/auth/login", {
    username,
    password,
  });
  return response.data.data;
}

/** Hasil `POST /auth/refresh`: hanya access token di body (ADR-0033). */
export interface RefreshResult {
  token: string;
  expires_at: string;
}

/**
 * Menukar cookie refresh menjadi access token baru. Cookie dilampirkan
 * peramban (`withCredentials`); tidak ada token yang dibaca dari storage.
 */
export async function refreshSession(): Promise<RefreshResult> {
  const response = await http.post<ApiSuccess<RefreshResult>>(
    "/auth/refresh",
    {},
  );
  return response.data.data;
}

export async function fetchProfile(): Promise<UserProfile> {
  const response = await http.get<ApiSuccess<UserProfile>>("/auth/me");
  return response.data.data;
}

export async function logout(all = false): Promise<void> {
  await http.post("/auth/logout", all ? { logout_all: true } : {});
}
