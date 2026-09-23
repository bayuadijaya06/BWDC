import { useState, type FormEvent, type MouseEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";

import {
  toLocalInputValue,
  validateUpdatedAtRange,
} from "@/services/documents";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { Field } from "@/components/common/Field";
import { EmptyState } from "@/components/common/States";
import { StatusBadge } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useDocumentList } from "@/queries/documents";
import { ApiError } from "@/services/http";
import { useProjectList } from "@/queries/projects";
import type { DocumentRecord } from "@/services/documents";
import { useAuthStore } from "@/store/auth";
import { documentStatus, type DocumentStatus } from "@/types/status";
import { EMPTY_VALUE, formatTimestamp } from "@/utils/format";

import { CreateDocumentDialog } from "./CreateDocumentDialog";

/** Enam nilai kanonik `50-FSD.md` §11.1 (ADR-0012), bukan daftar karangan. */
const documentStatuses: DocumentStatus[] = [
  "draft",
  "in_review",
  "revision_required",
  "approved",
  "rejected",
  "archived",
];

/**
 * Sub-halaman `50-FSD.md` §4.1 (`51-UX.md` §2.1), dalam bentuk yang benar-
 * benar ada di kontrak `42-API.md` §4.
 *
 * Baris tab ini dulu hidup di sidebar. Ia dipindahkan ke halaman yang memiliki
 * daftarnya (`51-UX.md` §2.1) karena penyaring status sudah ada di form bawah;
 * dua tempat memilih untuk hal yang sama membuat menu panjang tanpa menambah
 * kemampuan. Yang **tidak** boleh hilang adalah penyaring `Milik saya`: ia
 * tetap dapat dibuka dari sini, dan halaman menyatakan alasannya tidak dapat
 * dijalankan (Q-016) alih-alih diam-diam menampilkan seluruh dokumen.
 */
const subPages: {
  label: string;
  query: string;
  matches: (state: { status: DocumentStatus | ""; mine: boolean; others: boolean }) => boolean;
}[] = [
  {
    label: "Semua",
    query: "",
    matches: ({ status, mine, others }) =>
      status === "" && !mine && !others,
  },
  { label: "Milik saya", query: "?view=mine", matches: ({ mine }) => mine },
  {
    label: "Pending Review",
    query: "?status=in_review",
    matches: ({ status }) => status === "in_review",
  },
  {
    label: "Revision Required",
    query: "?status=revision_required",
    matches: ({ status }) => status === "revision_required",
  },
  {
    label: "Approved",
    query: "?status=approved",
    matches: ({ status }) => status === "approved",
  },
];

/**
 * Document List (`50-FSD.md` §4.1).
 *
 * Yang dipegang halaman ini:
 *
 * 1. **Penyaring dan halaman hidup di URL.** `/documents?status=in_review` dapat
 *    dikirim ke rekan kerja dan menghasilkan daftar yang sama; baris tab
 *    di atas form memakai bentuk yang sama, dan berada di sini karena inilah
 *    halaman yang mendaftar (`51-UX.md` §2.1).
 * 2. **Kolomnya persis `50-FSD.md` §4.1.** Tidak ada kolom tambahan yang
 *    dikarang (mis. nama project) — nama project sudah terbaca dari prefiks
 *    `document_number` dan dari penyaringnya.
 * 3. **Tanpa `?status=`, dokumen terarsip tidak ikut.** Aturan itu dijalankan
 *    server di dalam kueri (ADR-0019 butir 5), jadi klien tidak menyaringnya
 *    sendiri dan tidak menghitung `total` versinya sendiri.
 */
export function DocumentsPage() {
  const [params, setParams] = useSearchParams();
  const navigateTo = useNavigate();
  const canCreate = useAuthStore((state) => state.has("document:create"));
  const canReadProjects = useAuthStore((state) => state.has("project:read"));

  const page = Math.max(Number(params.get("page") ?? "1") || 1, 1);
  const statusParam = params.get("status") ?? "";
  const status: DocumentStatus | "" = documentStatuses.includes(
    statusParam as DocumentStatus,
  )
    ? (statusParam as DocumentStatus)
    : "";
  const search = params.get("search") ?? "";
  const projectId = params.get("project_id") ?? "";
  const updatedFrom = params.get("updated_from") ?? "";
  const updatedTo = params.get("updated_to") ?? "";
  const onlyMine = params.get("view") === "mine";
  const openUpload = params.get("upload") === "1";

  const [searchDraft, setSearchDraft] = useState(search);
  const [updatedFromDraft, setUpdatedFromDraft] = useState(
    toLocalInputValue(updatedFrom),
  );
  const [updatedToDraft, setUpdatedToDraft] = useState(
    toLocalInputValue(updatedTo),
  );
  const [rangeErrors, setRangeErrors] = useState<Record<string, string>>({});

  const query = useDocumentList({
    page,
    limit: 20,
    status,
    search,
    project_id: projectId,
    updated_from: updatedFrom,
    updated_to: updatedTo,
  });
  // Daftar project hanya dipakai untuk mengisi pilihan penyaring, jadi ia tidak
  // diminta bila izinnya tidak ada atau bila penyaringnya tidak sedang dipakai.
  const projects = useProjectList(
    { limit: 100 },
    { enabled: canReadProjects },
  );

  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];
  const filtered =
    status !== "" ||
    search.trim() !== "" ||
    projectId !== "" ||
    updatedFrom !== "" ||
    updatedTo !== "";
  /** Menulis parameter baru; nilai kosong dibuang supaya URL tetap pendek. */
  function navigate(next: Record<string, string | null>) {
    const updated = new URLSearchParams(params);
    for (const [key, value] of Object.entries(next)) {
      if (value === null || value === "") updated.delete(key);
      else updated.set(key, value);
    }
    setParams(updated);
  }

  function clearFilters() {
    setSearchDraft("");
    setUpdatedFromDraft("");
    setUpdatedToDraft("");
    setRangeErrors({});
    navigate({
      search: null,
      status: null,
      project_id: null,
      updated_from: null,
      updated_to: null,
      page: null,
    });
  }

  /**
   * Satu aksi terapkan untuk pencarian **dan** rentang. Alasannya bentuk HTML:
   * Enter di dalam isian mana pun memicu submit tombol submit pertama, jadi
   * dengan dua aksi terpisah Enter di isian tanggal diam-diam menjalankan
   * pencarian dan membuang draft rentangnya (atau sebaliknya). Menyatukannya
   * berarti tidak ada masukan yang hilang, dan rentang yang tidak sah menahan
   * semuanya dengan pesan yang menyebut batasnya — pola yang sama dengan
   * penyaring Tasks (P-049).
   */
  function applyFilters(
    event: FormEvent<HTMLFormElement> | MouseEvent<HTMLButtonElement>,
  ) {
    event.preventDefault();
    const { errors, updated_from, updated_to } = validateUpdatedAtRange({
      from: updatedFromDraft,
      to: updatedToDraft,
    });
    setRangeErrors(errors);
    if (Object.keys(errors).length > 0) return;
    navigate({
      search: searchDraft.trim(),
      updated_from,
      updated_to,
      page: null,
    });
  }

  const columns: DataTableColumn<DocumentRecord>[] = [
    {
      key: "document_number",
      header: "Nomor",
      width: "128px",
      render: (row) => (
        <span className="font-mono text-12 text-text-soft">
          {row.document_number}
        </span>
      ),
    },
    {
      key: "title",
      header: "Judul",
      render: (row) => (
        <Link
          to={`/documents/${row.id}`}
          className="rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          {row.title}
        </Link>
      ),
    },
    {
      key: "category",
      header: "Kategori",
      width: "150px",
      render: (row) => row.category_name ?? EMPTY_VALUE,
    },
    {
      key: "status",
      header: "Status",
      width: "150px",
      render: (row) => (
        <StatusBadge presentation={documentStatus(row.status)} />
      ),
    },
    {
      key: "version",
      header: "Versi",
      width: "80px",
      render: (row) =>
        row.latest_version ?? (
          <span className="text-text-muted">Belum ada versi</span>
        ),
    },
    {
      key: "owner",
      header: "Pemilik",
      width: "130px",
      render: (row) => row.owner_username ?? EMPTY_VALUE,
    },
    {
      key: "updated_at",
      header: "Diperbarui",
      width: "170px",
      render: (row) => formatTimestamp(row.updated_at),
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Documents"
        description={
          meta
            ? `${meta.total} dokumen dalam cakupan Anda. Dokumen di luar keanggotaan project tidak dikirim server.`
            : "Daftar dokumen dalam cakupan Anda."
        }
        actions={
          canCreate ? (
            <Button
              variant="primary"
              onClick={() => navigate({ upload: "1" })}
            >
              Unggah dokumen
            </Button>
          ) : null
        }
      />

      <nav aria-label="Sub-halaman dokumen" className="flex flex-wrap gap-1.5">
        {subPages.map((sub) => {
          const active = sub.matches({
            status,
            mine: onlyMine,
            others: search.trim() !== "" || projectId !== "",
          });
          return (
            <Link
              key={sub.label}
              to={{ pathname: "/documents", search: sub.query }}
              aria-current={active ? "page" : undefined}
              className={[
                "tap-target inline-flex items-center rounded-control border px-2.5 text-13",
                active
                  ? "border-line-strong bg-surface-sunken font-medium text-text"
                  : "border-line text-text-soft hover:bg-surface-hover hover:text-text",
              ].join(" ")}
            >
              {sub.label}
            </Link>
          );
        })}
      </nav>

      {onlyMine ? (
        // Jujur di layar, bukan diam-diam diabaikan: `50-FSD.md` §4.1 menyebut
        // penyaring Owner, sedangkan `42-API.md` §4 belum memuat parameternya.
        // Menyaring di klien akan salah menghitung `total` halaman.
        <p
          role="status"
          className="rounded-panel border border-line bg-surface-sunken px-3 py-2.5 text-13 text-text"
        >
          Penyaring Milik saya belum dapat dijalankan: kontrak GET /documents
          belum memuat parameter pemilik (Q-016). Daftar di bawah memuat
          seluruh dokumen dalam cakupan Anda.
        </p>
      ) : null}

      <form
        role="search"
        aria-label="Penyaring dokumen"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={applyFilters}
      >
        <div className="min-w-[220px] flex-1">
          <Field
            label="Cari"
            value={searchDraft}
            onChange={(event) => setSearchDraft(event.target.value)}
            placeholder="Judul atau nomor dokumen"
            maxLength={255}
          />
        </div>

        {/*
          `min-w-0` bukan hiasan: kolom penyaring adalah item flex, dan ukuran
          minimum otomatisnya adalah **min-content** anaknya. Untuk `<select>`,
          min-content ditentukan oleh **teks pilihan terpanjang** — teks milik
          data pengguna, bukan milik halaman. Satu project bernama panjang
          karena itu cukup untuk melebarkan baris penyaring melewati layar
          ponsel (terukur: `#penyaring-project-dokumen` 403px pada 375px,
          halaman menggulir mendatar 44px). Dengan `min-w-0` kolomnya boleh
          menyusut dan pilihan yang panjang dipotong browser, bukan halaman.
          Aturannya berlaku untuk **semua** kolom ber-`select`, karena nama
          anggota pun datang dari data (`51-UX.md` §2.1).
        */}
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="penyaring-status-dokumen"
            className="text-13 font-medium text-text-soft"
          >
            Status
          </label>
          <select
            id="penyaring-status-dokumen"
            value={status}
            onChange={(event) =>
              navigate({ status: event.target.value, page: null })
            }
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua kecuali terarsip</option>
            {documentStatuses.map((value) => (
              <option key={value} value={value}>
                {documentStatus(value).label}
              </option>
            ))}
          </select>
        </div>

        {canReadProjects ? (
          <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
            <label
              htmlFor="penyaring-project-dokumen"
              className="text-13 font-medium text-text-soft"
            >
              Project
            </label>
            <select
              id="penyaring-project-dokumen"
              value={projectId}
              onChange={(event) =>
                navigate({ project_id: event.target.value, page: null })
              }
              className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
            >
              <option value="">Semua project</option>
              {(projects.data?.items ?? []).map((project) => (
                <option key={project.id} value={project.id}>
                  {project.code} · {project.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}

        {/*
          Rentang `updated_at` adalah **satu kendali** dengan dua isian, bukan
          dua penyaring yang kebetulan bertetangga — pola yang sama dengan
          rentang tenggat Tasks (P-049): label yang terlihat adalah label
          **kelompok**, tiap isian bernama sendiri lewat `aria-label`, dan
          pemisah "sampai" dibaca mata saja. Keduanya menumpuk di dalam
          kelompok yang sama di layar sempit, karena berdampingan menuntut
          ~400px dan di 375px itu menggulirkan halaman (terukur, §3.14f).
        */}
        <div className="flex flex-col gap-1.5">
          <span
            id="penyaring-rentang-pembaruan"
            className="text-13 font-medium text-text-soft"
          >
            Rentang pembaruan
          </span>
          <div
            role="group"
            aria-labelledby="penyaring-rentang-pembaruan"
            className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-2"
          >
            <input
              id="penyaring-pembaruan-dari"
              type="datetime-local"
              aria-label="Pembaruan dari"
              value={updatedFromDraft}
              onChange={(event) => setUpdatedFromDraft(event.target.value)}
              aria-invalid={rangeErrors.updated_from ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.updated_from ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
            {/* Pemisah yang dibaca mata, bukan alat bantu: kedua isian sudah
                bernama "Pembaruan dari" dan "Pembaruan sampai". */}
            <span aria-hidden="true" className="text-13 text-text-muted">
              sampai
            </span>
            <input
              id="penyaring-pembaruan-sampai"
              type="datetime-local"
              aria-label="Pembaruan sampai"
              value={updatedToDraft}
              onChange={(event) => setUpdatedToDraft(event.target.value)}
              aria-invalid={rangeErrors.updated_to ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.updated_to ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
          </div>
        </div>

        <Button type="submit">Cari</Button>
        <Button type="button" variant="secondary" onClick={applyFilters}>
          Terapkan rentang
        </Button>
        {filtered ? (
          <Button variant="quiet" onClick={clearFilters}>
            Bersihkan penyaring
          </Button>
        ) : null}
      </form>

      {projects.isError ? (
        <p role="status" className="text-12 text-text-muted">
          Daftar project tidak dapat dimuat, sehingga penyaring Project hanya
          berisi Semua project. Penyaring lain tetap bekerja.
        </p>
      ) : null}

      {Object.keys(rangeErrors).length > 0 ? (
        <p role="alert" className="text-12 text-danger">
          {rangeErrors.updated_from ?? rangeErrors.updated_to}
        </p>
      ) : null}

      <DataTable
        caption="Daftar dokumen pada halaman ini"
        columns={columns}
        rows={rows}
        rowKey={(row) => row.id}
        tone={(row) => documentStatus(row.status).tone}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={(next) => navigate({ page: String(next) })}
        emptyState={
          filtered ? (
            <EmptyState
              title="Tidak ada dokumen yang cocok"
              description="Penyaring yang sedang aktif tidak menyisakan satu dokumen pun. Cakupan data tetap berlaku: dokumen di luar keanggotaan project Anda memang tidak pernah dikirim server."
              action={
                <Button variant="quiet" onClick={clearFilters}>
                  Bersihkan penyaring
                </Button>
              }
            />
          ) : (
            <EmptyState
              title="Belum ada dokumen"
              description={
                canCreate
                  ? "Unggah dokumen pertama. Nomor dokumen dibangkitkan server dengan format {KODE-PROJECT}-{NNN}, jadi Anda tidak mengetiknya."
                  : "Belum ada dokumen pada project yang Anda ikuti. Dokumen dibuat Contributor ke atas."
              }
              action={
                canCreate ? (
                  <Button
                    variant="primary"
                    onClick={() => navigate({ upload: "1" })}
                  >
                    Unggah dokumen
                  </Button>
                ) : null
              }
            />
          )
        }
      />

      {openUpload ? (
        <CreateDocumentDialog
          onClose={() => navigate({ upload: null })}
          onCreated={(documentId) => {
            navigate({ upload: null });
            navigateTo(`/documents/${documentId}`);
          }}
        />
      ) : null}
    </div>
  );
}
