import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const { exportReport } = await import("./reports");

beforeEach(() => {
  mocks.get.mockReset();
});

function blobResponse(disposition: string | null) {
  const headers: Record<string, string> = {};
  if (disposition !== null) headers["content-disposition"] = disposition;
  return { data: new Blob(["a,b"]), headers };
}

describe("exportReport", () => {
  it("selalu mengirim type + format csv, dan membuang penyaring kosong", async () => {
    mocks.get.mockResolvedValue(blobResponse(null));

    await exportReport({ type: "projects", search: "   ", status: "" });

    expect(mocks.get).toHaveBeenCalledWith("/reports/export", {
      params: { type: "projects", format: "csv" },
      responseType: "blob",
    });
  });

  it("meneruskan search/status/project_id yang terisi", async () => {
    mocks.get.mockResolvedValue(blobResponse(null));

    await exportReport({
      type: "documents",
      search: "  BRD  ",
      status: "in_review",
      project_id: "p1",
    });

    expect(mocks.get).toHaveBeenCalledWith("/reports/export", {
      params: {
        type: "documents",
        format: "csv",
        search: "BRD",
        status: "in_review",
        project_id: "p1",
      },
      responseType: "blob",
    });
  });

  it("memakai nama berkas dari Content-Disposition server", async () => {
    mocks.get.mockResolvedValue(
      blobResponse('attachment; filename="bwdcs-tasks-20260925.csv"'),
    );

    const file = await exportReport({ type: "tasks" });

    expect(file.filename).toBe("bwdcs-tasks-20260925.csv");
    expect(file.blob).toBeInstanceOf(Blob);
  });

  it("jatuh ke nama cadangan tanpa tanggal bila header tidak ada", async () => {
    mocks.get.mockResolvedValue(blobResponse(null));

    const file = await exportReport({ type: "projects" });

    expect(file.filename).toBe("bwdcs-projects.csv");
  });
});
