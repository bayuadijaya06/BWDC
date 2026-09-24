import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const {
  listWorkflowInstances,
  fetchWorkflowInstance,
  actWorkflowInstance,
  resubmitWorkflowInstance,
  listWorkflowDefinitions,
  submitWorkflowInstance,
} = await import("./workflows");

function meta(total = 0) {
  return { page: 1, limit: 20, total, total_page: total === 0 ? 0 : 1 };
}

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
});

describe("listWorkflowInstances", () => {
  it("selalu mengirim halaman dan batas, dan membuang status/scope kosong", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listWorkflowInstances({ page: 2, limit: 20, status: "", scope: "" });

    expect(mocks.get).toHaveBeenCalledWith("/workflows/instances", {
      params: { page: 2, limit: 20 },
    });
  });

  it("mengirim status kanonik dan scope assigned_to_me saat diminta", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listWorkflowInstances({ status: "running", scope: "assigned_to_me" });

    expect(mocks.get).toHaveBeenCalledWith("/workflows/instances", {
      params: { page: 1, limit: 20, status: "running", scope: "assigned_to_me" },
    });
  });

  it("memakai meta pengganti yang jujur bila server tidak mengirimnya", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [] } });

    const result = await listWorkflowInstances({ status: "completed" });

    expect(result.items).toEqual([]);
    expect(result.meta).toEqual({ page: 1, limit: 20, total: 0, total_page: 0 });
  });
});

describe("fetchWorkflowInstance", () => {
  it("mengambil detail instance dengan id ter-encode", async () => {
    mocks.get.mockResolvedValue({
      data: { success: true, data: { id: "wi-1", status: "running" } },
    });

    const result = await fetchWorkflowInstance("wi-1");

    expect(mocks.get).toHaveBeenCalledWith("/workflows/instances/wi-1");
    expect(result).toEqual({ id: "wi-1", status: "running" });
  });
});

describe("actWorkflowInstance", () => {
  it("POST /workflows/instances/:id/actions dengan action dan version", async () => {
    mocks.post.mockResolvedValue({
      data: { success: true, data: { id: "wi-1", version: 1, status: "running" } },
    });

    const result = await actWorkflowInstance("wi-1", {
      action: "approve",
      comment: "  Looks good  ",
      version: 0,
    });

    expect(mocks.post).toHaveBeenCalledWith("/workflows/instances/wi-1/actions", {
      action: "approve",
      comment: "  Looks good  ",
      version: 0,
    });
    // Server membalas instance terbarunya; klien tidak mengubah version.
    expect(result.version).toBe(1);
  });

  it("mengirim action tanpa comment/version bila tidak ada", async () => {
    mocks.post.mockResolvedValue({ data: { success: true, data: {} } });

    await actWorkflowInstance("wi-2", { action: "reject" });

    expect(mocks.post).toHaveBeenCalledWith("/workflows/instances/wi-2/actions", {
      action: "reject",
    });
  });
});

describe("resubmitWorkflowInstance", () => {
  it("POST /workflows/instances/:id/resubmit tanpa version bila tidak dikirim", async () => {
    mocks.post.mockResolvedValue({ data: { success: true, data: { id: "wi-1", status: "running" } } });

    await resubmitWorkflowInstance("wi-1");

    expect(mocks.post).toHaveBeenCalledWith("/workflows/instances/wi-1/resubmit", {});
  });

  it("mengirim version bila disediakan", async () => {
    mocks.post.mockResolvedValue({ data: { success: true, data: {} } });

    await resubmitWorkflowInstance("wi-1", 3);

    expect(mocks.post).toHaveBeenCalledWith("/workflows/instances/wi-1/resubmit", { version: 3 });
  });
});

describe("listWorkflowDefinitions", () => {
  it("GET /workflows/definitions tanpa parameter", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [] } });

    const result = await listWorkflowDefinitions();

    expect(mocks.get).toHaveBeenCalledWith("/workflows/definitions");
    expect(result).toEqual([]);
  });
});

describe("submitWorkflowInstance", () => {
  it("POST /workflows/submit dengan document_id dan workflow_definition_id", async () => {
    mocks.post.mockResolvedValue({
      data: { success: true, data: { id: "wi-1", status: "running", version: 0 } },
    });

    const result = await submitWorkflowInstance({ document_id: "d1", workflow_definition_id: "wd-1" });

    expect(mocks.post).toHaveBeenCalledWith("/workflows/submit", {
      document_id: "d1",
      workflow_definition_id: "wd-1",
    });
    expect(result.version).toBe(0);
  });
});
