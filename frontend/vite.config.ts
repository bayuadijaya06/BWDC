import { fileURLToPath, URL } from "node:url";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { loadEnv } from "vite";
import { defineConfig } from "vitest/config";

// Konfigurasi dev, build, dan test dalam satu berkas.
//
// Proxy `/api` ada di sini supaya dev server tidak butuh konfigurasi CORS di
// backend (`12-DEVELOPMENT-WORKFLOW.md` §7.1). Targetnya diambil dari
// `VITE_API_PROXY_TARGET` karena port backend berbeda antar mesin: dokumen
// menyebut `8080`, mesin development ini memakai `8081` (`APP_HOST_PORT`).
export default defineConfig(({ mode }) => {
  // `.env*` lebih dulu, variabel lingkungan shell menimpanya: itu yang membuat
  // `VITE_API_PROXY_TARGET=http://localhost:8082 npm run dev` bekerja tanpa
  // menyunting berkas.
  const env: Record<string, string | undefined> = {
    ...loadEnv(mode, process.cwd(), "VITE_"),
    ...process.env,
  };
  const proxyTarget = env.VITE_API_PROXY_TARGET ?? "http://localhost:8081";
  const devPort = Number(env.VITE_DEV_PORT ?? 5173);

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
    optimizeDeps: {
      include: ["recharts"],
    },
    server: {
      port: devPort,
      strictPort: false,
      proxy: {
        "/api": { target: proxyTarget, changeOrigin: true },
      },
    },
    preview: {
      port: Number(env.VITE_PREVIEW_PORT ?? 4173),
    },
    test: {
      environment: "jsdom",
      setupFiles: ["./src/test/setup.ts"],
      include: ["src/**/*.test.{ts,tsx}"],
      // Batas per test dinaikkan bersama jendela `findBy*` (5 detik) di
      // `src/test/setup.ts`: tanpa ini, batas bawaan 5 detik vitest membuat test
      // yang menunggu penuh justru gagal karena **batas**, bukan karena isinya
      // (temuan **C-082**).
      testTimeout: 20000,
      coverage: {
        provider: "v8",
        reporter: ["text", "html"],
        include: ["src/**/*.{ts,tsx}"],
        exclude: ["src/**/*.test.{ts,tsx}", "src/test/**", "src/main.tsx"],
      },
    },
  };
});
