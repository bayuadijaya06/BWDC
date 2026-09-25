import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const {
  createComment,
  deleteComment,
  listComments,
  updateComment,
  validateCommentContent,
} = await import("./comments");

function meta(total = 0) {
  return { page: 1, limit: 20, total, total_page: total === 0 ? 0 : 1 };
}

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
});

describe("listComments", () => {
  it("selalu mengirim entity_type dan entity_id", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listComments({ entity_type: "document", entity_id: "d1" });

    expect(mocks.get).toHaveBeenCalledWith("/comments", {
      params: { page: 1, limit: 20, entity_type: "document", entity_id: "d1" },
    });
  });
});

describe("endpoint tulis", () => {
  it("POST /comments mengirim parent_id bila balasan", async () => {
    mocks.post.mockResolvedValue({
      data: { success: true, data: { id: "c2", parent_id: "c1" } },
    });

    const reply = await createComment({
      entity_type: "document",
      entity_id: "d1",
      content: "Setuju.",
      parent_id: "c1",
    });

    expect(mocks.post).toHaveBeenLastCalledWith("/comments", {
      entity_type: "document",
      entity_id: "d1",
      content: "Setuju.",
      parent_id: "c1",
    });
    expect(reply.parent_id).toBe("c1");
  });

  it("PATCH /comments/:id hanya mengirim content", async () => {
    mocks.patch.mockResolvedValue({ data: { success: true, data: { id: "c1" } } });

    await updateComment("c1", { content: "Revisi." });

    expect(mocks.patch).toHaveBeenCalledWith("/comments/c1", { content: "Revisi." });
  });

  it("DELETE /comments/:id tanpa body", async () => {
    mocks.delete.mockResolvedValue({ data: { success: true, data: null } });

    await deleteComment("c1");

    expect(mocks.delete).toHaveBeenCalledWith("/comments/c1");
  });
});

describe("validateCommentContent", () => {
  it("menolak kosong dan melebihi 2000 rune", () => {
    expect(validateCommentContent("   ")).toBe("Isi komentar wajib diisi.");
    expect(validateCommentContent("a".repeat(2001))).toBe(
      "Isi komentar maksimal 2000 karakter.",
    );
    expect(validateCommentContent("  ok  ")).toBeNull();
  });
});
