import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import type { UserProfile } from "@/services/auth";
import { useAuthStore } from "@/store/auth";
import { renderWithProviders } from "@/test/render";
import { navigation, subPages } from "@/config/navigation";

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

  it("tidak lagi punya halaman pending: seluruh menu dan anaknya ready", () => {
    // Bentuk halaman pending diuji saat masih ada contohnya (Approvals lalu
    // Reports lalu Administration lalu Audit); sejak T-106 tidak ada lagi
    // modul berstatus pending, dan test ini mengunci tonggak itu — bukan
    // bentuk yang sudah tidak punya contoh.
    const pending = [...navigation, ...subPages].filter(
      (item) => item.status === "pending",
    );
    expect(pending).toEqual([]);
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
