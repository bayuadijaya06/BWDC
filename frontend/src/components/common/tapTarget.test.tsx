import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import type { UserProfile } from "@/services/auth";
import { useAuthStore } from "@/store/auth";
import { renderWithProviders } from "@/test/render";

import { App } from "../../App";

/**
 * Target sentuh (`51-UX.md` §9: 44x44px; antislop R-03).
 *
 * jsdom tidak menghitung tata letak, jadi tingginya tidak dapat diukur di sini.
 * Yang **dapat** diperiksa mesin adalah bahwa setiap kontrol memakai utility
 * yang sama — `tap-target` — dan utility itulah yang sudah didefinisikan di
 * `tokens.css` beserta test kontras/ukuran pemakaiannya. Menambah kontrol baru
 * tanpa kelas itu akan gagal di sini, dan itu memang tujuannya: ukuran target
 * adalah aturan yang paling mudah terlupakan saat menambah tombol.
 *
 * Bukti bahwa 44px benar-benar berlaku diukur di dev server (log P-043 §6),
 * bukan diklaim dari kelas.
 */

const profile: UserProfile = {
  id: "u1",
  username: "admin",
  email: "admin@example.test",
  roles: ["administrator"],
  organization_id: "org-1",
  is_active: true,
  permissions: ["project:read"],
};

beforeEach(() => {
  useAuthStore.setState({
    status: "authenticated",
    profile,
    error: null,
    pending: false,
  });
});

describe("target sentuh", () => {
  it("setiap kontrol di header dan sidebar memakai tap-target", () => {
    renderWithProviders(<App />, { route: "/" });

    const controls = [
      screen.getByRole("button", { name: "Menu" }),
      screen.getByRole("button", { name: /^Tema:/ }),
      screen.getByRole("button", { name: /admin/ }),
    ];

    const sidebar = screen.getByRole("navigation", { name: "Menu utama" });
    controls.push(...within(sidebar).getAllByRole("link"));

    for (const control of controls) {
      expect(control.className).toContain("tap-target");
    }
  });

  it("form login memakai tap-target pada input dan tombolnya", () => {
    useAuthStore.setState({ status: "anonymous", profile: null });
    renderWithProviders(<App />, { route: "/login" });

    expect(screen.getByLabelText("Username").className).toContain("tap-target");
    expect(screen.getByLabelText("Password").className).toContain("tap-target");
    expect(screen.getByRole("button", { name: "Masuk" }).className).toContain(
      "tap-target",
    );
  });
});
