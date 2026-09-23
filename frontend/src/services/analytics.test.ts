import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const { fetchDashboard, validateDashboardRange } = await import("./analytics");

beforeEach(() => {
  mocks.get.mockReset();
});

describe("fetchDashboard", () => {
  it("mengirim from/to/project_id bila diisi, membuang yang kosong", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: { kpis: {}, charts: {} } } });

    await fetchDashboard({ from: "2026-09-01T00:00:00Z", to: "2026-09-30T00:00:00Z", project_id: "p1" });

    expect(mocks.get).toHaveBeenCalledWith("/analytics/dashboard", {
      params: { from: "2026-09-01T00:00:00Z", to: "2026-09-30T00:00:00Z", project_id: "p1" },
    });
  });

  it("membuang query kosong", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: { kpis: {}, charts: {} } } });

    await fetchDashboard({});

    expect(mocks.get).toHaveBeenCalledWith("/analytics/dashboard", { params: {} });
  });

  it("mengembalikan data dashboard apa adanya", async () => {
    const data = { kpis: { total_documents: 42 }, charts: { statusDist: [] } };
    mocks.get.mockResolvedValue({ data: { success: true, data } });

    const result = await fetchDashboard({});

    expect(result).toEqual(data);
  });
});

describe("validateDashboardRange", () => {
  it("menerima rentang kosong dan rentang valid", () => {
    expect(validateDashboardRange({ from: "", to: "" }).errors).toEqual({});
    const { errors, from, to } = validateDashboardRange({ from: "2026-09-01T00:00", to: "2026-09-30T00:00" });
    expect(errors).toEqual({});
    expect(from).toMatch(/Z$/);
    expect(to).toMatch(/Z$/);
  });

  it("menolak rentang terbalik", () => {
    const { errors } = validateDashboardRange({ from: "2026-09-30T00:00", to: "2026-09-01T00:00" });
    expect(errors.to).toBeDefined();
  });
});
