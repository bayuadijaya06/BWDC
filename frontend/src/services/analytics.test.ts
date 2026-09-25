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

  it("menormalkan respons backend lama tanpa chart/KPI Phase 5 (temuan P-084)", async () => {
    // Backend sebelum migrasi 012 mengirim 8 KPI + 8 chart; halaman tidak boleh
    // meledak menjadi layar putih — panel menampilkan keadaan kosongnya.
    const legacy = {
      kpis: { total_documents: 42, avg_approval_time_hours: 52.3 },
      charts: { statusDist: [{ status: "draft", count: 10 }] },
    };
    mocks.get.mockResolvedValue({ data: { success: true, data: legacy } });

    const result = await fetchDashboard({});

    expect(result.kpis.total_documents).toBe(42);
    expect(result.kpis.avg_approval_time_hours).toBe(52.3);
    expect(result.kpis.sla_on_time).toBe(0);
    expect(result.kpis.review_due).toBe(0);
    expect(result.charts.statusDist).toEqual([{ status: "draft", count: 10 }]);
    expect(result.charts.byDepartment).toEqual([]);
    expect(result.charts.slaBreakdown).toEqual([]);
    expect(result.charts.reviewDueTrend).toEqual([]);
    expect(result.charts.funnel).toEqual({
      draft: 0,
      in_review: 0,
      revision_required: 0,
      approved: 0,
      rejected: 0,
    });
  });

  it("mengembalikan data dashboard lengkap apa adanya", async () => {
    const data = {
      kpis: { total_documents: 42 },
      charts: { statusDist: [] },
    };
    mocks.get.mockResolvedValue({ data: { success: true, data } });

    const result = await fetchDashboard({});

    // Bentuk penuh yang dinormalkan: field yang dikirim bertahan, yang hilang
    // diisi netral — tidak pernah `undefined`.
    expect(result.kpis.total_documents).toBe(42);
    expect(result.charts.statusDist).toEqual([]);
    expect(result.charts.byDepartment).toEqual([]);
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
