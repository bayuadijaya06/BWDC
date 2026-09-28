import { useState, type FormEvent } from "react";
import { useSearchParams } from "react-router";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { Dialog } from "@/components/common/Dialog";
import { Field } from "@/components/common/Field";
import { EmptyState, ErrorMessage } from "@/components/common/States";
import { PageHeader } from "@/components/layout/PageHeader";
import {
  useAdminOrganizations,
  useAdminRoles,
  useAdminUsers,
  useCreateAdminOrganization,
  useCreateAdminUser,
  useResetAdminPassword,
  useSetAdminUserRoles,
  useUpdateAdminOrganization,
  useUpdateAdminUser,
} from "@/queries/admin";
import {
  useCreateDocumentCategory,
  useDeleteDocumentCategory,
  useDocumentCategories,
  useUpdateDocumentCategory,
} from "@/queries/documents";
import { ApiError } from "@/services/http";
import type { AdminOrg, AdminRole, AdminUser } from "@/services/admin";
import type { DocumentCategory } from "@/services/documents";
import { useAuthStore } from "@/store/auth";

/**
 * Halaman Administration (`50-FSD.md` §10, T-105).
 *
 * Tab `?section=` di URL (dapat dibagikan). Bagian Users dan Roles berdiri di
 * sesi ini; Organizations, Categories, dan Settings menyusul (T-105 bagian
 * 2/2) — tabnya belum dirender, bukan dirender mati.
 *
 * Yang dipegang halaman ini:
 *
 * 1. **Satu endpoint� satu izin dari matriks** — tanpa pasangan karangan
 *    (ADR-0014): tulis user `user:update`, role `user_role:manage`, baca
 *    `user:read`/`role:read`. Tombol disembunyikan tanpa izinnya (pola
 *    `canCreate` halaman modul), bukan dinonaktifkan tanpa alasan.
 * 2. **Galat server dipetakan ke field** (`422 fieldErrors`) atau pesan umum
 *    (`409`) — pola dialog buat project (`T-053`).
 * 3. **Batas yang dinyatakan terbuka**: matriks izin per role belum punya
 *    endpoint (bagian Roles), dan reset password tanpa bell (Q-027) — keduanya
 *    tertulis di layar.
 */

const adminSections = [
  { id: "users", label: "Users" },
  { id: "roles", label: "Roles" },
  { id: "organizations", label: "Organizations" },
  { id: "categories", label: "Categories" },
  { id: "settings", label: "Settings" },
] as const;

type AdminSection = (typeof adminSections)[number]["id"];

const userColumns: DataTableColumn<AdminUser>[] = [
  {
    key: "username",
    header: "Username",
    render: (row) => (
      <span className="font-mono text-12 text-text">{row.username}</span>
    ),
  },
  {
    key: "email",
    header: "Email",
    render: (row) => row.email,
  },
  {
    key: "roles",
    header: "Role",
    // `?? []` menahan backend lama yang mengirim `roles: null` untuk user
    // tanpa role (ditemukan dari peramban sungguhan; backend baru selalu
    // mengirim array).
    render: (row) => (row.roles ?? []).join(", "),
  },
  {
    key: "is_active",
    header: "Status",
    width: "110px",
    render: (row) => (row.is_active ? "Aktif" : "Nonaktif"),
  },
];

function CreateUserDialog({
  roles,
  onClose,
}: {
  roles: AdminRole[];
  onClose: () => void;
}) {
  const createUser = useCreateAdminUser();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [roleIds, setRoleIds] = useState<string[]>([]);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<ApiError | null>(null);

  function toggleRole(id: string) {
    setRoleIds((current) =>
      current.includes(id)
        ? current.filter((roleId) => roleId !== id)
        : [...current, id],
    );
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrors({});
    setServerError(null);
    createUser.mutate(
      {
        username: username.trim(),
        email: email.trim(),
        password,
        role_ids: roleIds,
      },
      {
        onSuccess: () => onClose(),
        onError: (error) => {
          if (error instanceof ApiError) {
            setErrors(error.fieldErrors);
            setServerError(error);
          }
        },
      },
    );
  }

  return (
    <Dialog
      title="Buat user"
      description="User lahir di organisasi Anda. Role awal ditetapkan di sini; mengubahnya nanti lewat aksi Role."
      onClose={createUser.isPending ? () => {} : onClose}
      footer={
        <>
          <Button
            variant="quiet"
            onClick={onClose}
            disabled={createUser.isPending}
          >
            Batal
          </Button>
          <Button
            variant="primary"
            pending={createUser.isPending}
            type="submit"
            form="buat-user-form"
          >
            Buat user
          </Button>
        </>
      }
    >
      <form id="buat-user-form" onSubmit={submit} className="flex flex-col gap-3">
        <Field
          label="Username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          maxLength={255}
          error={errors.username}
        />
        <Field
          label="Email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          maxLength={255}
          error={errors.email}
        />
        <Field
          label="Password sementara"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          maxLength={255}
          error={errors.password}
          hint="Minimal 8 karakter. Sampaikan ke user terkait."
        />
        <fieldset>
          <legend className="text-13 font-medium text-text-soft">
            Role awal
          </legend>
          {roles.map((role) => (
            <label key={role.id} className="flex items-center gap-2 py-1">
              <input
                type="checkbox"
                checked={roleIds.includes(role.id)}
                onChange={() => toggleRole(role.id)}
                className="h-4 w-4"
              />
              <span className="text-14 text-text">{role.name}</span>
            </label>
          ))}
          {errors.role_ids ? (
            <p role="alert" className="text-12 text-danger">
              {errors.role_ids}
            </p>
          ) : null}
        </fieldset>
        {serverError && Object.keys(errors).length === 0 ? (
          <ErrorMessage error={serverError} />
        ) : null}
      </form>
    </Dialog>
  );
}

function EditUserDialog({
  user,
  onClose,
}: {
  user: AdminUser;
  onClose: () => void;
}) {
  const updateUser = useUpdateAdminUser();
  const [email, setEmail] = useState(user.email);
  const [isActive, setIsActive] = useState(user.is_active);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<ApiError | null>(null);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrors({});
    setServerError(null);
    updateUser.mutate(
      { id: user.id, input: { email: email.trim(), is_active: isActive } },
      {
        onSuccess: () => onClose(),
        onError: (error) => {
          if (error instanceof ApiError) {
            setErrors(error.fieldErrors);
            setServerError(error);
          }
        },
      },
    );
  }

  return (
    <Dialog
      title={`Ubah ${user.username}`}
      description="Menonaktifkan akun langsung mematikan seluruh sesinya; mengaktifkannya tidak mengembalikan sesi yang sudah mati."
      onClose={updateUser.isPending ? () => {} : onClose}
      footer={
        <>
          <Button
            variant="quiet"
            onClick={onClose}
            disabled={updateUser.isPending}
          >
            Batal
          </Button>
          <Button
            variant="primary"
            pending={updateUser.isPending}
            type="submit"
            form="ubah-user-form"
          >
            Simpan
          </Button>
        </>
      }
    >
      <form id="ubah-user-form" onSubmit={submit} className="flex flex-col gap-3">
        <Field
          label="Email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          maxLength={255}
          error={errors.email}
        />
        <label className="flex items-center gap-2 py-1">
          <input
            type="checkbox"
            checked={isActive}
            onChange={(event) => setIsActive(event.target.checked)}
            className="h-4 w-4"
          />
          <span className="text-14 text-text">Akun aktif</span>
        </label>
        {serverError && Object.keys(errors).length === 0 ? (
          <ErrorMessage error={serverError} />
        ) : null}
      </form>
    </Dialog>
  );
}

function SetRolesDialog({
  user,
  roles,
  onClose,
}: {
  user: AdminUser;
  roles: AdminRole[];
  onClose: () => void;
}) {
  const setRoles = useSetAdminUserRoles();
  const [roleIds, setRoleIds] = useState<string[]>(() =>
    roles.filter((role) => user.roles.includes(role.name)).map((role) => role.id),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<ApiError | null>(null);

  function toggleRole(id: string) {
    setRoleIds((current) =>
      current.includes(id)
        ? current.filter((roleId) => roleId !== id)
        : [...current, id],
    );
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrors({});
    setServerError(null);
    setRoles.mutate(
      { id: user.id, role_ids: roleIds },
      {
        onSuccess: () => onClose(),
        onError: (error) => {
          if (error instanceof ApiError) {
            setErrors(error.fieldErrors);
            setServerError(error);
          }
        },
      },
    );
  }

  return (
    <Dialog
      title={`Role ${user.username}`}
      description="Menggantikan seluruh himpunan role, bukan menambah. Mencabut administrator terakhir ditolak server."
      onClose={setRoles.isPending ? () => {} : onClose}
      footer={
        <>
          <Button
            variant="quiet"
            onClick={onClose}
            disabled={setRoles.isPending}
          >
            Batal
          </Button>
          <Button
            variant="primary"
            pending={setRoles.isPending}
            type="submit"
            form="atur-role-form"
          >
            Simpan role
          </Button>
        </>
      }
    >
      <form id="atur-role-form" onSubmit={submit} className="flex flex-col gap-1">
        {roles.map((role) => (
          <label key={role.id} className="flex items-center gap-2 py-1">
            <input
              type="checkbox"
              checked={roleIds.includes(role.id)}
              onChange={() => toggleRole(role.id)}
              className="h-4 w-4"
            />
            <span className="text-14 text-text">{role.name}</span>
          </label>
        ))}
        {errors.role_ids ? (
          <p role="alert" className="text-12 text-danger">
            {errors.role_ids}
          </p>
        ) : null}
        {serverError && Object.keys(errors).length === 0 ? (
          <ErrorMessage error={serverError} />
        ) : null}
      </form>
    </Dialog>
  );
}

function ResetPasswordDialog({
  user,
  onClose,
}: {
  user: AdminUser;
  onClose: () => void;
}) {
  const resetPassword = useResetAdminPassword();
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<ApiError | null>(null);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrors({});
    setServerError(null);
    resetPassword.mutate(
      { id: user.id, new_password: password },
      {
        onSuccess: () => onClose(),
        onError: (error) => {
          if (error instanceof ApiError) {
            setErrors(error.fieldErrors);
            setServerError(error);
          }
        },
      },
    );
  }

  return (
    <Dialog
      title={`Reset password ${user.username}`}
      description="Sesi user tersebut langsung mati semua. Sampaikan password sementara ini ke user terkait. Sistem tidak mengirim notifikasi."
      onClose={resetPassword.isPending ? () => {} : onClose}
      footer={
        <>
          <Button
            variant="quiet"
            onClick={onClose}
            disabled={resetPassword.isPending}
          >
            Batal
          </Button>
          <Button
            variant="primary"
            pending={resetPassword.isPending}
            type="submit"
            form="reset-password-form"
          >
            Reset password
          </Button>
        </>
      }
    >
      <form
        id="reset-password-form"
        onSubmit={submit}
        className="flex flex-col gap-3"
      >
        <Field
          label="Password sementara baru"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          maxLength={255}
          error={errors.new_password}
          hint="Minimal 8 karakter."
        />
        {serverError && Object.keys(errors).length === 0 ? (
          <ErrorMessage error={serverError} />
        ) : null}
      </form>
    </Dialog>
  );
}

function UsersTab() {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [searchDraft, setSearchDraft] = useState("");
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState<AdminUser | null>(null);
  const [settingRoles, setSettingRoles] = useState<AdminUser | null>(null);
  const [resetting, setResetting] = useState<AdminUser | null>(null);

  const canCreate = useAuthStore((state) => state.has("user:create"));
  const canUpdate = useAuthStore((state) => state.has("user:update"));
  const canManageRoles = useAuthStore((state) =>
    state.has("user_role:manage"),
  );

  const query = useAdminUsers({ page, limit: 20, search });
  const rolesQuery = useAdminRoles();
  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];
  const filtered = search.trim() !== "";

  const actionColumn: DataTableColumn<AdminUser> = {
    key: "actions",
    header: "Aksi",
    width: "220px",
    render: (row) => (
      <span className="flex flex-wrap gap-1.5">
        {canUpdate ? (
          <Button variant="quiet" onClick={() => setEditing(row)}>
            Ubah
          </Button>
        ) : null}
                      {canManageRoles ? (
                        <Button variant="quiet" onClick={() => setSettingRoles(row)}>
                          Role
                        </Button>
                      ) : null}
        {canUpdate ? (
          <Button variant="quiet" onClick={() => setResetting(row)}>
            Reset password
          </Button>
        ) : null}
      </span>
    ),
  };

  return (
    <div className="flex flex-col gap-4">
      <form
        role="search"
        aria-label="Penyaring user"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={(event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          setSearch(searchDraft.trim());
          setPage(1);
        }}
      >
        <div className="min-w-[220px] flex-1">
          <Field
            label="Cari"
            value={searchDraft}
            onChange={(event) => setSearchDraft(event.target.value)}
            placeholder="Username atau email"
            maxLength={255}
          />
        </div>
        <Button type="submit">Cari</Button>
        {filtered ? (
          <Button
            variant="quiet"
            onClick={() => {
              setSearch("");
              setSearchDraft("");
              setPage(1);
            }}
          >
            Bersihkan penyaring
          </Button>
        ) : null}
        {canCreate ? (
          <Button variant="primary" onClick={() => setCreating(true)}>
            Buat user
          </Button>
        ) : null}
      </form>

      <DataTable
        caption="Daftar user organisasi ini"
        columns={[
          ...userColumns,
          ...(canUpdate || canManageRoles ? [actionColumn] : []),
        ]}
        rows={rows}
        rowKey={(row) => row.id}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={setPage}
        emptyState={
          <EmptyState
            title={filtered ? "Tidak ada user yang cocok" : "Belum ada user"}
            description="Penyaring yang aktif tidak menyisakan satu user pun, atau organisasi ini belum punya user selain Anda."
          />
        }
      />

      {creating ? (
        <CreateUserDialog
          roles={rolesQuery.data ?? []}
          onClose={() => setCreating(false)}
        />
      ) : null}
      {editing ? (
        <EditUserDialog user={editing} onClose={() => setEditing(null)} />
      ) : null}
      {settingRoles ? (
        <SetRolesDialog
          user={settingRoles}
          roles={rolesQuery.data ?? []}
          onClose={() => setSettingRoles(null)}
        />
      ) : null}
      {resetting ? (
        <ResetPasswordDialog
          user={resetting}
          onClose={() => setResetting(null)}
        />
      ) : null}
    </div>
  );
}

function RolesTab() {
  const rolesQuery = useAdminRoles();
  const rows = rolesQuery.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      {/*
        Kejujuran batas: matriks izin per role dibaca dari `44-SECURITY.md`
        §3.1 di dokumen dan dari tabel `role_permissions` di database — tidak
        ada endpoint yang menyajikannya, jadi halaman ini tidak mengarang
        tabelnya.
      */}
      <p role="status" className="text-12 text-text-muted">
        Matriks izin per role belum punya endpoint baca; daftar di bawah hanya
        nama role. Sumber izin yang berlaku: tabel `role_permissions`.
      </p>
      <DataTable
        caption="Daftar role sistem"
        columns={[
          {
            key: "name",
            header: "Nama",
            render: (row: AdminRole) => (
              <span className="font-mono text-12 text-text">{row.name}</span>
            ),
          },
        ]}
        rows={rows}
        rowKey={(row) => row.id}
        loading={rolesQuery.isPending}
        error={rolesQuery.error instanceof ApiError ? rolesQuery.error : null}
        onRetry={() => void rolesQuery.refetch()}
        emptyState={
          <EmptyState
            title="Belum ada role"
            description="Seharusnya tidak terjadi: migrasi seed selalu menanam empat role sistem."
          />
        }
      />
    </div>
  );
}

function OrganizationDialog({
  initial,
  onClose,
}: {
  initial: AdminOrg | null;
  onClose: () => void;
}) {
  const createOrg = useCreateAdminOrganization();
  const updateOrg = useUpdateAdminOrganization();
  const pending = createOrg.isPending || updateOrg.isPending;
  const [name, setName] = useState(initial?.name ?? "");
  const [code, setCode] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<ApiError | null>(null);
  const creating = initial === null;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrors({});
    setServerError(null);
    const done = () => onClose();
    const failed = (error: unknown) => {
      if (error instanceof ApiError) {
        setErrors(error.fieldErrors);
        setServerError(error);
      }
    };
    if (creating) {
      createOrg.mutate(
        { name: name.trim(), code: code.trim() },
        { onSuccess: done, onError: failed },
      );
    } else if (initial) {
      updateOrg.mutate(
        { id: initial.id, name: name.trim() },
        { onSuccess: done, onError: failed },
      );
    }
  }

  return (
    <Dialog
      title={creating ? "Buat organisasi" : `Ubah ${initial?.name ?? ""}`}
      description={
        creating
          ? "Kode unik per sistem dan tidak dapat diubah sesudah dibuat."
          : "Hanya nama yang dapat diubah; kode dipakai sebagai rujukan eksternal."
      }
      onClose={pending ? () => {} : onClose}
      footer={
        <>
          <Button variant="quiet" onClick={onClose} disabled={pending}>
            Batal
          </Button>
          <Button
            variant="primary"
            pending={pending}
            type="submit"
            form="organisasi-form"
          >
            {creating ? "Buat organisasi" : "Simpan"}
          </Button>
        </>
      }
    >
      <form
        id="organisasi-form"
        onSubmit={submit}
        className="flex flex-col gap-3"
      >
        <Field
          label="Nama"
          value={name}
          onChange={(event) => setName(event.target.value)}
          maxLength={255}
          error={errors.name}
        />
        {creating ? (
          <Field
            label="Kode"
            value={code}
            onChange={(event) => setCode(event.target.value)}
            maxLength={50}
            error={errors.code}
            hint="Unik per sistem, mis. CONTOH."
          />
        ) : null}
        {serverError && Object.keys(errors).length === 0 ? (
          <ErrorMessage error={serverError} />
        ) : null}
      </form>
    </Dialog>
  );
}

function OrganizationsTab() {
  const [dialog, setDialog] = useState<{ open: boolean; org: AdminOrg | null }>({
    open: false,
    org: null,
  });

  const canCreate = useAuthStore((state) =>
    state.has("organization:create"),
  );
  const canUpdate = useAuthStore((state) =>
    state.has("organization:update"),
  );

  const query = useAdminOrganizations();
  const rows = query.data ?? [];

  const actionColumn: DataTableColumn<AdminOrg> = {
    key: "actions",
    header: "Aksi",
    width: "110px",
    render: (row) => (
      <Button variant="quiet" onClick={() => setDialog({ open: true, org: row })}>
        Ubah
      </Button>
    ),
  };

  return (
    <div className="flex flex-col gap-4">
      {canCreate ? (
        <div>
          <Button
            variant="primary"
            onClick={() => setDialog({ open: true, org: null })}
          >
            Buat organisasi
          </Button>
        </div>
      ) : null}
      <DataTable
        caption="Daftar organisasi"
        columns={[
          {
            key: "name",
            header: "Nama",
            render: (row: AdminOrg) => (
              <span className="font-medium text-text">{row.name}</span>
            ),
          },
          {
            key: "code",
            header: "Kode",
            width: "140px",
            render: (row: AdminOrg) => (
              <span className="font-mono text-12 text-text-soft">
                {row.code}
              </span>
            ),
          },
          ...(canUpdate ? [actionColumn] : []),
        ]}
        rows={rows}
        rowKey={(row) => row.id}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        emptyState={
          <EmptyState
            title="Belum ada organisasi"
            description="Seharusnya tidak terjadi: bootstrap selalu membuat organisasi pertama."
          />
        }
      />
      {dialog.open ? (
        <OrganizationDialog
          initial={dialog.org}
          onClose={() => setDialog({ open: false, org: null })}
        />
      ) : null}
    </div>
  );
}

function CategoryDialog({
  initial,
  onClose,
}: {
  initial: DocumentCategory | null;
  onClose: () => void;
}) {
  const createCategory = useCreateDocumentCategory();
  const updateCategory = useUpdateDocumentCategory();
  const pending = createCategory.isPending || updateCategory.isPending;
  const [name, setName] = useState(initial?.name ?? "");
  const [code, setCode] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [serverError, setServerError] = useState<ApiError | null>(null);
  const creating = initial === null;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrors({});
    setServerError(null);
    const done = () => onClose();
    const failed = (error: unknown) => {
      if (error instanceof ApiError) {
        setErrors(error.fieldErrors);
        setServerError(error);
      }
    };
    if (creating) {
      createCategory.mutate(
        { name: name.trim(), code: code.trim() },
        { onSuccess: done, onError: failed },
      );
    } else if (initial) {
      updateCategory.mutate(
        { id: initial.id, name: name.trim() },
        { onSuccess: done, onError: failed },
      );
    }
  }

  return (
    <Dialog
      title={creating ? "Buat kategori" : `Ubah ${initial?.name ?? ""}`}
      description={
        creating
          ? "Kode unik per organisasi dan tidak dapat diubah sesudah dibuat."
          : "Hanya nama yang dapat diubah."
      }
      onClose={pending ? () => {} : onClose}
      footer={
        <>
          <Button variant="quiet" onClick={onClose} disabled={pending}>
            Batal
          </Button>
          <Button
            variant="primary"
            pending={pending}
            type="submit"
            form="kategori-form"
          >
            {creating ? "Buat kategori" : "Simpan"}
          </Button>
        </>
      }
    >
      <form id="kategori-form" onSubmit={submit} className="flex flex-col gap-3">
        <Field
          label="Nama"
          value={name}
          onChange={(event) => setName(event.target.value)}
          maxLength={100}
          error={errors.name}
        />
        {creating ? (
          <Field
            label="Kode"
            value={code}
            onChange={(event) => setCode(event.target.value)}
            maxLength={50}
            error={errors.code}
            hint="Unik per organisasi, mis. SOP."
          />
        ) : null}
        {serverError && Object.keys(errors).length === 0 ? (
          <ErrorMessage error={serverError} />
        ) : null}
      </form>
    </Dialog>
  );
}

function CategoriesTab() {
  const [dialog, setDialog] = useState<{
    open: boolean;
    category: DocumentCategory | null;
  }>({ open: false, category: null });
  const [deleting, setDeleting] = useState<DocumentCategory | null>(null);
  const [deleteError, setDeleteError] = useState<ApiError | null>(null);
  const deleteCategory = useDeleteDocumentCategory();

  const canManage = useAuthStore((state) =>
    state.has("document_category:manage"),
  );

  const query = useDocumentCategories();
  const rows = query.data ?? [];

  const actionColumn: DataTableColumn<DocumentCategory> = {
    key: "actions",
    header: "Aksi",
    width: "190px",
    render: (row) => (
      <span className="flex flex-wrap gap-1.5">
        <Button
          variant="quiet"
          onClick={() => setDialog({ open: true, category: row })}
        >
          Ubah
        </Button>
        <Button
          variant="quiet"
          onClick={() => {
            setDeleteError(null);
            setDeleting(row);
          }}
        >
          Hapus
        </Button>
      </span>
    ),
  };

  function confirmDelete() {
    if (!deleting) return;
    setDeleteError(null);
    deleteCategory.mutate(deleting.id, {
      onSuccess: () => setDeleting(null),
      onError: (error) => {
        if (error instanceof ApiError) setDeleteError(error);
      },
    });
  }

  return (
    <div className="flex flex-col gap-4">
      {canManage ? (
        <div>
          <Button
            variant="primary"
            onClick={() => setDialog({ open: true, category: null })}
          >
            Buat kategori
          </Button>
        </div>
      ) : null}
      <DataTable
        caption="Daftar kategori dokumen organisasi ini"
        columns={[
          {
            key: "name",
            header: "Nama",
            render: (row: DocumentCategory) => (
              <span className="font-medium text-text">{row.name}</span>
            ),
          },
          {
            key: "code",
            header: "Kode",
            width: "140px",
            render: (row: DocumentCategory) => (
              <span className="font-mono text-12 text-text-soft">
                {row.code}
              </span>
            ),
          },
          ...(canManage ? [actionColumn] : []),
        ]}
        rows={rows}
        rowKey={(row) => row.id}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        emptyState={
          <EmptyState
            title="Belum ada kategori"
            description="Kategori dipakai memilih golongan dokumen pada form dan penyaring."
          />
        }
      />
      {dialog.open ? (
        <CategoryDialog
          initial={dialog.category}
          onClose={() => setDialog({ open: false, category: null })}
        />
      ) : null}
      {deleting ? (
        <Dialog
          title={`Hapus ${deleting.name}`}
          description="Kategori yang masih dipakai dokumen ditolak server. Penghapusan bersifat permanen."
          onClose={deleteCategory.isPending ? () => {} : () => setDeleting(null)}
          footer={
            <>
              <Button
                variant="quiet"
                onClick={() => setDeleting(null)}
                disabled={deleteCategory.isPending}
              >
                Batal
              </Button>
              <Button
                variant="danger"
                pending={deleteCategory.isPending}
                onClick={confirmDelete}
              >
                Hapus kategori
              </Button>
            </>
          }
        >
          {deleteError ? <ErrorMessage error={deleteError} /> : null}
        </Dialog>
      ) : null}
    </div>
  );
}

function SettingsTab() {
  return (
    <div className="flex flex-col gap-4">
      {/*
        Kejujuran batas: halaman Settings `50-FSD.md` §10.5 hanya boleh
        menampilkan kunci yang benar-benar dibaca aplikasi — tetapi tidak ada
        endpoint baca (`setting:read` tidak dipakai route mana pun), dan
        `PATCH /admin/settings/:key` terkontrak tanpa route. Menulis tanpa
        membaca berarti buta; bagian ini menunggu backend-nya.
      */}
      <p role="status" className="text-13 text-text">
        Pengaturan sistem belum dapat dikelola dari sini: tidak ada endpoint
        baca, dan endpoint tulisnya (`PATCH /admin/settings/:key`) belum
        terpasang. Kunci yang berlaku tercatat di `41-DATABASE.md` §2.6.
      </p>
    </div>
  );
}

export function AdministrationPage() {
  const [params, setParams] = useSearchParams();

  const sectionParam = params.get("section") ?? "";
  const section: AdminSection =
    sectionParam === "roles" ||
    sectionParam === "organizations" ||
    sectionParam === "categories" ||
    sectionParam === "settings"
      ? sectionParam
      : "users";

  function selectSection(next: AdminSection) {
    setParams(next === "users" ? {} : { section: next });
  }

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Administration"
        description="Kelola user, role, organisasi, dan kategori organisasi ini."
      />

      <nav aria-label="Bagian administrasi" className="flex flex-wrap gap-1.5">
        <div role="tablist" aria-label="Bagian" className="flex flex-wrap gap-1.5">
          {adminSections.map((item) => {
            const active = section === item.id;
            return (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => selectSection(item.id)}
                className={[
                  "tap-target inline-flex items-center rounded-control border px-2.5 text-13",
                  active
                    ? "border-line-strong bg-surface-sunken font-medium text-text"
                    : "border-line text-text-soft hover:bg-surface-hover hover:text-text",
                ].join(" ")}
              >
                {item.label}
              </button>
            );
          })}
        </div>
      </nav>

      {section === "users" ? (
        <UsersTab />
      ) : section === "roles" ? (
        <RolesTab />
      ) : section === "organizations" ? (
        <OrganizationsTab />
      ) : section === "categories" ? (
        <CategoriesTab />
      ) : (
        <SettingsTab />
      )}
    </div>
  );
}
