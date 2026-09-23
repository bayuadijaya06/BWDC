import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router";

import "@/styles/tokens.css";
import "@/styles/base.css";
import { App } from "@/App";
import { createQueryClient } from "@/queries/client";
import { useAuthStore } from "@/store/auth";

const container = document.getElementById("root");
if (!container) throw new Error("elemen #root tidak ditemukan di index.html");

// Pemulihan sesi dijalankan sekali, sebelum render pertama menyelesaikan
// keadaannya; karena itu layar "Memuat sesi…" ada dan bukan form login.
void useAuthStore.getState().restore();

// Satu query client untuk seluruh aplikasi: cache rekam dibagi antar halaman,
// sehingga membuka detail project sesudah membuatnya tidak memanggil ulang API.
const queryClient = createQueryClient();

createRoot(container).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
