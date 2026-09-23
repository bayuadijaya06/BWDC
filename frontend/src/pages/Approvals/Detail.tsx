import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";

import { Button } from "@/components/common/Button";
import { Field } from "@/components/common/Field";
import { StatusBadge } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useActWorkflowInstance, useWorkflowInstance } from "@/queries/workflows";
import { ApiError } from "@/services/http";
import { EMPTY_VALUE, formatTimestamp } from "@/utils/format";

export function ApprovalDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const query = useWorkflowInstance(id);
  const act = useActWorkflowInstance();

  const [comment, setComment] = useState("");
  const [error, setError] = useState<string | null>(null);

  const instance = query.data;

  async function handleAction(action: "approve" | "reject" | "request_revision") {
    setError(null);
    try {
      await act.mutateAsync({
        id,
        input: {
          action,
          comment: comment.trim() || undefined,
          version: instance?.version,
        },
      });
      setComment("");
    } catch (err) {
      if (err instanceof ApiError) {
        // 409 WORKFLOW_CONFLICT menampilkan details
        if (err.code === "WORKFLOW_CONFLICT" || err.code === "CONFLICT") {
          const details = (err as unknown as { details?: unknown }).details;
          setError(
            `${err.message}${details ? ` - ${JSON.stringify(details)}` : ""}. Muat ulang untuk melihat keadaan terbaru.`,
          );
          void query.refetch();
        } else {
          setError(err.message);
        }
      } else {
        setError(err instanceof Error ? err.message : "Gagal menjalankan aksi.");
      }
    }
  }

  if (query.isPending) {
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Approval Detail" description="Memuat instance…" />
        <p role="status" className="text-13 text-text-muted">
          Memuat…
        </p>
      </div>
    );
  }

  if (query.error) {
    const apiErr = query.error instanceof ApiError ? query.error : null;
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Approval Detail" description="Gagal memuat instance." />
        <p role="alert" className="rounded-panel border border-danger/30 bg-surface-raised px-3 py-2 text-13 text-danger">
          {apiErr ? apiErr.message : "Gagal memuat instance."}
        </p>
        <Button variant="quiet" onClick={() => void query.refetch()}>
          Muat ulang
        </Button>
      </div>
    );
  }

  if (!instance) {
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Approval Detail" description="Instance tidak ditemukan." />
        <p className="text-13 text-text-muted">Instance tidak ada atau di luar cakupan Anda (404).</p>
        <Link to="/approvals" className="text-13 text-accent underline">
          Kembali ke Approvals
        </Link>
      </div>
    );
  }

  const isRunning = instance.status === "running";
  const isRevisionRequired = instance.document_status === "revision_required";

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={`Approval ${instance.document_number ?? instance.document_id.slice(0, 8)}`}
        description={`${instance.document_title ?? EMPTY_VALUE} - ${instance.project_name ?? EMPTY_VALUE}`}
        actions={
          <Button variant="quiet" onClick={() => navigate("/approvals")}>
            Kembali
          </Button>
        }
      />

      <div className="grid gap-4 md:grid-cols-2">
        <div className="rounded-panel border border-line bg-surface-raised p-4">
          <h2 className="text-13 font-medium text-text">Instance</h2>
          <dl className="mt-3 grid gap-2 text-13">
            <div className="flex justify-between gap-4">
              <dt className="text-text-muted">Status</dt>
              <dd>
                <StatusBadge
                  presentation={
                    {
                      label:
                        instance.status === "running"
                          ? "Pending"
                          : instance.status === "completed"
                            ? "Approved"
                            : "Rejected",
                      tone:
                        instance.status === "running"
                          ? "info"
                          : instance.status === "completed"
                            ? "success"
                            : "danger",
                    } as never
                  }
                />
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-text-muted">Dokumen</dt>
              <dd>
                <Link
                  to={`/documents/${instance.document_id}`}
                  className="inline rounded-control text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
                >
                  {instance.document_number ?? instance.document_id}
                </Link>
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-text-muted">Status dokumen</dt>
              <dd className="font-medium text-text">{instance.document_status ?? EMPTY_VALUE}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-text-muted">Step</dt>
              <dd className="text-text">
                {instance.current_step_name} (#{instance.current_step})
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-text-muted">Tenggat step</dt>
              <dd className="text-text">
                {instance.current_step_deadline
                  ? formatTimestamp(instance.current_step_deadline)
                  : EMPTY_VALUE}
                {instance.is_overdue ? (
                  <span className="ml-2 text-12 font-medium text-danger">Overdue</span>
                ) : null}
              </dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-text-muted">Version</dt>
              <dd className="font-mono text-12 text-text-soft">{instance.version}</dd>
            </div>
            <div className="flex justify-between gap-4">
              <dt className="text-text-muted">Dibuat</dt>
              <dd className="text-text-soft">{formatTimestamp(instance.created_at)}</dd>
            </div>
          </dl>
        </div>

        <div className="rounded-panel border border-line bg-surface-raised p-4">
          <h2 className="text-13 font-medium text-text">Aksi</h2>
          {isRevisionRequired ? (
            <p role="status" className="mt-3 rounded-control border border-line bg-surface-sunken px-3 py-2 text-13 text-text">
              Jeda revisi: dokumen berstatus <strong>revision_required</strong>. Tidak ada aksi yang dapat
              dijalankan sampai owner mengunggah versi baru dan melakukan re-submit (`POST
              /workflows/instances/:id/resubmit`) - instance tetap `running`.
            </p>
          ) : !isRunning ? (
            <p className="mt-3 text-13 text-text-muted">
              Instance sudah {instance.status}. Tidak ada aksi yang dapat dijalankan.
            </p>
          ) : (
            <div className="mt-3 flex flex-col gap-3">
              <Field
                label="Catatan (opsional)"
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder="Alasan atau komentar untuk aksi ini"
                maxLength={2000}
              />
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="primary"
                  pending={act.isPending}
                  onClick={() => void handleAction("approve")}
                >
                  Approve
                </Button>
                <Button
                  variant="secondary"
                  pending={act.isPending}
                  onClick={() => void handleAction("request_revision")}
                >
                  Request Revision
                </Button>
                <Button
                  variant="danger"
                  pending={act.isPending}
                  onClick={() => void handleAction("reject")}
                >
                  Reject
                </Button>
              </div>
              <p className="text-12 text-text-muted">
                Aksi membutuhkan <strong>role penanggung jawab step</strong> dan izin
                `workflow_instance:approve/reject/request_revision`. Konflik versi (`409
                WORKFLOW_CONFLICT`) berarti instance sudah berubah - muat ulang dan coba lagi.
              </p>
            </div>
          )}
          {error ? (
            <p role="alert" className="mt-3 text-12 text-danger">
              {error}
            </p>
          ) : null}
          {act.isSuccess ? (
            <p role="status" className="mt-3 text-12 text-status-approved-ink">
              Aksi berhasil - instance diperbarui.
            </p>
          ) : null}
        </div>
      </div>

      <div className="rounded-panel border border-line bg-surface-raised p-4">
        <h2 className="text-13 font-medium text-text">Riwayat Aksi</h2>
        {instance.actions && instance.actions.length > 0 ? (
          <ol className="mt-3 flex flex-col gap-3">
            {instance.actions.map((a) => (
              <li
                key={a.id}
                className="flex flex-col gap-1 border-l-2 border-line-strong pl-3"
              >
                <span className="text-13 font-medium text-text">
                  {a.step_name} - {a.action}
                </span>
                <span className="text-12 text-text-muted">
                  {a.actor_username ?? a.actor_id} · {formatTimestamp(a.created_at)}
                </span>
                {a.comment ? (
                  <span className="text-13 text-text">{a.comment}</span>
                ) : null}
              </li>
            ))}
          </ol>
        ) : (
          <p className="mt-3 text-13 text-text-muted">Belum ada aksi pada instance ini.</p>
        )}
      </div>
    </div>
  );
}
