import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderResult } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter, Route, Routes } from "react-router";

/**
 * Pembungkus render untuk test.
 *
 * Query client dibuat **per test** dan dengan `retry: false`: kebijakan ulang
 * milik aplikasi (dua percobaan untuk kegagalan jaringan) akan membuat test
 * kesalahan menunggu dan dapat lulus karena percobaan kedua, padahal yang
 * sedang diuji adalah tampilan saat gagal.
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
      mutations: { retry: false },
    },
  });
}

/**
 * `path` dipakai halaman yang membaca `useParams`: tanpa rute yang benar-benar
 * cocok, `useParams()` mengembalikan objek kosong dan halaman akan merender
 * kerangka pemuatan selamanya — test yang tampak gagal padahal halamannya benar.
 */
export function renderWithProviders(
  ui: ReactElement,
  options: { route?: string; path?: string; client?: QueryClient } = {},
): RenderResult & { queryClient: QueryClient } {
  const queryClient = options.client ?? createTestQueryClient();
  const result = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[options.route ?? "/"]}>
        {options.path ? (
          <Routes>
            <Route path={options.path} element={ui} />
          </Routes>
        ) : (
          ui
        )}
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...result, queryClient };
}
