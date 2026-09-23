import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";

import type { UserProfile } from "@/services/auth";
import { useAuthStore } from "@/store/auth";
import { renderWithProviders } from "@/test/render";

import { App } from "../App";
import { LoginPage } from "../pages/Login";
import { runAxe } from "./a11y";

const profile: UserProfile = {
  id: "u1",
  username: "admin",
  email: "admin@example.test",
  roles: ["administrator"],
  organization_id: "org-1",
  is_active: true,
  permissions: ["project:read", "document:read", "task:read"],
};

beforeEach(() => {
  useAuthStore.setState({
    status: "anonymous",
    profile: null,
    error: null,
    pending: false,
  });
});

describe("aksesibilitas", () => {
  it("pemeriksanya memang menemukan pelanggaran bila ada", async () => {
    // Test pemeriksa: tanpa ini, `toEqual([])` di bawah bisa berarti "lulus"
    // atau bisa berarti pemeriksanya tidak pernah benar-benar bekerja.
    const { container } = render(
      <form>
        <input type="text" />
      </form>,
    );

    const violations = await runAxe(container);
    expect(violations.join(" ")).toContain("label");
  });

  it("form login lolos pemeriksaan axe", async () => {
    const { container } = render(
      <MemoryRouter initialEntries={["/login"]}>
        <LoginPage />
      </MemoryRouter>,
    );

    expect(await runAxe(container)).toEqual([]);
  });

  it("kerangka aplikasi dan dashboard lolos pemeriksaan axe", async () => {
    useAuthStore.setState({ status: "authenticated", profile });
    const { container } = renderWithProviders(<App />, { route: "/" });

    expect(await runAxe(container)).toEqual([]);
  });

  it("laci menu yang terbuka lolos pemeriksaan axe", async () => {
    const user = userEvent.setup();
    useAuthStore.setState({ status: "authenticated", profile });
    const { container } = renderWithProviders(<App />, { route: "/" });

    await user.click(screen.getByRole("button", { name: "Menu" }));
    expect(screen.getByRole("dialog", { name: "Menu utama" })).toBeInTheDocument();

    expect(await runAxe(container)).toEqual([]);
  });

  it("halaman modul pending lolos pemeriksaan axe", async () => {
    useAuthStore.setState({ status: "authenticated", profile });
    const { container } = renderWithProviders(<App />, {
      route: "/tasks?view=mine",
    });

    expect(await runAxe(container)).toEqual([]);
  });
});
