import { useMemo, useState, type FormEvent } from "react";

import { Button } from "@/components/common/Button";
import { TextareaField } from "@/components/common/Field";
import { Panel } from "@/components/common/Panel";
import { EmptyState, ErrorMessage, ErrorState } from "@/components/common/States";
import {
  useCreateComment,
  useDeleteComment,
  useCommentList,
  useUpdateComment,
} from "@/queries/comments";
import {
  validateCommentContent,
  type CommentEntityType,
  type CommentRecord,
} from "@/services/comments";
import { ApiError } from "@/services/http";
import { useAuthStore } from "@/store/auth";
import { formatTimestamp } from "@/utils/format";

/**
 * Utas komentar satu entitas (`50-FSD.md` §7, `42-API.md` §7).
 *
 * Daftar kronologis datar dari server; balasan (`parent_id`, ADR-0032)
 * ditandai "Membalas {penulis}" yang dipetakan dari daftar yang sama —
 * tanpa pengurutan ulang per cabang dan tanpa endpoint tambahan. Induk yang
 * tidak termuat (paginasi) ditandai generik "Membalas komentar lain", bukan
 * disembunyikan: barisnya tetap ada dan isinya tetap terbaca.
 */
export function CommentThread({
  entityType,
  entityId,
}: {
  entityType: CommentEntityType;
  entityId: string;
}) {
  const canCreate = useAuthStore((state) => state.has("comment:create"));
  // Edit/hapus dijaga `comment:read` di route (matriks tidak punya
  // `comment:update`/`comment:delete`); kepemilikannya ditegakkan server,
  // klien hanya menyembunyikan tombol milik orang lain.
  const canEdit = useAuthStore((state) => state.has("comment:read"));
  const currentUserId = useAuthStore((state) => state.profile?.id ?? "");

  const list = useCommentList({ entity_type: entityType, entity_id: entityId, limit: 100 });
  const create = useCreateComment();
  const update = useUpdateComment();
  const remove = useDeleteComment();

  const [draft, setDraft] = useState("");
  const [draftError, setDraftError] = useState<string | null>(null);
  const [replyTo, setReplyTo] = useState<Pick<CommentRecord, "id" | "created_by_username"> | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editDraft, setEditDraft] = useState("");
  const [editError, setEditError] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const items = useMemo(() => list.data?.items ?? [], [list.data]);
  const byId = useMemo(() => new Map(items.map((item) => [item.id, item])), [items]);
  const serverError = [create.error, update.error, remove.error].find(
    (error) => error instanceof ApiError,
  ) as ApiError | undefined;

  function parentLabel(comment: CommentRecord): string | null {
    if (!comment.parent_id) return null;
    const parent = byId.get(comment.parent_id);
    const name = parent?.created_by_username?.trim();
    return name ? `Membalas ${name}` : "Membalas komentar lain";
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const invalid = validateCommentContent(draft);
    setDraftError(invalid);
    if (invalid !== null) return;
    create.mutate(
      {
        entity_type: entityType,
        entity_id: entityId,
        content: draft.trim(),
        ...(replyTo === null ? {} : { parent_id: replyTo.id }),
      },
      {
        onSuccess: () => {
          setDraft("");
          setDraftError(null);
          setReplyTo(null);
        },
        onError: (error) => {
          if (error instanceof ApiError) setDraftError(error.fieldErrors.content ?? null);
        },
      },
    );
  }

  function startEdit(comment: CommentRecord) {
    setEditingId(comment.id);
    setEditDraft(comment.content);
    setEditError(null);
    setConfirmDeleteId(null);
  }

  function saveEdit(id: string) {
    const invalid = validateCommentContent(editDraft);
    setEditError(invalid);
    if (invalid !== null) return;
    update.mutate(
      { id, content: editDraft.trim() },
      {
        onSuccess: () => {
          setEditingId(null);
          setEditDraft("");
          setEditError(null);
        },
        onError: (error) => {
          if (error instanceof ApiError) setEditError(error.fieldErrors.content ?? null);
        },
      },
    );
  }

  function askDelete(id: string) {
    setConfirmDeleteId(id);
    setEditingId(null);
  }

  function confirmDelete(id: string) {
    remove.mutate(id, {
      onSuccess: () => setConfirmDeleteId(null),
    });
  }

  return (
    <Panel
      title="Komentar"
      note="Urutan kronologis dari server; balasan ditandai, bukan disarang ulang."
    >
      {list.isPending ? (
        <p role="status" className="text-13 text-text-muted">
          Memuat komentar...
        </p>
      ) : list.error instanceof ApiError ? (
        <ErrorState error={list.error} onRetry={() => void list.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState
          title="Belum ada komentar"
          description="Mulai diskusi pada halaman ini. Komentar tercatat di audit."
        />
      ) : (
        <ol className="flex flex-col gap-4">
          {items.map((comment) => {
            const marker = parentLabel(comment);
            const own = canEdit && comment.created_by_id === currentUserId && currentUserId !== "";
            const editing = editingId === comment.id;
            const confirming = confirmDeleteId === comment.id;
            const initial = (comment.created_by_username?.trim()[0] ?? "?").toUpperCase();
            return (
              <li key={comment.id} className="flex gap-2.5">
                <span
                  aria-hidden="true"
                  className="tap-target inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-line-strong bg-surface-sunken text-13 font-medium text-text-soft"
                >
                  {initial}
                </span>
                <div className="flex min-w-0 flex-1 flex-col gap-1">
                  <p className="flex flex-wrap items-baseline gap-x-2 text-13">
                    <span className="font-medium text-text">
                      {comment.created_by_username}
                    </span>
                    <span className="text-12 text-text-muted">
                      {formatTimestamp(comment.created_at)}
                    </span>
                    {marker !== null ? (
                      <span className="text-12 text-text-muted">· {marker}</span>
                    ) : null}
                  </p>
                  {editing ? (
                    <div className="flex flex-col gap-1.5">
                      <TextareaField
                        label={`Ubah komentar ${comment.created_by_username}`}
                        rows={2}
                        value={editDraft}
                        onChange={(event) => setEditDraft(event.target.value)}
                        error={editError}
                      />
                      <div className="flex gap-1.5">
                        <Button
                          type="button"
                          variant="primary"
                          pending={update.isPending}
                          onClick={() => saveEdit(comment.id)}
                        >
                          Simpan
                        </Button>
                        <Button
                          type="button"
                          variant="quiet"
                          onClick={() => setEditingId(null)}
                          disabled={update.isPending}
                        >
                          Batal
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <p className="max-w-prose whitespace-pre-wrap text-13 text-text">
                      {comment.content}
                    </p>
                  )}
                  {(own && !editing) || canCreate ? (
                    <div className="flex gap-1.5">
                      {canCreate ? (
                        <button
                          type="button"
                          onClick={() =>
                            setReplyTo({ id: comment.id, created_by_username: comment.created_by_username ?? "" })
                          }
                          className="text-12 text-text-soft underline decoration-line-strong underline-offset-2 hover:text-text"
                        >
                          Balas
                        </button>
                      ) : null}
                      {own && !editing ? (
                        <>
                          <button
                            type="button"
                            onClick={() => startEdit(comment)}
                            className="text-12 text-text-soft underline decoration-line-strong underline-offset-2 hover:text-text"
                          >
                            Ubah
                          </button>
                          {confirming ? (
                            <>
                              <button
                                type="button"
                                onClick={() => void confirmDelete(comment.id)}
                                className="text-12 font-medium text-danger hover:underline"
                              >
                                Ya, hapus
                              </button>
                              <button
                                type="button"
                                onClick={() => setConfirmDeleteId(null)}
                                className="text-12 text-text-soft underline decoration-line-strong underline-offset-2 hover:text-text"
                              >
                                Batal
                              </button>
                            </>
                          ) : (
                            <button
                              type="button"
                              onClick={() => askDelete(comment.id)}
                              className="text-12 text-text-soft underline decoration-line-strong underline-offset-2 hover:text-danger"
                            >
                              Hapus
                            </button>
                          )}
                        </>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              </li>
            );
          })}
        </ol>
      )}

      {serverError !== undefined && serverError.status !== 422 ? (
        <ErrorMessage error={serverError} />
      ) : null}

      {canCreate ? (
        <form onSubmit={submit} className="flex flex-col gap-2 pt-4" noValidate>
          {replyTo !== null ? (
            <p className="flex flex-wrap items-center gap-2 text-13 text-text">
              <span>
                Membalas <strong>{replyTo.created_by_username}</strong>
              </span>
              <button
                type="button"
                onClick={() => setReplyTo(null)}
                className="text-12 text-text-soft underline decoration-line-strong underline-offset-2 hover:text-text"
              >
                Batalkan balasan
              </button>
            </p>
          ) : null}
          <TextareaField
            label={replyTo !== null ? "Tulis balasan" : "Tulis komentar"}
            rows={2}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            error={draftError}
            placeholder="Tulis dengan jelas; komentar tercatat di audit."
          />
          <div>
            <Button type="submit" variant="primary" pending={create.isPending}>
              {replyTo !== null ? "Kirim balasan" : "Kirim komentar"}
            </Button>
          </div>
        </form>
      ) : (
        <p className="pt-4 text-13 text-text-muted">
          Anda tidak memiliki izin berkomentar.
        </p>
      )}
    </Panel>
  );
}
