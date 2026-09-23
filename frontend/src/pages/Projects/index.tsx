import { useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { Field } from "@/components/common/Field";
import { Dialog } from "@/components/common/Dialog";
import { EmptyState, ErrorMessage } from "@/components/common/States";
import { StatusBadge } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useArchiveProject, useProjectList } from "@/queries/projects";
import { ApiError } from "@/services/http";
import type { Project } from "@/services/projects";
import { useAuthStore } from "@/store/auth";
import { projectStatus, type ProjectStatus } from "@/types/status";
import { EMPTY_DATE, EMPTY_VALUE } from "@/utils/format";

import { CreateProjectDialog } from "./CreateProjectDialog";

/**
 * Project List (`50-FSD.md` §3.1).
 *
 * Dua hal yang dipegang halaman ini:
 *
 * 1. **Penyaring dan halaman hidup di URL**, bukan di state komponen. Alamat
 *    `/projects?status=archived&page=2` dapat dikirim ke rekan kerja dan
 *    menghasilkan daftar yang sama; tombol Back peramban juga berperilaku
 *    benar. Halaman ini tidak butuh baris tab: "List" adalah halaman itu
 *    sendiri dan "Create" adalah tombol aksi di header, bukan sub-halaman
 *    (`51-UX.md` §2.1).
 * 2. **Arsip bukan hapus** (`FR-PROJ-07`). Konfirmasinya menyebut akibatnya
 *    dengan kalimat, dan sesudah diarsipkan project tetap dapat dibuka serta
 *    tetap muncul pada penyaring `Archived`.
 */
export function ProjectsPage() {
  const [params, setParams] = useSearchParams();
  const navigateTo = useNavigate();
  const canCreate = useAuthStore((state) => state.has("project:create"));
  const canArchive = useAuthStore((state) => state.has("project:archive"));
  const archiveProject = useArchiveProject();

  const page = Math.max(Number(params.get("page") ?? "1") || 1, 1);
  const statusParam = params.get("status") ?? "";
  const status: ProjectStatus | "" =
    statusParam === "active" || statusParam === "archived" ? statusParam : "";
  const search = params.get("search") ?? "";
  const openCreate = params.get("view") === "create";

  const [searchDraft, setSearchDraft] = useState(search);
  const [archiveTarget, setArchiveTarget] = useState<Project | null>(null);
  const [archiveError, setArchiveError] = useState<ApiError | null>(null);

  const query = useProjectList({ page, limit: 20, status, search });
  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];

  /** Menulis parameter baru; nilai kosong dibuang supaya URL tetap pendek. */
  function navigate(next: Record<string, string | null>) {
    const updated = new URLSearchParams(params);
    for (const [key, value] of Object.entries(next)) {
      if (value === null || value === "") updated.delete(key);
      else updated.set(key, value);
    }
    setParams(updated);
  }

  const filtered = status !== "" || search.trim() !== "";

  const columns: DataTableColumn<Project>[] = [
    {
      key: "code",
      header: "Kode",
      width: "108px",
      render: (row) => (
        <span className="font-mono text-12 text-text-soft">{row.code}</span>
      ),
    },
    {
      key: "name",
      header: "Nama",
      render: (row) => (
        <Link
          to={`/projects/${row.id}`}
          className="rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          {row.name}
        </Link>
      ),
    },
    {
      key: "status",
      header: "Status",
      width: "120px",
      render: (row) => (
        <StatusBadge presentation={projectStatus(row.status)} />
      ),
    },
    {
      key: "owner",
      header: "Pemilik",
      width: "140px",
      render: (row) => row.owner_username ?? EMPTY_VALUE,
    },
    {
      key: "start_date",
      header: "Mulai",
      width: "108px",
      render: (row) => row.start_date ?? EMPTY_DATE,
    },
    {
      key: "target_end_date",
      header: "Target",
      width: "108px",
      render: (row) => row.target_end_date ?? EMPTY_DATE,
    },
    {
      key: "member_count",
      header: "Anggota",
      align: "right",
      width: "84px",
      render: (row) => row.member_count,
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Projects"
        description={
          meta
            ? `${meta.total} project dalam cakupan Anda. Project di luar keanggotaan Anda tidak dikirim server.`
            : "Daftar project dalam cakupan Anda."
        }
        actions={
          canCreate ? (
            <Button
              variant="primary"
              onClick={() => navigate({ view: "create" })}
            >
              Buat project
            </Button>
          ) : null
        }
      />

      <form
        role="search"
        aria-label="Penyaring project"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={(event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          navigate({ search: searchDraft.trim(), page: null });
        }}
      >
        <div className="min-w-[220px] flex-1">
          <Field
            label="Cari"
            value={searchDraft}
            onChange={(event) => setSearchDraft(event.target.value)}
            placeholder="Nama atau kode project"
            maxLength={255}
          />
        </div>
        {/*
          `min-w-0` wajib di setiap kolom ber-`select` (alasan lengkap di
          `Documents/index.tsx` dan `51-UX.md` §2.1): ukuran minimum otomatis
          kolom flex adalah min-content anaknya, dan min-content sebuah
          `<select>` ditentukan teks pilihan terpanjang. Cacat ini sempat lolos
          dari pemeriksaan tata letak justru karena pilihan di halaman ini
          pendek dan tetap: pemeriksa yang mengandalkan data yang kebetulan ada
          akan selalu melaporkan OK. Yang menemukannya probe stres di
          `scripts/responsive-evidence.mjs`, dan sekarang ia menguji aturannya —
          panjang pilihan tidak menentukan lebar halaman.
        */}
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="penyaring-status"
            className="text-13 font-medium text-text-soft"
          >
            Status
          </label>
          <select
            id="penyaring-status"
            value={status}
            onChange={(event) =>
              navigate({ status: event.target.value, page: null })
            }
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua</option>
            {(["active", "archived"] as ProjectStatus[]).map((value) => (
              <option key={value} value={value}>
                {projectStatus(value).label}
              </option>
            ))}
          </select>
        </div>
        <Button type="submit">Cari</Button>
        {filtered ? (
          <Button
            variant="quiet"
            onClick={() => {
              setSearchDraft("");
              navigate({ search: null, status: null, page: null });
            }}
          >
            Bersihkan penyaring
          </Button>
        ) : null}
      </form>

      <DataTable
        caption="Daftar project pada halaman ini"
        columns={columns}
        rows={rows}
        rowKey={(row) => row.id}
        tone={(row) => projectStatus(row.status).tone}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={(next) => navigate({ page: String(next) })}
        actions={(row) =>
          canArchive && row.status === "active" ? (
            <Button
              variant="quiet"
              onClick={() => {
                setArchiveError(null);
                setArchiveTarget(row);
              }}
            >
              Arsipkan
            </Button>
          ) : null
        }
        emptyState={
          filtered ? (
            <EmptyState
              title="Tidak ada project yang cocok"
              description="Penyaring yang sedang aktif tidak menyisakan satu project pun. Cakupan data tetap berlaku: project di luar keanggotaan Anda memang tidak pernah dikirim server."
              action={
                <Button
                  variant="quiet"
                  onClick={() => {
                    setSearchDraft("");
                    navigate({ search: null, status: null, page: null });
                  }}
                >
                  Bersihkan penyaring
                </Button>
              }
            />
          ) : (
            <EmptyState
              title="Belum ada project"
              description={
                canCreate
                  ? "Buat project pertama; kode project menjadi awalan nomor dokumen di dalamnya."
                  : "Anda belum terdaftar sebagai anggota project mana pun. Minta Manager atau Administrator menambahkan Anda."
              }
              action={
                canCreate ? (
                  <Button
                    variant="primary"
                    onClick={() => navigate({ view: "create" })}
                  >
                    Buat project
                  </Button>
                ) : null
              }
            />
          )
        }
      />

      {openCreate ? (
        <CreateProjectDialog
          onClose={() => navigate({ view: null })}
          onCreated={(projectId) => {
            // Detail project hasil 201 sudah masuk cache, jadi halaman tujuan
            // tidak memanggil ulang `GET /projects/:id`.
            navigate({ view: null });
            navigateTo(`/projects/${projectId}`);
          }}
        />
      ) : null}

      {/* Dialog tidak dapat ditutup selama permintaan berjalan: menutupnya
          menyembunyikan hasilnya, dan pengguna akan menekan Arsipkan lagi. */}
      {archiveTarget ? (
        <Dialog
          title="Arsipkan project"
          description="Project terarsip tetap tersimpan beserta dokumen dan tugasnya."
          onClose={
            archiveProject.isPending ? () => {} : () => setArchiveTarget(null)
          }
          footer={
            <>
              <Button
                variant="quiet"
                onClick={() => setArchiveTarget(null)}
                disabled={archiveProject.isPending}
              >
                Batal
              </Button>
              <Button
                variant="danger"
                pending={archiveProject.isPending}
                onClick={() => {
                  archiveProject.mutate(archiveTarget.id, {
                    onSuccess: () => {
                      setArchiveTarget(null);
                      setArchiveError(null);
                    },
                    onError: (error) => {
                      setArchiveError(
                        error instanceof ApiError ? error : null,
                      );
                    },
                  });
                }}
              >
                Arsipkan project
              </Button>
            </>
          }
        >
          <p className="text-13 text-text">
            <span className="font-mono text-12">{archiveTarget.code}</span>{" "}
            {archiveTarget.name} akan berstatus <strong>Archived</strong>. Tidak
            ada baris, versi dokumen, atau berkas yang dihapus; project tetap
            dapat dibuka dan tetap muncul pada penyaring Archived.
          </p>
          {archiveError ? (
            <div className="pt-2.5">
              <ErrorMessage error={archiveError} />
            </div>
          ) : null}
        </Dialog>
      ) : null}
    </div>
  );
}
