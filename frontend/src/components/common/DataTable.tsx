import type { ReactNode } from "react";

import type { ApiMeta } from "@/types/api";
import type { StatusTone } from "@/types/status";
import type { ApiError } from "@/services/http";

import { EmptyState, ErrorState, TableSkeleton } from "./States";
import { Spine } from "./StatusBadge";

/**
 * Tabel rekam. Semantiknya `<table>` asli (bukan div ber-ARIA), karena itulah
 * yang memberi navigasi papan ketik dan pembaca layar tanpa usaha tambahan.
 *
 * Aturan yang dikodekan di sini dan bukan diserahkan ke pemanggil:
 * - kolom angka rata kanan dan memakai tabular numerals (digit sejajar);
 * - aksi baris selalu terlihat, tidak pernah muncul hanya saat hover;
 * - `Density` default compact 36px sesuai `DESIGN.md` §4;
 * - keadaan memuat, kosong, dan gagal punya tempatnya sendiri.
 */
export interface DataTableColumn<T> {
  key: string;
  header: string;
  align?: "left" | "right";
  width?: string;
  render: (row: T) => ReactNode;
}

export interface DataTableProps<T> {
  /** Dipakai sebagai `<caption>`: menyebut data apa yang sedang dilihat. */
  caption: string;
  columns: DataTableColumn<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  /** Nada punggung rekam per baris. Opsional: tidak semua tabel punya status. */
  tone?: (row: T) => StatusTone;
  density?: "compact" | "comfortable";
  loading?: boolean;
  error?: ApiError | null;
  onRetry?: () => void;
  emptyState: ReactNode;
  meta?: ApiMeta;
  onPageChange?: (page: number) => void;
  actions?: (row: T) => ReactNode;
  actionsHeader?: string;
}

export function DataTable<T>({
  caption,
  columns,
  rows,
  rowKey,
  tone,
  density = "compact",
  loading = false,
  error = null,
  onRetry,
  emptyState,
  meta,
  onPageChange,
  actions,
  actionsHeader = "Aksi",
}: DataTableProps<T>) {
  if (error) return <ErrorState error={error} onRetry={onRetry} />;
  if (loading)
    return <TableSkeleton columns={columns.length + (actions ? 1 : 0)} />;
  if (rows.length === 0)
    return (
      emptyState ?? (
        <EmptyState
          title="Tidak ada rekam"
          description="Belum ada data yang cocok dengan penyaring saat ini."
        />
      )
    );

  const rowHeight = density === "compact" ? "h-9" : "h-11";
  const firstPageRow = meta ? (meta.page - 1) * meta.limit + 1 : null;
  const lastPageRow = meta ? firstPageRow! + rows.length - 1 : null;
  const hasPrev = (meta?.page ?? 1) > 1;
  const hasNext = meta ? meta.page < meta.total_page : false;

  return (
    <div className="flex flex-col">
      <div className="overflow-x-auto rounded-panel border border-line bg-surface-raised">
        <table className="w-full border-collapse text-left">
          <caption className="sr-only">{caption}</caption>
          <thead className="sticky top-0 z-10 bg-surface-raised">
            <tr className="border-b border-line-strong">
              {tone ? (
                <th scope="col" className="w-[3px] p-0">
                  {/* Teks tersembunyi, bukan `aria-label`: kolom punggung
                      rekam tetap butuh nama yang terbaca dari isinya, dan
                      header kosong ditandai pelanggaran oleh axe
                      (`empty-table-header`). */}
                  <span className="sr-only">Status</span>
                </th>
              ) : null}
              {columns.map((column) => (
                <th
                  key={column.key}
                  scope="col"
                  style={column.width ? { width: column.width } : undefined}
                  className={[
                    "index-label px-3 py-2",
                    column.align === "right" ? "text-right" : "",
                  ].join(" ")}
                >
                  {column.header}
                </th>
              ))}
              {actions ? (
                <th scope="col" className="index-label px-3 py-2 text-right">
                  {actionsHeader}
                </th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr
                key={rowKey(row)}
                className="border-b border-line last:border-b-0 hover:bg-surface-hover"
              >
                {tone ? (
                  <td className="w-[3px] p-0">
                    <div className={rowHeight}>
                      <Spine tone={tone(row)} />
                    </div>
                  </td>
                ) : null}
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={[
                      "px-3 text-13 text-text",
                      rowHeight,
                      column.align === "right" ? "text-right tabular-nums" : "",
                    ].join(" ")}
                  >
                    {column.render(row)}
                  </td>
                ))}
                {actions ? (
                  <td className={["px-3 text-right", rowHeight].join(" ")}>
                    {actions(row)}
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {meta ? (
        <div className="flex flex-wrap items-center justify-between gap-2 pt-2.5 text-12 text-text-muted">
          <p>
            Menampilkan {firstPageRow} sampai {lastPageRow} dari {meta.total}{" "}
            rekam
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => onPageChange?.(meta.page - 1)}
              disabled={!hasPrev}
              className="tap-target rounded-control border border-line-strong px-2.5 text-12 text-text hover:bg-surface-hover disabled:opacity-45"
            >
              Sebelumnya
            </button>
            <span className="tabular-nums">
              Halaman {meta.page} dari {Math.max(meta.total_page, 1)}
            </span>
            <button
              type="button"
              onClick={() => onPageChange?.(meta.page + 1)}
              disabled={!hasNext}
              className="tap-target rounded-control border border-line-strong px-2.5 text-12 text-text hover:bg-surface-hover disabled:opacity-45"
            >
              Berikutnya
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
