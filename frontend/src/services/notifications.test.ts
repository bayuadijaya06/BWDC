import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  patch: vi.fn(),
  post: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const {
  listNotifications,
  markNotificationRead,
  markAllNotificationsRead,
  notificationTarget,
} = await import("./notifications");

beforeEach(() => {
  mocks.get.mockReset();
  mocks.patch.mockReset();
  mocks.post.mockReset();
});

describe("listNotifications", () => {
  it("selalu mengirim halaman dan batas, dan membuang is_read yang tidak diisi", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: null } });

    await listNotifications({ page: 2 });

    expect(mocks.get).toHaveBeenCalledWith("/notifications", {
      params: { page: 2, limit: 20 },
    });
  });

  it("mengirim is_read true dan false sebagai boolean, bukan string", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: null } });

    await listNotifications({ is_read: false });

    expect(mocks.get).toHaveBeenCalledWith("/notifications", {
      params: { page: 1, limit: 20, is_read: false },
    });
  });
});

describe("markNotificationRead", () => {
  it("PATCH /notifications/:id/read dengan id ter-encode", async () => {
    mocks.patch.mockResolvedValue({ data: { success: true, data: null } });

    await markNotificationRead("n1");

    expect(mocks.patch).toHaveBeenCalledWith("/notifications/n1/read");
  });
});

describe("markAllNotificationsRead", () => {
  it("POST /notifications/read-all dan mengembalikan jumlah yang diubah", async () => {
    mocks.post.mockResolvedValue({ data: { success: true, data: { updated: 3 } } });

    const updated = await markAllNotificationsRead();

    expect(mocks.post).toHaveBeenCalledWith("/notifications/read-all");
    expect(updated).toBe(3);
  });
});

describe("notificationTarget", () => {
  it("memetakan entitas yang punya halaman ke path-nya", async () => {
    const base = {
      id: "n1",
      type: "X",
      title: "t",
      message: "m",
      is_read: false,
      created_at: "2026-09-24T00:00:00+07:00",
    };
    expect(
      notificationTarget({ ...base, entity_type: "document", entity_id: "d1" }),
    ).toBe("/documents/d1");
    expect(
      notificationTarget({ ...base, entity_type: "task", entity_id: "t1" }),
    ).toBe("/tasks/t1");
    expect(
      notificationTarget({ ...base, entity_type: "project", entity_id: "p1" }),
    ).toBe("/projects/p1");
    expect(
      notificationTarget({ ...base, entity_type: "workflow_instance", entity_id: "wi-1" }),
    ).toBe("/approvals/wi-1");
  });

  it("mengembalikan null untuk comment, tanpa entitas, dan yang tidak dikenal", async () => {
    const base = {
      id: "n1",
      type: "X",
      title: "t",
      message: "m",
      is_read: false,
      created_at: "2026-09-24T00:00:00+07:00",
    };
    expect(
      notificationTarget({ ...base, entity_type: "comment", entity_id: "c1" }),
    ).toBeNull();
    expect(
      notificationTarget({ ...base, entity_type: null, entity_id: null }),
    ).toBeNull();
    expect(
      notificationTarget({ ...base, entity_type: "audit", entity_id: "a1" }),
    ).toBeNull();
  });
});
