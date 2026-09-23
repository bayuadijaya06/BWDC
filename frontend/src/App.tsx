import { useEffect, type ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router";

import { AppShell } from "@/components/layout/AppShell";
import { navigation, subPages, type NavItem } from "@/config/navigation";
import { DashboardPage } from "@/pages/Dashboard";
import { LoginPage } from "@/pages/Login";
import { ModulePendingPage } from "@/pages/ModulePending";
import { NotFoundPage } from "@/pages/NotFound";
import { DocumentDetailPage } from "@/pages/Documents/DocumentDetail";
import { DocumentsPage } from "@/pages/Documents";
import { ProjectDetailPage } from "@/pages/Projects/ProjectDetail";
import { ProjectsPage } from "@/pages/Projects";
import { TasksPage } from "@/pages/Tasks";
import { TaskDetailPage } from "@/pages/Tasks/TaskDetail";
import { ApprovalDetailPage } from "@/pages/Approvals/Detail";
import { ApprovalsPage } from "@/pages/Approvals";
import { useAuthStore } from "@/store/auth";
import { useThemeStore } from "@/store/theme";

/**
 * Penjaga rute.
 *
 * Sesi dipulihkan lewat `POST /auth/refresh` (ADR-0023), jadi sebelum hasilnya
 * diketahui halaman tidak boleh menebak: keadaan `unknown` menampilkan
 * penanda memuat, bukan form login yang berkedip. Rute yang dituju disimpan di
 * state navigasi supaya pengguna kembali ke sana sesudah masuk.
 */
function RequireAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((state) => state.status);
  const location = useLocation();

  if (status === "unknown") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-surface">
        <p role="status" className="text-13 text-text-muted">
          Memuat sesi…
        </p>
      </div>
    );
  }

  if (status === "anonymous") {
    return (
      <Navigate
        to="/login"
        replace
        state={{ from: location.pathname + location.search }}
      />
    );
  }

  return <>{children}</>;
}

/**
 * Rute yang izinnya wajib. Izin diperiksa di sini juga: menyembunyikan menu
 * tidak cukup, karena alamat dapat diketik langsung (`44-SECURITY.md` §3.1.1
 * menaruh penegakan di server, dan ini hanya menyelaraskan tampilan). Yang
 * dijawab di sini adalah halaman "tidak ditemukan", bukan halaman kosong:
 * halaman yang tampil lalu gagal memuat karena `403` lebih membingungkan.
 */
function GuardedRoute({
  item,
  children,
}: {
  item: NavItem;
  children?: ReactNode;
}) {
  const granted = useAuthStore((state) => state.profile?.permissions ?? []);
  const allowed =
    item.permissions.length === 0 ||
    item.permissions.some((p) => granted.includes(p));
  if (!allowed) return <NotFoundPage />;
  return children ? <>{children}</> : <ModulePendingPage item={item} />;
}

export function App() {
  const watchSystem = useThemeStore((state) => state.watchSystem);

  useEffect(() => watchSystem(), [watchSystem]);

  // Rute pending dibangun dari **kedua** daftar: modul sidebar dan halaman
  // anaknya. Halaman anak bukan menu (`51-UX.md` §2.1), tetapi tetap punya
  // rute — dan rutenya tetap hidup walau modul induknya belum dibangun, supaya
  // tidak ada tautan yang menunjuk halaman tidak ada (R-24).
  const pendingItems = [...navigation, ...subPages].filter(
    (item) => item.status === "pending",
  );
  const projects = navigation.find((item) => item.path === "/projects");
  const documents = navigation.find((item) => item.path === "/documents");
  const tasks = navigation.find((item) => item.path === "/tasks");
  const approvals = navigation.find((item) => item.path === "/approvals");

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />

      <Route
        element={
          <RequireAuth>
            <AppShell />
          </RequireAuth>
        }
      >
        <Route path="/" element={<DashboardPage />} />
        {projects ? (
          <Route path="/projects" element={<GuardedRoute item={projects}><ProjectsPage /></GuardedRoute>} />
        ) : null}
        {projects ? (
          <Route
            path="/projects/:id"
            element={
              <GuardedRoute item={projects}>
                <ProjectDetailPage />
              </GuardedRoute>
            }
          />
        ) : null}
        {documents ? (
          <Route
            path="/documents"
            element={
              <GuardedRoute item={documents}>
                <DocumentsPage />
              </GuardedRoute>
            }
          />
        ) : null}
        {documents ? (
          <Route
            path="/documents/:id"
            element={
              <GuardedRoute item={documents}>
                <DocumentDetailPage />
              </GuardedRoute>
            }
          />
        ) : null}
        {tasks ? (
          <Route
            path="/tasks"
            element={
              <GuardedRoute item={tasks}>
                <TasksPage />
              </GuardedRoute>
            }
          />
        ) : null}
        {tasks ? (
          <Route
            path="/tasks/:id"
            element={
              <GuardedRoute item={tasks}>
                <TaskDetailPage />
              </GuardedRoute>
            }
          />
        ) : null}
        {approvals ? (
          <Route
            path="/approvals"
            element={
              <GuardedRoute item={approvals}>
                <ApprovalsPage />
              </GuardedRoute>
            }
          />
        ) : null}
        {approvals ? (
          <Route
            path="/approvals/:id"
            element={
              <GuardedRoute item={approvals}>
                <ApprovalDetailPage />
              </GuardedRoute>
            }
          />
        ) : null}
        {pendingItems.map((item) => (
          <Route
            key={item.path}
            path={item.path}
            element={<GuardedRoute item={item} />}
          />
        ))}
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
