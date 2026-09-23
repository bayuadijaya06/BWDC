import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import type { UserProfile } from "@/services/auth";
import { useAuthStore } from "@/store/auth";
import { renderWithProviders } from "@/test/render";

import { App } from "./App";

function profileWith(permissions: string[]): UserProfile {
  return {
    id: "u1",
    username: "admin",
    email: "admin@example.test",
    roles: ["administrator"],
    organization_id: "org-1",
    is_active: true,
    permissions,
  };
}

function renderApp(path: string) {
  return renderWithProviders(<App />, { route: path });
}

beforeEach(() => {
  useAuthStore.setState({
    status: "unknown",
    profile: null,
    error: null,
    pending: false,
  });
});

describe("routing dan penjagaan sesi", () => {
  it("menunggu hasil pemulihan sesi sebelum menampilkan apa pun", () => {
    renderApp("/");
    expect(screen.getByRole("status")).toHaveTextContent("Memuat sesi…");
    expect(
      screen.queryByRole("button", { name: "Masuk" }),
    ).not.toBeInTheDocument();
  });

  it("mengalihkan halaman terlindungi ke form login saat anonim", () => {
    useAuthStore.setState({ status: "anonymous" });
    renderApp("/documents");
    expect(screen.getByRole("button", { name: "Masuk" })).toBeInTheDocument();
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
  });

  it("menampilkan halaman dashboard yang sudah dibangun sesudah masuk", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["project:read"]),
    });
    renderApp("/");

    expect(
      screen.getByRole("heading", { level: 1, name: "Dashboard" }),
    ).toBeInTheDocument();
    expect(screen.getByText("admin@example.test")).toBeInTheDocument();
    expect(screen.getByText("1 pasangan resource:action")).toBeInTheDocument();
    expect(document.title).toBe("BWDCS");
  });

  it("menautkan hanya modul yang izinnya dimiliki pada panel dashboard", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["project:read", "task:read"]),
    });
    renderApp("/");

    const panel = screen
      .getByRole("heading", { name: "Modul bisnis" })
      .closest("section");
    const links = within(panel as HTMLElement)
      .getAllByRole("link")
      .map((link) => link.textContent?.trim());

    expect(links).toEqual(["Projects", "Tasks"]);
    // Panel itu menyebut keadaan nyata: berapa modul yang halamannya sudah
    // dibangun dan berapa yang masih berupa halaman penjelasan.
    expect(
      screen.getByText(
        "2 modul siap dipakai, 0 masih menampilkan halaman penjelasan",
      ),
    ).toBeInTheDocument();
  });

  it("menyembunyikan menu yang izinnya tidak dimiliki, dan menampilkan yang dimiliki", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["project:read"]),
    });
    renderApp("/");

    const sidebar = screen.getByRole("navigation", { name: "Menu utama" });
    expect(
      within(sidebar).getByRole("link", { name: /Projects/ }),
    ).toBeInTheDocument();
    expect(
      within(sidebar).queryByRole("link", { name: /Documents/ }),
    ).not.toBeInTheDocument();
    expect(
      within(sidebar).queryByRole("link", { name: /Administration/ }),
    ).not.toBeInTheDocument();
  });

  it("menampilkan halaman modul pending dengan menyebut kontraknya, bukan data contoh", () => {
    // Modul pending berubah dari waktu ke waktu; yang diuji di sini adalah
    // **bentuk** halamannya, dan modul yang dipakai harus modul yang memang
    // belum dibangun saat test ini dibaca (Approvals selesai pada T-070, jadi
    // contohnya Reports).
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["report:read"]),
    });
    renderApp("/reports?status=running");

    expect(
      screen.getByText(/Modul ini belum dibangun di antarmuka/),
    ).toBeInTheDocument();
    expect(
      screen.getByText("docs/design/50-FSD.md §10.6, docs/design/42-API.md §10"),
    ).toBeInTheDocument();
    expect(screen.getByText("status=running")).toBeInTheDocument();
    expect(
      screen.getByText("Belum ada data yang ditampilkan"),
    ).toBeInTheDocument();
  });

  it("menjawab halaman tidak ditemukan untuk halaman yang izinnya tidak ada", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["task:read"]),
    });
    renderApp("/projects");

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Halaman tidak ditemukan",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("/projects")).toBeInTheDocument();
  });

  it("menjawab halaman tidak ditemukan untuk modul yang izinnya tidak ada", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["task:read"]),
    });
    renderApp("/documents");

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Halaman tidak ditemukan",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("/documents")).toBeInTheDocument();
  });

  it("menjawab halaman tidak ditemukan untuk rute asing", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith([]),
    });
    renderApp("/entah-dimana");

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Halaman tidak ditemukan",
      }),
    ).toBeInTheDocument();
  });
});
