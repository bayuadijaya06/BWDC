import { useState, type FormEvent } from "react";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { Field } from "@/components/common/Field";
import { EmptyState } from "@/components/common/States";
import { PageHeader } from "@/components/layout/PageHeader";
import { useAuditList } from "@/queries/audit";
import { ApiError } from "@/services/http";
import type { AuditLog } from "@/services/audit";
import { formatTimestamp } from "@/utils/format";

/**
 * Audit Log Reader (`50-FSD.md` §10.6 menunjuk `44-SECURITY.md` §2.5/§4,
 * endpoint `GET /audit` di `42-API.md` §9, T-106).
 *
 * Yang dipegang halaman ini:
 *
 * 1. **Semua filter opsional dan digabung** — actor, aksi, entitas, tanggal.
 *    Aksi/entitas berupa isian teks bebas, BUKAN dropdown kosakata karangan:
 *    kontrak menetapkan nilai tak dikenal mengembalikan hasil kosong, bukan
 *    galat, supaya filter UI tidak pernah gagal.
 * 2. **Tanggal `type=date`** (`YYYY-MM-DD` sesuai contoh kontrak) — tanpa
 *    konverter zona waktu, karena kontrak menerima bentuk tanggal polos.
 * 3. **Hanya membaca.** Audit bersifat append-only (FR-AUDIT-03): tidak ada
 *    aksi ubah/hapus di halaman ini.
 */

const auditColumns: DataTableColumn<AuditLog>[] = [
  {
    key: "created_at",
    header: "Waktu",
    width: "170px",
    render: (row) => formatTimestamp(row.created_at),
  },
  {
    key: "actor",
    header: "Aktor",
    width: "130px",
    render: (row) => row.actor_name,
  },
  {
    key: "action",
    header: "Aksi",
    width: "220px",
    render: (row) => (
      <span className="font-mono text-12 text-text">{row.action}</span>
    ),
  },
  {
    key: "entity",
    header: "Entitas",
    width: "220px",
    render: (row) => (
      <span className="flex flex-col">
        <span className="font-mono text-12 text-text">{row.entity}</span>
        <span className="font-mono text-12 text-text-muted">
          {row.entity_id}
        </span>
      </span>
    ),
  },
  {
    key: "description",
    header: "Deskripsi",
    render: (row) => row.description,
  },
];

export function AuditPage() {
  const [actorId, setActorId] = useState("");
  const [action, setAction] = useState("");
  const [entity, setEntity] = useState("");
  const [entityId, setEntityId] = useState("");
  const [projectId, setProjectId] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [page, setPage] = useState(1);
  const [drafts, setDrafts] = useState({
    actor_id: "",
    action: "",
    entity: "",
    entity_id: "",
    project_id: "",
    date_from: "",
    date_to: "",
  });

  const query = useAuditList({
    page,
    limit: 50,
    actor_id: actorId,
    action,
    entity,
    entity_id: entityId,
    project_id: projectId,
    date_from: dateFrom,
    date_to: dateTo,
  });
  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];
  const filtered =
    actorId !== "" ||
    action.trim() !== "" ||
    entity.trim() !== "" ||
    entityId.trim() !== "" ||
    projectId.trim() !== "" ||
    dateFrom !== "" ||
    dateTo !== "";

  function setDraft(key: keyof typeof drafts, value: string) {
    setDrafts((current) => ({ ...current, [key]: value }));
  }

  function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setActorId(drafts.actor_id.trim());
    setAction(drafts.action.trim());
    setEntity(drafts.entity.trim());
    setEntityId(drafts.entity_id.trim());
    setProjectId(drafts.project_id.trim());
    setDateFrom(drafts.date_from);
    setDateTo(drafts.date_to);
    setPage(1);
  }

  function clearFilters() {
    setActorId("");
    setAction("");
    setEntity("");
    setEntityId("");
    setProjectId("");
    setDateFrom("");
    setDateTo("");
    setPage(1);
    setDrafts({
      actor_id: "",
      action: "",
      entity: "",
      entity_id: "",
      project_id: "",
      date_from: "",
      date_to: "",
    });
  }

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Audit log"
        description={
          meta
            ? `${meta.total} entri jejak audit, terbaru dulu. Jejak ini append-only: halaman ini hanya membaca.`
            : "Jejak audit sistem, terbaru dulu."
        }
      />

      <form
        role="search"
        aria-label="Penyaring audit"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={applyFilters}
      >
        <div className="min-w-[220px] flex-1">
          <Field
            label="Aktor (UUID)"
            value={drafts.actor_id}
            onChange={(event) => setDraft("actor_id", event.target.value)}
            placeholder="UUID pembuat aksi"
            maxLength={36}
          />
        </div>
        <div className="min-w-[180px] flex-1">
          <Field
            label="Aksi"
            value={drafts.action}
            onChange={(event) => setDraft("action", event.target.value)}
            placeholder="mis. DOCUMENT_CREATED"
            maxLength={100}
          />
        </div>
        <div className="min-w-[180px] flex-1">
          <Field
            label="Entitas"
            value={drafts.entity}
            onChange={(event) => setDraft("entity", event.target.value)}
            placeholder="mis. document"
            maxLength={100}
          />
        </div>
        <div className="min-w-[220px] flex-1">
          <Field
            label="ID entitas"
            value={drafts.entity_id}
            onChange={(event) => setDraft("entity_id", event.target.value)}
            placeholder="UUID atau nomor dokumen"
            maxLength={255}
          />
        </div>
        <div className="min-w-[220px] flex-1">
          <Field
            label="Project (UUID)"
            value={drafts.project_id}
            onChange={(event) => setDraft("project_id", event.target.value)}
            placeholder="UUID project"
            maxLength={36}
          />
        </div>
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="audit-tanggal-dari"
            className="text-13 font-medium text-text-soft"
          >
            Tanggal dari
          </label>
          <input
            id="audit-tanggal-dari"
            type="date"
            value={drafts.date_from}
            onChange={(event) => setDraft("date_from", event.target.value)}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          />
        </div>
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="audit-tanggal-sampai"
            className="text-13 font-medium text-text-soft"
          >
            Tanggal sampai
          </label>
          <input
            id="audit-tanggal-sampai"
            type="date"
            value={drafts.date_to}
            onChange={(event) => setDraft("date_to", event.target.value)}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          />
        </div>
        <Button type="submit">Terapkan</Button>
        {filtered ? (
          <Button variant="quiet" onClick={clearFilters}>
            Bersihkan penyaring
          </Button>
        ) : null}
      </form>

      <DataTable
        caption="Jejak audit pada halaman ini"
        columns={auditColumns}
        rows={rows}
        rowKey={(row) => row.id}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={setPage}
        emptyState={
          <EmptyState
            title={filtered ? "Tidak ada entri yang cocok" : "Belum ada jejak"}
            description="Penyaring yang aktif tidak menyisakan satu entri pun, atau belum ada aksi teraudit."
          />
        }
      />
    </div>
  );
}
