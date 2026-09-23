import { Link, useLocation } from "react-router";

import { Panel } from "@/components/common/Panel";
import { EmptyState } from "@/components/common/States";
import { PageHeader } from "@/components/layout/PageHeader";
import { subNavFamily, type NavItem } from "@/config/navigation";
import { useAuthStore } from "@/store/auth";

/**
 * Halaman untuk modul yang belum dibangun.
 *
 * Halaman ini ada supaya tidak ada tautan navigasi yang menunjuk halaman tidak
 * ada (R-24) dan supaya keadaan kosongnya jujur (R-27): ia menyebut modul apa,
 * dokumen kontrak mana yang mengaturnya, dan bahwa datanya memang belum
 * diambil. Tidak ada tabel kosong berpura-pura dan tidak ada angka contoh.
 */
export function ModulePendingPage({ item }: { item: NavItem }) {
  const location = useLocation();
  const query = location.search.replace(/^\?/, "");
  const granted = useAuthStore((state) => state.profile?.permissions ?? []);

  // Halaman yang berada di bawah sebuah modul menampilkan baris sub-navigasi
  // modul itu — termasuk saat modul induknya masih berupa halaman penjelasan.
  // Tanpa itu, memindahkan halaman anak dari sidebar ke sub-navigasi (aturan
  // `51-UX.md` §2.1) akan membuatnya tidak punya jalan masuk sama sekali.
  // Bentuk barisnya sengaja sama dengan tab Documents dan Tasks; halaman yang
  // benar-benar dibangun akan menggantinya dengan barisnya sendiri.
  const family = subNavFamily(item.path, granted);

  return (
    <>
      <PageHeader
        title={item.label}
        description={`Modul ini belum dibangun di antarmuka. Kontraknya sudah ada: ${item.reference}`}
      />

      {family ? (
        <nav
          aria-label={`Sub-halaman ${family.parent.label}`}
          className="flex flex-wrap gap-1.5"
        >
          {[family.parent, ...family.links].map((page) => {
            const active = location.pathname === page.path;
            return (
              <Link
                key={page.path}
                to={page.path}
                aria-current={active ? "page" : undefined}
                className={[
                  "tap-target inline-flex items-center rounded-control border px-2.5 text-13",
                  active
                    ? "border-line-strong bg-surface-sunken font-medium text-text"
                    : "border-line text-text-soft hover:bg-surface-hover hover:text-text",
                ].join(" ")}
              >
                {page.label}
              </Link>
            );
          })}
        </nav>
      ) : null}

      <Panel
        title="Keadaan modul"
        note="Ditampilkan apa adanya, tanpa data contoh"
      >
        <div className="flex flex-col gap-3">
          <dl className="grid grid-cols-1 gap-x-6 gap-y-2 text-13 sm:grid-cols-2">
            <div className="flex gap-2">
              <dt className="w-28 shrink-0 text-text-muted">Rute</dt>
              <dd className="mono">{item.path}</dd>
            </div>
            <div className="flex gap-2">
              <dt className="w-28 shrink-0 text-text-muted">Izin menu</dt>
              <dd className="mono">
                {item.permissions.length > 0
                  ? item.permissions.join(", ")
                  : "tanpa izin khusus"}
              </dd>
            </div>
            <div className="flex gap-2">
              <dt className="w-28 shrink-0 text-text-muted">Dokumen</dt>
              <dd>{item.reference}</dd>
            </div>
            <div className="flex gap-2">
              <dt className="w-28 shrink-0 text-text-muted">Task</dt>
              <dd>{item.task}</dd>
            </div>
            {query ? (
              <div className="flex gap-2 sm:col-span-2">
                <dt className="w-28 shrink-0 text-text-muted">Penyaring</dt>
                <dd className="mono">{query}</dd>
              </div>
            ) : null}
          </dl>

          <EmptyState
            title="Belum ada data yang ditampilkan"
            description="Endpoint modul ini sudah berjalan di backend, tetapi antarmukanya belum dibuat. Tabel, penyaring, dan aksinya menyusul sesuai urutan Phase di 80-ROADMAP.md."
            action={
              <Link
                to="/"
                className="inline-block rounded-control border border-line-strong px-3 py-1.5 text-13 text-text hover:bg-surface-hover"
              >
                Kembali ke Dashboard
              </Link>
            }
          />
        </div>
      </Panel>
    </>
  );
}
