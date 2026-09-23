import { act, fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import type { UserProfile } from "@/services/auth";
import { useAuthStore } from "@/store/auth";
import { renderWithProviders } from "@/test/render";

import { AppShell } from "./AppShell";

/**
 * Laci menu di layar sempit (`51-UX.md` §8, temuan C-067).
 *
 * Yang diuji di sini hanya perilaku yang tidak dapat dilihat mata di satu
 * lebar layar: state laci, kunci gulir, fokus, dan apa yang terjadi ketika
 * jendela dilebarkan saat laci masih terbuka. Ukuran tap target diuji
 * terpisah di `tapTarget.test.tsx` (jsdom tidak mengukur CSS).
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

/**
 * Media query tiruan yang bisa diubah dari test.
 *
 * Menirukan perilaku aslinya: listener menerima **event** berisi `matches`
 * terbaru, bukan dipanggil tanpa argumen — kalau tidak, test akan lulus dari
 * stub yang salah alih-alih dari kode yang benar.
 */
function installMatchMedia(initial: { tablet?: boolean } = {}) {
  let tablet = initial.tablet ?? false;
  const listeners = new Set<{
    query: string;
    listener: (event: MediaQueryListEvent) => void;
  }>();

  // `prefers-color-scheme: dark` sengaja selalu `false` (tema terang) supaya
  // test tidak bergantung pada tema OS mesin yang menjalankannya.
  const matchesFor = (query: string) =>
    query.includes("min-width: 48rem") ? tablet : false;

  window.matchMedia = ((query: string) => ({
    media: query,
    get matches() {
      return matchesFor(query);
    },
    onchange: null,
    addEventListener: (
      _type: string,
      listener: (event: MediaQueryListEvent) => void,
    ) => {
      listeners.add({ query, listener });
    },
    removeEventListener: (
      _type: string,
      listener: (event: MediaQueryListEvent) => void,
    ) => {
      for (const entry of listeners) {
        if (entry.query === query && entry.listener === listener)
          listeners.delete(entry);
      }
    },
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia;

  return {
    setTablet(value: boolean) {
      tablet = value;
      for (const { query, listener } of [...listeners]) {
        listener({ matches: matchesFor(query), media: query } as MediaQueryListEvent);
      }
    },
  };
}

function renderShell() {
  useAuthStore.setState({ status: "authenticated", profile });
  return renderWithProviders(
    <AppShell>
      <p>Isi halaman</p>
    </AppShell>,
  );
}

beforeEach(() => {
  installMatchMedia();
  useAuthStore.setState({
    status: "authenticated",
    profile,
    error: null,
    pending: false,
  });
});

afterEach(() => {
  document.body.style.overflow = "";
});

describe("laci menu pada layar sempit", () => {
  it("menyatakan keadaan laci lewat aria-expanded dan tidak mengunci gulir saat tertutup", () => {
    renderShell();

    const menuButton = screen.getByRole("button", { name: "Menu" });
    expect(menuButton).toHaveAttribute("aria-expanded", "false");
    expect(document.body.style.overflow).not.toBe("hidden");
    // Menu di belakang laci tidak boleh menerima fokus atau dibaca pembaca layar.
    expect(screen.getByRole("main")).not.toHaveAttribute("inert");
  });

  it("membuka laci, mengunci gulir, dan memindahkan fokus ke item menu pertama", async () => {
    const user = userEvent.setup();
    renderShell();

    await user.click(screen.getByRole("button", { name: "Menu" }));

    const panel = screen.getByRole("dialog", { name: "Menu utama" });
    expect(panel).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Menu" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(document.body.style.overflow).toBe("hidden");
    expect(screen.getByRole("main")).toHaveAttribute("inert");
    expect(within(panel).getByRole("link", { name: "Dashboard" })).toHaveFocus();
  });

  it("menutup laci dengan Escape dan mengembalikan fokus ke tombol Menu", async () => {
    const user = userEvent.setup();
    renderShell();

    const menuButton = screen.getByRole("button", { name: "Menu" });
    await user.click(menuButton);
    expect(screen.getByRole("dialog", { name: "Menu utama" })).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(
      screen.queryByRole("dialog", { name: "Menu utama" }),
    ).not.toBeInTheDocument();
    expect(document.body.style.overflow).not.toBe("hidden");
    expect(screen.getByRole("button", { name: "Menu" })).toHaveFocus();
  });

  it("menutup laci dengan tombol Tutup dan dengan klik latar penutup", async () => {
    const user = userEvent.setup();
    renderShell();

    await user.click(screen.getByRole("button", { name: "Menu" }));
    await user.click(screen.getByRole("button", { name: "Tutup" }));
    expect(
      screen.queryByRole("dialog", { name: "Menu utama" }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Menu" }));
    expect(screen.getByRole("dialog", { name: "Menu utama" })).toBeInTheDocument();
    // Latar penutup: elemen kosong tanpa peran, tidak diumumkan ke pembaca
    // layar, dan memang hanya menerima klik. Dicari lewat penandanya, bukan
    // lewat posisinya, supaya urutan elemen boleh berubah.
    const backdrop = document.querySelector("[data-drawer-backdrop]") as HTMLElement;
    expect(backdrop).not.toBeNull();
    expect(backdrop).toHaveAttribute("aria-hidden", "true");
    fireEvent.click(backdrop);

    expect(
      screen.queryByRole("dialog", { name: "Menu utama" }),
    ).not.toBeInTheDocument();
  });

  it("menutup laci sendiri saat jendela dilebarkan, dan tidak membukanya kembali saat dipersempit", async () => {
    const media = installMatchMedia();
    const user = userEvent.setup();
    renderShell();

    await user.click(screen.getByRole("button", { name: "Menu" }));
    expect(screen.getByRole("dialog", { name: "Menu utama" })).toBeInTheDocument();

    act(() => media.setTablet(true));

    expect(
      screen.queryByRole("dialog", { name: "Menu utama" }),
    ).not.toBeInTheDocument();
    expect(document.body.style.overflow).not.toBe("hidden");
    expect(screen.getByRole("main")).not.toHaveAttribute("inert");
    // Sidebar tetap ada sebagai kolom, dan tombol Menu disembunyikan CSS.
    expect(
      screen.getByRole("navigation", { name: "Menu utama" }),
    ).toBeInTheDocument();

    // Jendela dipersempit lagi: menu yang tadi ditutup **tidak** muncul sendiri.
    // Kalau state-nya hanya disembunyikan (bukan dilupakan), di sini ia kembali
    // terbuka tanpa ada yang meminta — termasuk gulir yang terkunci lagi.
    act(() => media.setTablet(false));

    expect(screen.getByRole("button", { name: "Menu" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    expect(
      screen.queryByRole("dialog", { name: "Menu utama" }),
    ).not.toBeInTheDocument();
    expect(document.body.style.overflow).not.toBe("hidden");
  });
});

/**
 * Sidebar memuat **modul** saja (`51-UX.md` §2.1): tab dan penyaring modul
 * hidup di halamannya sendiri. Dulu setiap modul mendaftarkan sub-itemnya di
 * sini, sehingga satu penyaring punya dua tempat memilih dan menunya
 * memanjang tanpa menambah kemampuan.
 *
 * Dua test di bawah mengunci hal yang tidak dapat dilihat dari satu tangkapan
 * layar: tidak ada item menu yang menunjuk penyaring halaman **maupun** halaman
 * bersarang (`Reports > Audit` dulu berdiri di sini, padahal Audit adalah
 * halaman di bawah Reports), dan hanya **satu** menu menyala — termasuk ketika
 * yang dibuka adalah path bersarang di bawah path menu lain.
 */
describe("menu sidebar", () => {
  /**
   * Tautan menu dicari lewat **label**-nya, bukan lewat nama aksesibelnya.
   * Modul yang belum dibangun membawa penanda `belum` di dalam tautannya, jadi
   * nama aksesibelnya ikut memuat kata itu — pencocokan nama penuh akan lulus
   * atau gagal karena penanda tersebut, bukan karena labelnya.
   */
  function menuLinks() {
    const sidebar = screen.getByRole("navigation", { name: "Menu utama" });
    return within(sidebar).getAllByRole("link");
  }

  it("memuat satu tautan per modul, tanpa item untuk penyaring halaman", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: { ...profile, permissions: ["task:read"] },
      error: null,
      pending: false,
    });

    renderWithProviders(
      <AppShell>
        <p>Isi halaman</p>
      </AppShell>,
      { route: "/tasks?view=mine" },
    );

    // Satu tautan per modul: tidak ada label yang muncul dua kali.
    const labels = menuLinks().map(
      (link) => link.firstElementChild?.textContent?.trim() ?? "",
    );
    expect(labels).toEqual(["Dashboard", "Tasks"]);
    expect(new Set(labels).size).toBe(labels.length);

    // Penyaring modul tidak lagi punya item menu sendiri; tautan ke
    // `?view=mine` ada di halaman Tasks, bukan di sini.
    const sidebar = screen.getByRole("navigation", { name: "Menu utama" });
    for (const sub of [
      "Semua",
      "Buat",
      "Milik saya",
      "Tim",
      "Overdue",
      "Completed",
      "Pending Review",
      "Revision Required",
      "Approved",
    ]) {
      expect(within(sidebar).queryByRole("link", { name: sub })).toBeNull();
    }
  });

  it("menandai tepat satu menu, yang cocok dengan halaman yang sedang dibuka", () => {
    renderWithProviders(
      <AppShell>
        <p>Isi halaman</p>
      </AppShell>,
      { route: "/projects" },
    );

    const sidebar = screen.getByRole("navigation", { name: "Menu utama" });
    const current = within(sidebar)
      .getAllByRole("link")
      .filter((link) => link.getAttribute("aria-current") === "page")
      .map((link) => link.textContent?.trim());

    expect(current).toEqual(["Projects"]);
  });

  it("tidak punya menu untuk sub-halaman, dan menyalakan modul yang memuatnya", () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: {
        ...profile,
        permissions: ["project:read", "report:read", "audit:read"],
      },
      error: null,
      pending: false,
    });

    renderWithProviders(
      <AppShell>
        <p>Isi halaman</p>
      </AppShell>,
      { route: "/reports/audit" },
    );

    // Audit adalah halaman di bawah Reports, bukan menu (`51-UX.md` §2.1):
    // tidak ada tautan menu yang menunjuknya, dan yang menyala adalah modul
    // yang memuatnya — tepat satu, bukan dua.
    expect(
      menuLinks().some(
        (link) =>
          (link.getAttribute("href") || "").replace(/^\//, "") ===
          "reports/audit",
      ),
    ).toBe(false);

    const sidebar = screen.getByRole("navigation", { name: "Menu utama" });
    const current = within(sidebar)
      .getAllByRole("link")
      .filter((link) => link.getAttribute("aria-current") === "page");
    expect(current).toHaveLength(1);
    expect(current[0].firstElementChild?.textContent?.trim()).toBe("Reports");
  });
});
