import { Link, useLocation } from "react-router";

import { PageHeader } from "@/components/layout/PageHeader";

export function NotFoundPage() {
  const location = useLocation();

  return (
    <>
      <PageHeader title="Halaman tidak ditemukan" />
      <div className="flex flex-col items-start gap-2 rounded-panel border border-dashed border-line bg-surface-raised px-4 py-6">
        <p className="text-13 text-text-soft">
          Tidak ada halaman pada rute{" "}
          <span className="mono">{location.pathname}</span>.
        </p>
        <Link
          to="/"
          className="rounded-control border border-line-strong px-3 py-1.5 text-13 text-text hover:bg-surface-hover"
        >
          Kembali ke Dashboard
        </Link>
      </div>
    </>
  );
}
