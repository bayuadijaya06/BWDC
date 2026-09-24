import { useState, type ReactNode } from "react";
import { Link, useNavigate, useParams } from "react-router";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { Dialog } from "@/components/common/Dialog";
import { TextareaField } from "@/components/common/Field";
import { SelectField } from "@/components/common/SelectField";
import { Panel } from "@/components/common/Panel";
import { EmptyState, ErrorState, TableSkeleton } from "@/components/common/States";
import { PageHeader } from "@/components/layout/PageHeader";
import {
  useArchiveDocument,
  useDocument,
  useDocumentVersions,
} from "@/queries/documents";
import {
  useResubmitWorkflowInstance,
  useSubmitWorkflowInstance,
  useWorkflowDefinitions,
} from "@/queries/workflows";
import {
  downloadDocumentVersion,
  type DocumentVersion,
} from "@/services/documents";
import { ApiError } from "@/services/http";
import { useAuthStore } from "@/store/auth";
import { documentStatus } from "@/types/status";
import { EMPTY_VALUE, formatFileSize, formatTimestamp } from "@/utils/format";
import { saveBlob } from "@/utils/download";

import { UploadVersionDialog } from "./UploadVersionDialog";

/**
 * Bagian `50-FSD.md` §4.3 yang **belum** dibangun. Ditulis sebagai data, bukan
 * komponen setengah jadi, dan alasannya menyebut keadaan sebenarnya.
 *
 * Workflow tidak ada di sini lagi: modulnya hidup dan panelnya dibangun di
 * bawah (`T-086` backend, submit/resubmit di halaman ini). Activity juga bukan
 * "endpoint belum ada" — `GET /audit` hidup, tetapi izin `audit:read` hanya
 * milik Administrator dan tampilan aktivitas per dokumen belum dibangun.
 */
const pendingSections: { label: string; reason: string; reference: string }[] = [
  {
    label: "Comments",
    reason:
      "Endpoint komentar sudah hidup untuk entitas document (bentuk kueri ?entity_type=&entity_id=), tetapi antarmuka utas komentar belum dibangun.",
    reference: "docs/design/42-API.md §7, docs/design/50-FSD.md §7",
  },
  {
    label: "Activity",
    reason:
      "Endpoint GET /audit sudah hidup dan menerima penyaring entity, tetapi izin audit:read hanya milik Administrator dan tampilan aktivitas per dokumen belum dibangun.",
    reference: "docs/design/42-API.md §9, docs/design/44-SECURITY.md §3.1",
  },
  {
    label: "Related Tasks",
    reason:
      "Kontrak §6 menyaring task menurut project, bukan menurut dokumen, jadi daftar tugas yang terkait satu dokumen belum dapat diminta dari server.",
    reference: "docs/design/42-API.md §6, docs/design/50-FSD.md §4.3",
  },
];

/**
 * Document Detail (`50-FSD.md` §4.3).
 *
 * Cakupan data berlaku di sini seperti pada daftar: dokumen di luar
 * keanggotaan project dibalas `404`, jadi halaman ini menampilkan "tidak
 * ditemukan" yang sama untuk dokumen yang tidak ada dan dokumen milik orang
 * lain — memang itu yang dijanjikan `42-API.md` §4.
 *
 * Dokumen terarsip tetap dapat dibaca dan diunduh (ADR-0019); yang hilang
 * hanyalah aksi yang mengubahnya: unggah versi dan arsip ulang.
 */
export function DocumentDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const canDownload = useAuthStore((state) =>
    state.has("document_version:download"),
  );
  const canUpload = useAuthStore((state) =>
    state.has("document_version:upload"),
  );
  const canArchive = useAuthStore((state) => state.has("document:update"));
  const canSubmit = useAuthStore((state) =>
    state.has("workflow_instance:submit"),
  );

  const query = useDocument(id);
  const versions = useDocumentVersions(id);
  const archive = useArchiveDocument();
  const submit = useSubmitWorkflowInstance();
  const resubmit = useResubmitWorkflowInstance();

  const [downloading, setDownloading] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState<ApiError | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [openUpload, setOpenUpload] = useState(false);
  const [openArchive, setOpenArchive] = useState(false);
  const [archiveReason, setArchiveReason] = useState("");
  const [openSubmit, setOpenSubmit] = useState(false);
  const [definitionId, setDefinitionId] = useState("");
  const [workflowError, setWorkflowError] = useState<string | null>(null);

  const definitions = useWorkflowDefinitions({
    enabled: openSubmit && canSubmit,
  });

  if (query.isPending) {
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Dokumen" />
        <TableSkeleton rows={4} columns={3} />
      </div>
    );
  }

  const detail = query.data;
  if (query.error instanceof ApiError || detail === undefined) {
    const error =
      query.error instanceof ApiError
        ? query.error
        : ApiError.network("detail dokumen tidak tersedia");
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Dokumen" />
        <ErrorState error={error} onRetry={() => void query.refetch()} />
        <p className="text-13 text-text-muted">
          Dokumen di luar keanggotaan project Anda dijawab server sebagai tidak
          ditemukan, jadi halaman ini tidak membedakan kedua sebabnya.
        </p>
        <div>
          <Link
            to="/documents"
            className="text-13 text-text underline decoration-line-strong underline-offset-2"
          >
            Kembali ke daftar dokumen
          </Link>
        </div>
      </div>
    );
  }

  const { document, current_version: currentVersion } = detail;
  const archived = document.status === "archived";
  const versionRows = versions.data ?? [];
  const isDraft = document.status === "draft";
  const isRevisionRequired = document.status === "revision_required";
  const instanceId = document.workflow_instance_id ?? null;

  async function handleSubmit() {
    setWorkflowError(null);
    if (definitionId === "") {
      setWorkflowError("Pilih alur review lebih dulu.");
      return;
    }
    try {
      const instance = await submit.mutateAsync({
        document_id: document.id,
        workflow_definition_id: definitionId,
      });
      setOpenSubmit(false);
      setDefinitionId("");
      navigate(`/approvals/${instance.id}`);
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.status === 409) {
          setWorkflowError(`${error.message}. Muat ulang untuk melihat keadaan terbaru.`);
          void query.refetch();
        } else if (error.status === 403) {
          setWorkflowError("Submit memerlukan izin workflow_instance:submit (Contributor ke atas).");
        } else if (error.status === 422) {
          const fields = error.fieldErrors;
          const first = Object.values(fields)[0];
          setWorkflowError(first ?? error.message);
        } else {
          setWorkflowError(error.message);
        }
      } else {
        setWorkflowError("Gagal memulai review.");
      }
    }
  }

  async function handleResubmit() {
    setWorkflowError(null);
    if (instanceId === null) return;
    try {
      await resubmit.mutateAsync({ id: instanceId });
      setNotice("Re-submit berhasil - review dilanjutkan pada instance yang sama.");
      void query.refetch();
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.code === "WORKFLOW_CONFLICT" || error.code === "CONFLICT") {
          const details = (errDetails(error));
          setWorkflowError(
            `${error.message}${details ? ` - ${details}` : ""}. Muat ulang untuk melihat keadaan terbaru.`,
          );
          void query.refetch();
        } else if (error.status === 403) {
          setWorkflowError("Re-submit memerlukan izin workflow_instance:submit (Contributor ke atas).");
        } else {
          setWorkflowError(error.message);
        }
      } else {
        setWorkflowError("Gagal melakukan re-submit.");
      }
    }
  }

  function errDetails(error: ApiError): string | null {
    const details = (error as unknown as { details?: unknown }).details;
    return details === undefined ? null : JSON.stringify(details);
  }

  async function download(version: DocumentVersion) {
    setDownloading(version.id);
    setDownloadError(null);
    setNotice(null);
    try {
      const file = await downloadDocumentVersion(
        document.id,
        version.id,
        version.original_name,
      );
      saveBlob(file.blob, file.filename);
      setNotice(`Berkas ${file.filename} mulai diunduh.`);
    } catch (error) {
      setDownloadError(error instanceof ApiError ? error : null);
      setNotice(null);
    } finally {
      setDownloading(null);
    }
  }

  const versionColumns: DataTableColumn<DocumentVersion>[] = [
    {
      key: "version",
      header: "Versi",
      width: "72px",
      render: (row) => (
        <span className="font-mono text-12 text-text">{row.version}</span>
      ),
    },
    {
      key: "file",
      header: "Berkas",
      render: (row) => (
        <span className="flex flex-col">
          <span className="text-text">{row.original_name}</span>
          <span className="text-12 text-text-muted">
            {formatFileSize(row.size)} · {row.mime_type}
          </span>
        </span>
      ),
    },
    {
      key: "checksum",
      header: "Checksum SHA-256",
      width: "270px",
      render: (row) => (
        <span className="font-mono text-12 break-all text-text-soft">
          {row.checksum}
        </span>
      ),
    },
    {
      key: "uploaded_by",
      header: "Diunggah oleh",
      width: "130px",
      render: (row) => row.uploaded_by_username ?? EMPTY_VALUE,
    },
    {
      key: "created_at",
      header: "Waktu",
      width: "170px",
      render: (row) => formatTimestamp(row.created_at),
    },
    {
      key: "revision_note",
      header: "Catatan revisi",
      render: (row) =>
        row.revision_note && row.revision_note.trim() !== "" ? (
          row.revision_note
        ) : (
          <span className="text-text-muted">Belum diisi</span>
        ),
    },
  ];

  const metadata: { label: string; value: ReactNode }[] = [
    {
      label: "Project",
      value: (
        <Link
          to={`/projects/${document.project_id}`}
          className="text-text underline decoration-line-strong underline-offset-2"
        >
          {document.project_name ?? document.project_code ?? EMPTY_VALUE}
        </Link>
      ),
    },
    { label: "Kategori", value: document.category_id ? (
        <Link
          to={`/documents?category_id=${encodeURIComponent(document.category_id)}`}
          className="text-text underline decoration-line-strong underline-offset-2"
        >
          {document.category_name ?? document.category_id}
        </Link>
      ) : (
        EMPTY_VALUE
      ) },
    { label: "Pemilik", value: document.owner_username ?? EMPTY_VALUE },
    { label: "Versi terakhir", value: document.latest_version ?? EMPTY_VALUE },
    {
      label: "Jumlah versi",
      value: String(document.current_version),
    },
    { label: "Dibuat", value: formatTimestamp(document.created_at) },
    { label: "Diperbarui", value: formatTimestamp(document.updated_at) },
    {
      label: "Diarsipkan",
      value: document.archived_at
        ? formatTimestamp(document.archived_at)
        : "Belum diarsipkan",
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={document.title}
        presentation={documentStatus(document.status)}
        description={
          <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span className="font-mono text-12 text-text-soft">
              {document.document_number}
            </span>
            <span aria-hidden="true">·</span>
            <span>
              {document.project_name ?? document.project_code ?? EMPTY_VALUE}
            </span>
            <span aria-hidden="true">·</span>
            <span>{documentStatus(document.status).label}</span>
          </span>
        }
        actions={
          <>
            <Link
              to="/documents"
              className="tap-target inline-flex items-center rounded-control border border-line-strong bg-surface-raised px-3 text-13 text-text hover:bg-surface-hover"
            >
              Daftar dokumen
            </Link>
            {canUpload && !archived ? (
              <Button variant="primary" onClick={() => setOpenUpload(true)}>
                Unggah versi
              </Button>
            ) : null}
            {canArchive && !archived ? (
              <Button
                variant="danger"
                onClick={() => {
                  setArchiveReason("");
                  setOpenArchive(true);
                }}
              >
                Arsipkan
              </Button>
            ) : null}
          </>
        }
      />

      {archived ? (
        <p
          role="status"
          className="rounded-panel border border-line bg-surface-sunken px-3 py-2.5 text-13 text-text"
        >
          Dokumen ini terarsip. Baris, versi, berkas, dan jejak auditnya tetap
          ada dan tetap dapat diunduh; yang ditolak server adalah unggahan versi
          baru dan pengarsipan ulang, dan dokumen terarsip tidak dapat
          di-un-archive (ADR-0019).
        </p>
      ) : null}

      {notice !== null ? (
        <p role="status" className="text-13 text-text">
          {notice}
        </p>
      ) : null}

      {downloadError !== null ? (
        <ErrorState error={downloadError} />
      ) : null}

      <Panel
        title="Metadata"
        note="Dikirim langsung oleh GET /documents/:id; tidak ada nilai yang dihitung ulang di klien."
      >
        <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-4">
          {metadata.map((entry) => (
            <div key={entry.label} className="flex flex-col gap-0.5">
              <dt className="index-label">{entry.label}</dt>
              <dd className="text-13 text-text">{entry.value}</dd>
            </div>
          ))}
        </dl>
        <div className="flex flex-col gap-0.5 pt-3.5">
          <span className="index-label">Deskripsi</span>
          <p className="max-w-prose text-13 text-text">
            {document.description.trim() === ""
              ? "Belum diisi."
              : document.description}
          </p>
        </div>
        <div className="flex flex-col gap-0.5 pt-3.5">
          <span className="index-label">Versi berjalan</span>
          <p className="text-13 text-text">
            {currentVersion === null
              ? "Dokumen ini belum punya unggahan. Dokumen tanpa versi tetap sah menurut kontrak, dan nomor dokumennya sudah final."
              : `${currentVersion.original_name} (${currentVersion.version}) diunggah ${formatTimestamp(currentVersion.created_at)}.`}
          </p>
        </div>
      </Panel>

      <Panel
        title="Versi"
        note="Terbaru lebih dulu, urutan dari server (FR-VER-04). Versi tidak pernah ditimpa atau dihapus (FR-VER-03)."
      >
        {versions.isPending ? (
          <TableSkeleton rows={2} columns={5} />
        ) : versions.error instanceof ApiError ? (
          <ErrorState
            error={versions.error}
            onRetry={() => void versions.refetch()}
          />
        ) : (
          <DataTable
            caption="Versi dokumen, terbaru lebih dulu"
            columns={versionColumns}
            rows={versionRows}
            rowKey={(row) => row.id}
            density="comfortable"
            actionsHeader="Unduh"
            actions={(row) =>
              canDownload ? (
                <Button
                  variant="secondary"
                  pending={downloading === row.id}
                  onClick={() => void download(row)}
                >
                  Unduh
                </Button>
              ) : null
            }
            emptyState={
              <EmptyState
                title="Belum ada versi"
                description={
                  canUpload
                    ? "Unggah berkas pertama lewat tombol Unggah versi; server akan menomorkannya 1.0."
                    : "Dokumen ini belum punya berkas, dan unggahan hanya dapat dilakukan Contributor ke atas."
                }
                action={
                  canUpload && !archived ? (
                    <Button
                      variant="primary"
                      onClick={() => setOpenUpload(true)}
                    >
                      Unggah versi
                    </Button>
                  ) : null
                }
              />
            }
          />
        )}
      </Panel>

      <Panel
        title="Workflow"
        note="Jalur review dokumen ini: submit pertama atau lanjutan sesudah revisi (ADR-0016)."
      >
        {archived ? (
          <p className="text-13 text-text-muted">
            Dokumen terarsip tidak dapat masuk atau melanjutkan review (`409`).
          </p>
        ) : isDraft && instanceId === null ? (
          <div className="flex flex-col gap-2">
            <p className="max-w-prose text-13 text-text-muted">
              Dokumen masih draft dan belum punya workflow instance. Memulai
              review membuat instance baru pada step 1.
            </p>
            {canSubmit ? (
              <div>
                <Button variant="primary" onClick={() => {
                  setDefinitionId("");
                  setWorkflowError(null);
                  setOpenSubmit(true);
                }}>
                  Submit for Review
                </Button>
              </div>
            ) : (
              <p className="text-12 text-text-muted">
                Submit memerlukan izin workflow_instance:submit (Contributor ke atas).
              </p>
            )}
          </div>
        ) : isRevisionRequired && instanceId !== null ? (
          <div className="flex flex-col gap-2">
            <p className="max-w-prose text-13 text-text-muted">
              Jeda revisi: instance tetap berjalan. Setelah versi baru diunggah,
              lanjutkan review pada instance yang sama.
            </p>
            <div className="flex flex-wrap items-center gap-2">
              {canSubmit ? (
                <Button
                  variant="primary"
                  pending={resubmit.isPending}
                  onClick={() => void handleResubmit()}
                >
                  Resubmit for Review
                </Button>
              ) : null}
              <Link
                to={`/approvals/${instanceId}`}
                className="tap-target inline-flex items-center rounded-control border border-line-strong bg-surface-raised px-3 text-13 text-text hover:bg-surface-hover"
              >
                Buka di Approvals
              </Link>
            </div>
            {!canSubmit ? (
              <p className="text-12 text-text-muted">
                Re-submit memerlukan izin workflow_instance:submit (Contributor ke atas).
              </p>
            ) : null}
          </div>
        ) : instanceId !== null ? (
          <div className="flex flex-col gap-2">
            <p className="max-w-prose text-13 text-text-muted">
              Review berjalan pada instance yang terikat dokumen ini
              {document.workflow_instance_status ? ` (status: ${document.workflow_instance_status})` : ""}.
            </p>
            <div>
              <Link
                to={`/approvals/${instanceId}`}
                className="tap-target inline-flex items-center rounded-control border border-line-strong bg-surface-raised px-3 text-13 text-text hover:bg-surface-hover"
              >
                Buka di Approvals
              </Link>
            </div>
          </div>
        ) : (
          <p className="text-13 text-text-muted">
            Belum ada workflow instance untuk dokumen ini.
          </p>
        )}
        {workflowError !== null && !openSubmit ? (
          <p role="alert" className="pt-2 text-12 text-danger">
            {workflowError}
          </p>
        ) : null}
      </Panel>

      <Panel
        title="Bagian lain halaman ini"
        note="Disebut 50-FSD.md §4.3 dan belum dibangun; alasannya per bagian, bukan satu kalimat umum."
      >
        <dl className="flex flex-col gap-3">
          {pendingSections.map((section) => (
            <div key={section.label} className="flex flex-col gap-0.5">
              <dt className="text-13 font-medium text-text">
                {section.label}{" "}
                <span className="font-normal text-text-muted">
                  belum dibangun
                </span>
              </dt>
              <dd className="max-w-prose text-13 text-text-muted">
                {section.reason}{" "}
                <span className="text-12">({section.reference})</span>
              </dd>
            </div>
          ))}
        </dl>
      </Panel>

      {openUpload ? (
        <UploadVersionDialog
          documentId={document.id}
          status={document.status}
          latestVersion={document.latest_version}
          onClose={() => setOpenUpload(false)}
          onUploaded={(version) => {
            setOpenUpload(false);
            setNotice(
              `Versi ${version.version} tersimpan sebagai ${version.original_name}.`,
            );
          }}
        />
      ) : null}

      {openArchive ? (
        <Dialog
          title="Arsipkan dokumen"
          description="Dokumen terarsip tetap tersimpan beserta seluruh versinya."
          onClose={archive.isPending ? () => {} : () => setOpenArchive(false)}
          footer={
            <>
              <Button
                variant="quiet"
                onClick={() => setOpenArchive(false)}
                disabled={archive.isPending}
              >
                Batal
              </Button>
              <Button
                variant="danger"
                pending={archive.isPending}
                onClick={() => {
                  archive.mutate(
                    {
                      id: document.id,
                      reason: archiveReason,
                    },
                    {
                      onSuccess: () => {
                        setOpenArchive(false);
                        setNotice(
                          `${document.document_number} terarsip. Versinya tetap dapat diunduh.`,
                        );
                      },
                    },
                  );
                }}
              >
                Arsipkan dokumen
              </Button>
            </>
          }
        >
          <div className="flex flex-col gap-3">
            <p className="text-13 text-text">
              <span className="font-mono text-12">
                {document.document_number}
              </span>{" "}
              akan berstatus <strong>Archived</strong> dan keluar dari daftar
              default. Tidak ada baris, versi, atau berkas yang dihapus.
            </p>
            <TextareaField
              label="Alasan"
              rows={2}
              value={archiveReason}
              onChange={(event) => setArchiveReason(event.target.value)}
              hint="Opsional; ikut tercatat di audit log."
            />
            {archive.error instanceof ApiError ? (
              <p role="alert" className="text-12 text-danger">
                {archive.error.message}
              </p>
            ) : null}
          </div>
        </Dialog>
      ) : null}

      {openSubmit ? (
        <Dialog
          title="Submit for Review"
          description="Memulai review membuat workflow instance baru pada step 1. Dokumen berpindah ke In Review."
          onClose={submit.isPending ? () => {} : () => setOpenSubmit(false)}
          footer={
            <>
              <Button
                variant="quiet"
                onClick={() => setOpenSubmit(false)}
                disabled={submit.isPending}
              >
                Batal
              </Button>
              <Button
                variant="primary"
                pending={submit.isPending}
                disabled={definitionId === ""}
                onClick={() => void handleSubmit()}
              >
                Mulai review
              </Button>
            </>
          }
        >
          <div className="flex flex-col gap-3">
            <SelectField
              label="Alur review"
              value={definitionId}
              onChange={(event) => setDefinitionId(event.target.value)}
              hint="Definisi tidak dapat diganti setelah instance dibuat."
            >
              <option value="">
                {definitions.isPending ? "Memuat alur…" : "Pilih alur"}
              </option>
              {(definitions.data ?? [])
                .filter((definition) => definition.is_active)
                .map((definition) => (
                  <option key={definition.id} value={definition.id}>
                    {definition.name}
                  </option>
                ))}
            </SelectField>
            {definitions.isError ? (
              <p className="text-12 text-text-muted">
                Daftar alur tidak dapat dimuat. Coba tutup lalu buka lagi dialog ini.
              </p>
            ) : null}
            {workflowError !== null ? (
              <p role="alert" className="text-12 text-danger">
                {workflowError}
              </p>
            ) : null}
          </div>
        </Dialog>
      ) : null}
    </div>
  );
}
