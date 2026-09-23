import { QueryClient } from "@tanstack/react-query";

import { ApiError } from "@/services/http";

/**
 * Kebijakan ulang-permintaan.
 *
 * Aturannya satu arah: jawaban final tidak diulang. `4xx` (`422`, `403`, `409`,
 * `404`) adalah keputusan server atas permintaan itu, dan mengulangnya dengan
 * body yang sama hanya menambah beban sambil menampilkan penundaan palsu.
 * Yang diulang hanyalah kegagalan sementara: jaringan (`status 0`) dan `5xx`.
 */
export function shouldRetryRequest(
  failureCount: number,
  error: unknown,
): boolean {
  if (error instanceof ApiError) {
    if (error.status >= 400 && error.status < 500) return false;
  }
  // Dua percobaan tambahan: cukup untuk jaringan yang berkedip, tidak cukup
  // untuk menyembunyikan server yang sedang mati.
  return failureCount < 2;
}

/** Query client aplikasi. Dibuat sekali di `main.tsx`. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: shouldRetryRequest,
        // Rekam jarang berubah dalam hitungan detik; 30 detik menghindari
        // permintaan ulang saat pengguna berpindah tab.
        staleTime: 30_000,
      },
      mutations: {
        // Mutasi tidak pernah diulang otomatis: `POST /projects` yang diulang
        // dapat membuat dua project bila jawaban pertama hilang di jaringan.
        retry: false,
      },
    },
  });
}
