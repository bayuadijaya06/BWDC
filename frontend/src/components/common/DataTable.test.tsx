import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { ApiError } from "@/services/http";

import { DataTable, type DataTableColumn } from "./DataTable";

interface Row {
  id: string;
  name: string;
  count: number;
}

const rows: Row[] = [
  { id: "a", name: "Dokumen A", count: 3 },
  { id: "b", name: "Dokumen B", count: 5 },
];

const columns: DataTableColumn<Row>[] = [
  { key: "name", header: "Nama", render: (row) => row.name },
  { key: "count", header: "Versi", align: "right", render: (row) => row.count },
];

function renderTable(
  overrides: Partial<Parameters<typeof DataTable<Row>>[0]> = {},
) {
  return render(
    <DataTable<Row>
      caption="Daftar dokumen"
      columns={columns}
      rows={rows}
      rowKey={(row) => row.id}
      emptyState={<p>Tidak ada dokumen</p>}
      {...overrides}
    />,
  );
}

describe("DataTable", () => {
  it("memakai tabel semantik dengan caption untuk pembaca layar", () => {
    renderTable();
    const table = screen.getByRole("table", { name: "Daftar dokumen" });
    expect(table).toBeInTheDocument();
    expect(screen.getAllByRole("columnheader")).toHaveLength(2);
    expect(screen.getAllByRole("row")).toHaveLength(3); // header + 2 baris
    expect(screen.getByText("Dokumen A")).toBeInTheDocument();
  });

  it("menampilkan keadaan kosong, bukan tabel tanpa baris", () => {
    renderTable({ rows: [] });
    expect(screen.getByText("Tidak ada dokumen")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("menampilkan keadaan gagal beserta kodenya, bukan tabel kosong", () => {
    const error = Object.assign(new Error("tidak dapat menghubungi server"), {
      status: 0,
      code: "NETWORK_ERROR",
      lockedForSeconds: null,
      isSessionLost: false,
    }) as unknown as ApiError;

    renderTable({ error });
    expect(screen.getByRole("alert")).toHaveTextContent(
      "tidak dapat menghubungi server",
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Kode: NETWORK_ERROR");
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("menawarkan muat ulang saat gagal", async () => {
    const onRetry = vi.fn();
    const error = Object.assign(
      new Error("server tidak menjawab tepat waktu"),
      {
        status: 0,
        code: "NETWORK_ERROR",
        lockedForSeconds: null,
        isSessionLost: false,
      },
    ) as unknown as ApiError;

    renderTable({ error, onRetry });
    await userEvent.click(screen.getByRole("button", { name: "Muat ulang" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("menulis rentang baris halaman dengan angka yang benar (C-048)", () => {
    renderTable({
      meta: { page: 2, limit: 20, total: 22, total_page: 2 },
      onPageChange: () => {},
    });
    expect(
      screen.getByText("Menampilkan 21 sampai 22 dari 22 rekam"),
    ).toBeInTheDocument();
    expect(screen.getByText("Halaman 2 dari 2")).toBeInTheDocument();
  });

  it("menonaktifkan tombol halaman di ujung rentang", () => {
    renderTable({
      meta: { page: 1, limit: 20, total: 22, total_page: 2 },
      onPageChange: () => {},
    });
    expect(screen.getByRole("button", { name: "Sebelumnya" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Berikutnya" })).toBeEnabled();
  });

  it("selalu menampilkan aksi baris, tidak hanya saat hover", () => {
    renderTable({
      actions: () => <button type="button">Hapus</button>,
      actionsHeader: "Aksi",
    });
    expect(screen.getAllByRole("button", { name: "Hapus" })).toHaveLength(2);
    expect(
      screen.getByRole("columnheader", { name: "Aksi" }),
    ).toBeInTheDocument();
  });
});
