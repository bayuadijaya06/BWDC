import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const {
  completeTask,
  createTask,
  listTasks,
  toLocalInputValue,
  toRfc3339FromLocal,
  updateTask,
  validateDueRange,
  validateTaskForm,
} = await import("./tasks");

function meta(total = 0) {
  return { page: 1, limit: 20, total, total_page: total === 0 ? 0 : 1 };
}

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.patch.mockReset();
});

describe("listTasks", () => {
  it("selalu mengirim halaman dan batas", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listTasks({ page: 2, limit: 50 });

    expect(mocks.get).toHaveBeenCalledWith("/tasks", {
      params: { page: 2, limit: 50 },
    });
  });

  it("tidak mengirim parameter overdue yang kosong, dan mengirim true/false apa adanya", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listTasks({ overdue: "" });
    expect(mocks.get).toHaveBeenLastCalledWith("/tasks", {
      params: { page: 1, limit: 20 },
    });

    await listTasks({ overdue: "true" });
    expect(mocks.get).toHaveBeenLastCalledWith("/tasks", {
      params: { page: 1, limit: 20, overdue: "true" },
    });

    // `false` **bukan** "tidak dikirim": kontrak §6 membedakan keduanya, dan
    // keduanya berarti hal yang berbeda bagi tugas tanpa tenggat.
    await listTasks({ overdue: "false" });
    expect(mocks.get).toHaveBeenLastCalledWith("/tasks", {
      params: { page: 1, limit: 20, overdue: "false" },
    });
  });

  it("meneruskan seluruh penyaring yang dijanjikan 50-FSD.md §6.1", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listTasks({
      status: "in_progress",
      priority: "high",
      project_id: "p1",
      assignee_id: "u2",
      due_from: "2026-03-01T00:00:00.000Z",
      due_to: "2026-03-31T23:59:59.000Z",
    });

    expect(mocks.get).toHaveBeenCalledWith("/tasks", {
      params: {
        page: 1,
        limit: 20,
        status: "in_progress",
        priority: "high",
        project_id: "p1",
        assignee_id: "u2",
        due_from: "2026-03-01T00:00:00.000Z",
        due_to: "2026-03-31T23:59:59.000Z",
      },
    });
  });

  it("mengembalikan daftar yang selalu array beserta meta", async () => {
    mocks.get.mockResolvedValue({
      data: { success: true, data: [], meta: meta(7) },
    });

    const result = await listTasks({});

    expect(result.items).toEqual([]);
    expect(result.meta.total).toBe(7);
  });
});

describe("createTask", () => {
  it("tidak mengirim status: task selalu lahir open (FR-TASK-03)", async () => {
    mocks.post.mockResolvedValue({ data: { success: true, data: { id: "t1" } } });

    await createTask({
      project_id: "p1",
      title: "Tinjau BRD",
      assignee_id: "u2",
      priority: "high",
      due_date: "2026-10-15T10:00:00.000Z",
    });

    const body = mocks.post.mock.calls[0]?.[1] as Record<string, unknown>;
    expect(body).not.toHaveProperty("status");
    expect(mocks.post).toHaveBeenCalledWith("/tasks", {
      project_id: "p1",
      title: "Tinjau BRD",
      assignee_id: "u2",
      priority: "high",
      due_date: "2026-10-15T10:00:00.000Z",
    });
  });
});

describe("updateTask", () => {
  it("mengirim hanya field yang diberikan", async () => {
    mocks.patch.mockResolvedValue({ data: { success: true, data: { id: "t1" } } });

    await updateTask("t1", { status: "in_progress" });

    expect(mocks.patch).toHaveBeenCalledWith("/tasks/t1", {
      status: "in_progress",
    });
  });

  it("tidak pernah mengirim project_id: memindahkan task bukan operasi yang ada", async () => {
    mocks.patch.mockResolvedValue({ data: { success: true, data: { id: "t1" } } });

    await updateTask("t1", { status: "open", title: "Judul baru" });

    const body = mocks.patch.mock.calls[0]?.[1] as Record<string, unknown>;
    expect(body).not.toHaveProperty("project_id");
  });
});

describe("completeTask", () => {
  it("memakai endpoint sendiri, bukan PATCH status (tabel transisi §6)", async () => {
    mocks.post.mockResolvedValue({ data: { success: true, data: { id: "t1" } } });

    await completeTask("t1");

    expect(mocks.post).toHaveBeenCalledWith("/tasks/t1/complete");
    expect(mocks.patch).not.toHaveBeenCalled();
  });
});

describe("toRfc3339FromLocal dan toLocalInputValue", () => {
  it("mengirim instan ber-offset, bukan tanggal yang zona waktunya ditebak", () => {
    const value = toRfc3339FromLocal("2026-03-31T14:30");
    expect(value).toMatch(/^2026-03-31T\d{2}:30:00\.000Z$/);
    expect(value.endsWith("Z") || /[+-]\d{2}:\d{2}$/.test(value)).toBe(true);
  });

  it("bolak-balik mempertahankan instan yang sama", () => {
    const instant = "2026-03-31T07:30:00.000Z";
    const local = toLocalInputValue(instant);
    expect(toRfc3339FromLocal(local)).toBe(instant);
  });

  it("mengembalikan string kosong untuk nilai kosong atau tidak sah", () => {
    expect(toRfc3339FromLocal("")).toBe("");
    expect(toRfc3339FromLocal("   ")).toBe("");
    expect(toRfc3339FromLocal("bukan tanggal")).toBe("");
    expect(toLocalInputValue("")).toBe("");
    expect(toLocalInputValue("bukan tanggal")).toBe("");
  });
});

describe("validateDueRange", () => {
  it("menerima rentang kosong: penyaringnya opsional", () => {
    const result = validateDueRange({ from: "", to: "" });
    expect(result.errors).toEqual({});
    expect(result.due_from).toBe("");
    expect(result.due_to).toBe("");
  });

  it("menolak rentang terbalik pada batas akhir, sesuai jawaban server", () => {
    const result = validateDueRange({
      from: "2026-03-31T10:00",
      to: "2026-03-01T10:00",
    });
    expect(result.errors.due_to).toBe(
      "Batas akhir tidak boleh mendahului batas awal.",
    );
  });

  it("menerima kedua batas yang sama persis: intervalnya tertutup", () => {
    const result = validateDueRange({
      from: "2026-03-31T10:00",
      to: "2026-03-31T10:00",
    });
    expect(result.errors).toEqual({});
    expect(result.due_from).toBe(result.due_to);
  });

  it("menandai batas yang bukan tanggal-waktu", () => {
    expect(validateDueRange({ from: "31/03/2026", to: "" }).errors).toHaveProperty(
      "due_from",
    );
    expect(validateDueRange({ from: "", to: "31/03/2026" }).errors).toHaveProperty(
      "due_to",
    );
  });
});

describe("validateTaskForm", () => {
  const valid = {
    project_id: "p1",
    title: "Tinjau BRD",
    description: "",
    assignee_id: "u2",
    priority: "medium" as const,
    due_date: "2026-10-15T10:00",
  };

  it("menerima form yang lengkap", () => {
    expect(validateTaskForm(valid)).toEqual({});
  });

  it("menuntut project, judul, penanggung jawab, prioritas, dan tenggat", () => {
    const errors = validateTaskForm({
      project_id: " ",
      title: "",
      description: "",
      assignee_id: "",
      priority: "",
      due_date: "",
    });

    expect(Object.keys(errors).sort()).toEqual([
      "assignee_id",
      "due_date",
      "priority",
      "project_id",
      "title",
    ]);
  });

  it("menegakkan batas panjang kolom (41-DATABASE.md §2.5)", () => {
    expect(validateTaskForm({ ...valid, title: "x".repeat(256) }).title).toBe(
      "Judul task maksimal 255 karakter.",
    );
    expect(
      validateTaskForm({ ...valid, description: "x".repeat(5001) }).description,
    ).toBe("Deskripsi maksimal 5000 karakter.");
  });
});
