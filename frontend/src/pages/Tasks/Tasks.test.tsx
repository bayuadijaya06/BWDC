import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { Project, ProjectDetail } from "@/services/projects";
import type { TaskRecord } from "@/services/tasks";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";
import { formatTimestamp } from "@/utils/format";

const mocks = vi.hoisted(() => ({
  listTasks: vi.fn(),
  listProjects: vi.fn(),
  fetchProject: vi.fn(),
}));

vi.mock("@/services/tasks", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/tasks")>();
  return { ...actual, listTasks: mocks.listTasks };
});

vi.mock("@/services/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/projects")>();
  return {
    ...actual,
    fetchProject: mocks.fetchProject,
    listProjects: mocks.listProjects,
  };
});

const { TasksPage } = await import("./index");

const project: Project = {
  id: "p1",
  code: "WEB",
  name: "Website Redesign",
  description: "",
  owner_id: "u1",
  owner_username: "admin",
  status: "active",
  start_date: null,
  target_end_date: null,
  member_count: 2,
  created_at: "2026-09-19T00:40:00+07:00",
  updated_at: "2026-09-19T00:40:00+07:00",
};

const detail: ProjectDetail = {
  project,
  members: [
    {
      user_id: "u1",
      username: "admin",
      email: "admin@example.test",
      role: "manager",
      joined_at: "2026-09-19T00:40:00+07:00",
    },
    {
      user_id: "u2",
      username: "uji-contrib",
      email: "contrib@example.test",
      role: "contributor",
      joined_at: "2026-09-19T00:41:00+07:00",
    },
  ],
};

const overdueTask: TaskRecord = {
  id: "t1",
  project_id: "p1",
  project_code: "WEB",
  project_name: "Website Redesign",
  project_archived: false,
  title: "Tinjau BRD",
  description: "periksa sebelum submit",
  status: "open",
  priority: "high",
  due_date: "2026-09-01T10:00:00+07:00",
  is_overdue: true,
  assignee_id: "u2",
  assignee_username: "uji-contrib",
  document_id: "d1",
  document_number: "WEB-001",
  created_by_id: "u1",
  created_by_username: "admin",
  created_at: "2026-09-19T14:00:00+07:00",
  updated_at: "2026-09-19T14:00:00+07:00",
};

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

function meta(total: number, page = 1, totalPage = 1) {
  return { page, limit: 20, total, total_page: totalPage };
}

/** Argumen terakhir yang diterima `listTasks`, supaya penyaringnya dapat diperiksa. */
function lastQuery() {
  return mocks.listTasks.mock.calls.at(-1)?.[0] as Record<string, unknown>;
}

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.listTasks.mockResolvedValue({ items: [overdueTask], meta: meta(1) });
  mocks.listProjects.mockResolvedValue({ items: [project], meta: meta(1) });
  mocks.fetchProject.mockResolvedValue(detail);
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith([
      "task:read",
      "task:create",
      "task:update",
      "task:complete",
      "project:read",
      "document:read",
    ]),
    error: null,
    pending: false,
  });
});

describe("halaman Tasks — daftar", () => {
  it("menampilkan kolom 50-FSD.md §6.1 dengan label status kanonik", async () => {
    renderWithProviders(<TasksPage />, { route: "/tasks" });

    const link = await screen.findByRole("link", { name: "Tinjau BRD" });
    expect(link).toHaveAttribute("href", "/tasks/t1");

    const row = link.closest("tr") as HTMLElement;
    expect(within(row).getByText("Open")).toBeInTheDocument();
    expect(within(row).queryByText("open")).toBeNull();
    expect(within(row).getByText("High")).toBeInTheDocument();
    expect(
      within(row).getByText(formatTimestamp(overdueTask.due_date)),
    ).toBeInTheDocument();
    expect(within(row).getByText("uji-contrib")).toBeInTheDocument();
    const projectLink = within(row).getByRole("link", { name: "WEB" });
    expect(projectLink).toHaveAttribute("href", "/projects/p1");
    expect(projectLink).toHaveAttribute("title", "Website Redesign");

    // Penanda overdue adalah turunan yang dihitung server, bukan status kedua.
    expect(within(row).getByText("Overdue")).toBeInTheDocument();
    expect(within(row).queryByText("overdue")).toBeNull();
  });

  it("tidak menandai overdue bila server tidak menghitungnya", async () => {
    mocks.listTasks.mockResolvedValue({
      items: [{ ...overdueTask, status: "completed", is_overdue: false }],
      meta: meta(1),
    });

    renderWithProviders(<TasksPage />, { route: "/tasks" });

    const row = (await screen.findByRole("link", { name: "Tinjau BRD" })).closest(
      "tr",
    ) as HTMLElement;
    expect(within(row).getByText("Completed")).toBeInTheDocument();
    expect(within(row).queryByText("Overdue")).toBeNull();
  });

  it("menyebut jumlah dari meta dan menjelaskan cakupannya", async () => {
    mocks.listTasks.mockResolvedValue({
      items: [overdueTask],
      meta: meta(42, 3, 3),
    });

    renderWithProviders(<TasksPage />, { route: "/tasks?page=3" });

    expect(await screen.findByText(/42 task dalam cakupan Anda/)).toBeInTheDocument();
    expect(
      screen.getByText(/Menampilkan 41 sampai 41 dari 42 rekam/),
    ).toBeInTheDocument();
    expect(mocks.listTasks).toHaveBeenCalledWith(
      expect.objectContaining({ page: 3, limit: 20 }),
    );
  });

  it("memetakan sub-halaman ke penyaring kontrak, bukan ke halaman terpisah", async () => {
    renderWithProviders(<TasksPage />, { route: "/tasks?view=mine" });

    await waitFor(() => expect(mocks.listTasks).toHaveBeenCalled());
    // My Tasks = ?assignee_id=<diri sendiri> (50-FSD.md §6.1).
    expect(lastQuery()).toMatchObject({ assignee_id: "u1" });

    expect(
      screen.getByRole("link", { name: "Milik saya" }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("Team Tasks berarti tanpa penyaring penanggung jawab, sesuai 42-API.md §6", async () => {
    renderWithProviders(<TasksPage />, { route: "/tasks?view=team" });

    await waitFor(() => expect(mocks.listTasks).toHaveBeenCalled());

    // Kosong berarti tanpa penyaring penanggung jawab; lapisan service membuang
    // nilai kosong sebelum dikirim (dibuktikan `services/tasks.test.ts`).
    expect(lastQuery()).toMatchObject({ assignee_id: "" });
    // `view` adalah penanda sub-halaman, bukan parameter API yang dikarang.
    expect(lastQuery()).not.toHaveProperty("view");
  });

  it("Completed memakai status kanonik, bukan label tampilan", async () => {
    renderWithProviders(<TasksPage />, { route: "/tasks?status=completed" });

    await waitFor(() => expect(mocks.listTasks).toHaveBeenCalled());

    expect(lastQuery()).toMatchObject({ status: "completed" });
    expect(
      screen.getByRole("link", { name: "Completed" }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("mengirim penyaring status, prioritas, dan project ke server", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    await user.selectOptions(screen.getByLabelText("Status"), "in_progress");
    await waitFor(() =>
      expect(lastQuery()).toMatchObject({ status: "in_progress" }),
    );

    await user.selectOptions(screen.getByLabelText("Prioritas"), "urgent");
    await waitFor(() => expect(lastQuery()).toMatchObject({ priority: "urgent" }));

    await user.selectOptions(
      screen.getByLabelText("Project"),
      await within(screen.getByLabelText("Project")).findByRole("option", {
        name: /Website Redesign/,
      }),
    );
    await waitFor(() => expect(lastQuery()).toMatchObject({ project_id: "p1" }));
  });

  it("mengisi pilihan penanggung jawab dari anggota project yang dipilih", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    // Sebelum project dipilih, tidak ada anggota yang dapat ditawarkan, dan
    // halaman menyatakan sumbernya alih-alih menampilkan pemilih kosong.
    expect(
      screen.getByText(/belum ada project yang dipilih/),
    ).toBeInTheDocument();
    expect(
      within(screen.getByLabelText("Penanggung jawab")).queryByRole("option", {
        name: "uji-contrib",
      }),
    ).toBeNull();

    await user.selectOptions(
      screen.getByLabelText("Project"),
      await within(screen.getByLabelText("Project")).findByRole("option", {
        name: /Website Redesign/,
      }),
    );

    const assignee = screen.getByLabelText("Penanggung jawab");
    await within(assignee).findByRole("option", { name: "uji-contrib" });
    // Diri sendiri tetap ditawarkan terpisah sebagai "Milik saya".
    expect(within(assignee).getByRole("option", { name: "Milik saya" })).toBeInTheDocument();

    await user.selectOptions(assignee, "u2");
    await waitFor(() => expect(lastQuery()).toMatchObject({ assignee_id: "u2" }));
  });

  it("membedakan tiga keadaan penyaring overdue, bukan dua", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    // Tidak dikirim: bukan `false`, karena keduanya berbeda arti. Yang menahan
    // nilainya agar tidak sampai ke kabel adalah lapisan service.
    expect(lastQuery()).toMatchObject({ overdue: "" });

    await user.selectOptions(screen.getByLabelText("Overdue"), "false");
    await waitFor(() => expect(lastQuery()).toMatchObject({ overdue: "false" }));

    await user.selectOptions(screen.getByLabelText("Overdue"), "true");
    await waitFor(() => expect(lastQuery()).toMatchObject({ overdue: "true" }));

    await user.selectOptions(screen.getByLabelText("Overdue"), "");
    await waitFor(() => expect(lastQuery()).toMatchObject({ overdue: "" }));
  });

  it("meneruskan rentang tenggat sebagai instan ber-offset", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    fireEvent.change(screen.getByLabelText("Tenggat dari"), {
      target: { value: "2026-03-01T08:00" },
    });
    fireEvent.change(screen.getByLabelText("Tenggat sampai"), {
      target: { value: "2026-03-31T17:00" },
    });
    await user.click(screen.getByRole("button", { name: "Terapkan rentang" }));

    await waitFor(() =>
      expect(lastQuery()).toMatchObject({
        due_from: expect.stringMatching(/^2026-03-01T\d{2}:00:00\.000Z$/),
        due_to: expect.stringMatching(/^2026-03-31T\d{2}:00:00\.000Z$/),
      }),
    );
  });

  it("menempatkan kedua batas rentang di satu kolom yang tidak melipat", async () => {
    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    // Dua kolom terpisah dapat jatuh ke garis yang berbeda saat baris penyaring
    // melipat, sehingga "dari" dan "sampai" terbaca sebagai dua penyaring yang
    // tidak berhubungan (kekurangan yang dinyatakan P-047). Yang dikunci di sini
    // adalah **strukturnya**: keduanya berada di satu kelompok berlabel.
    const group = screen.getByRole("group", { name: "Rentang tenggat" });
    expect(within(group).getByLabelText("Tenggat dari")).toBeInTheDocument();
    expect(within(group).getByLabelText("Tenggat sampai")).toBeInTheDocument();

    // Kedua isian berada di baris yang sama dengan pemisahnya, dan baris itu
    // tidak boleh membungkus: tanpa `flex-wrap` kedua batas tidak dapat
    // dipisahkan ke garis yang berbeda.
    const row = within(group).getByLabelText("Tenggat dari").parentElement;
    expect(row).not.toBeNull();
    expect(row).toHaveClass("flex");
    expect(row).not.toHaveClass("flex-wrap");
    expect(row?.children).toHaveLength(3);
  });

  it("membuat setiap kolom penyaring ber-select dapat menyusut", async () => {
    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    // Ukuran minimum otomatis kolom flex adalah **min-content** anaknya, dan
    // min-content sebuah <select> ditentukan teks pilihan terpanjang — yaitu
    // data pengguna (nama project, nama anggota). Satu nama yang panjang karena
    // itu cukup untuk melebarkan baris penyaring melewati layar ponsel; terukur
    // `#penyaring-project-dokumen` 403px pada 375px, halaman menggulir mendatar
    // 44px. `min-w-0` yang menutupnya, dan ia wajib ada di **setiap** kolom
    // ber-select supaya tidak bergantung pada data yang kebetulan ada
    // (`scripts/responsive-evidence.mjs`, probe pilihan panjang).
    const form = screen.getByRole("search", { name: "Penyaring task" });
    const columns = [...form.querySelectorAll("select")].map((select) =>
      select.closest("div"),
    );
    expect(columns.length).toBeGreaterThan(0);
    for (const column of columns) {
      expect(column).not.toBeNull();
      expect(column).toHaveClass("min-w-0");
    }
  });

  it("menahan rentang terbalik di klien dan menyebut batas yang salah", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });
    const callsBefore = mocks.listTasks.mock.calls.length;

    fireEvent.change(screen.getByLabelText("Tenggat dari"), {
      target: { value: "2026-03-31T08:00" },
    });
    fireEvent.change(screen.getByLabelText("Tenggat sampai"), {
      target: { value: "2026-03-01T08:00" },
    });
    await user.click(screen.getByRole("button", { name: "Terapkan rentang" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Batas akhir tidak boleh mendahului batas awal.",
    );
    // Rentangnya tidak pernah dikirim: server akan menjawab 422, dan
    // mengirimkannya berarti meminta jawaban yang sudah diketahui.
    expect(mocks.listTasks.mock.calls.length).toBe(callsBefore);
  });

  it("membedakan keadaan kosong karena penyaring dan karena memang belum ada", async () => {
    mocks.listTasks.mockResolvedValue({ items: [], meta: meta(0) });

    const { unmount } = renderWithProviders(<TasksPage />, {
      route: "/tasks?status=completed",
    });
    expect(await screen.findByText("Tidak ada task yang cocok")).toBeInTheDocument();
    unmount();

    renderWithProviders(<TasksPage />, { route: "/tasks" });
    expect(await screen.findByText("Belum ada task")).toBeInTheDocument();
    expect(
      screen.getByText(/Statusnya selalu lahir Open/),
    ).toBeInTheDocument();
  });

  it("menampilkan kesalahan server apa adanya dengan tombol muat ulang", async () => {
    mocks.listTasks.mockRejectedValue(
      new ApiError({
        status: 500,
        code: "INTERNAL_ERROR",
        message: "gagal membaca task",
      }),
    );

    renderWithProviders(<TasksPage />, { route: "/tasks" });

    expect(await screen.findByRole("alert")).toHaveTextContent("gagal membaca task");
    expect(screen.getByRole("button", { name: "Muat ulang" })).toBeInTheDocument();
  });

  it("menyembunyikan tombol buat task bila izinnya tidak ada", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["task:read"]),
      error: null,
      pending: false,
    });

    renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    expect(screen.queryByRole("button", { name: "Buat task" })).toBeNull();
  });

  it("lolos pemeriksaan axe", async () => {
    const { container } = renderWithProviders(<TasksPage />, { route: "/tasks" });
    await screen.findByRole("link", { name: "Tinjau BRD" });

    expect(await runAxe(container)).toEqual([]);
  });
});
