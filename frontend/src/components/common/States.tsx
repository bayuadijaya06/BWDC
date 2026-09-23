import type { ReactNode } from "react";

import { ApiError } from "@/services/http";
import { formatWait } from "@/utils/format";

/**
 * Keadaan kosong, memuat, dan gagal. Ketiganya wajib ada di setiap halaman
 * (antislop R-27, `51-UX.md` §7). Tidak ada ilustrasi: antislop R-22 melarang
 * ilustrasi generik tanpa hubungan dengan produk, dan tidak ada aset yang
 * boleh dikarang (R-23).
 */
export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-start gap-2 rounded-panel border border-dashed border-line bg-surface px-4 py-6">
      <p className="text-14 font-medium text-text">{title}</p>
      <p className="max-w-prose text-13 text-text-muted">{description}</p>
      {action ? <div className="pt-1">{action}</div> : null}
    </div>
  );
}

/** Pesan kesalahan yang menyebut kode dan, bila ada, lama tunggu. */
export function ErrorMessage({ error }: { error: ApiError }) {
  const wait = error.lockedForSeconds;
  return (
    <div
      role="alert"
      className="rounded-panel border border-danger/45 bg-status-rejected-surface px-3 py-2.5 text-13 text-status-rejected-ink"
    >
      <p>{error.message}</p>
      {wait !== null ? (
        <p className="pt-0.5">Coba lagi dalam {formatWait(wait)}.</p>
      ) : null}
      {error.status === 0 ? (
        <p className="pt-0.5">
          Periksa apakah server backend berjalan dan proxy dev mengarah ke
          portnya.
        </p>
      ) : null}
      <p className="pt-0.5 text-12 opacity-80">Kode: {error.code}</p>
    </div>
  );
}

export function ErrorState({
  error,
  onRetry,
}: {
  error: ApiError;
  onRetry?: () => void;
}) {
  return (
    <div className="flex flex-col items-start gap-2">
      <ErrorMessage error={error} />
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="tap-target rounded-control border border-line-strong bg-surface-raised px-3 text-13 text-text hover:bg-surface-hover"
        >
          Muat ulang
        </button>
      ) : null}
    </div>
  );
}

/** Kerangka pemuatan tabel: baris abu, tanpa animasi (MOTION 1). */
export function TableSkeleton({
  rows = 5,
  columns = 4,
}: {
  rows?: number;
  columns?: number;
}) {
  return (
    <div aria-hidden="true" className="flex flex-col gap-px bg-line">
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <div
          key={rowIndex}
          className="flex gap-4 bg-surface-raised px-3 py-2.5"
        >
          {Array.from({ length: columns }).map((__, columnIndex) => (
            <span
              key={columnIndex}
              className="h-3 flex-1 rounded-control bg-surface-sunken"
            />
          ))}
        </div>
      ))}
    </div>
  );
}
