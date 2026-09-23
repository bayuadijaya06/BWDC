# CHANGELOG — Perubahan File

**Sifat:** append-only. Format mengikuti semangat Keep a Changelog: tambahan per tanggal, kategori `Added`, `Changed`, `Fixed`, `Removed`.
**Aturan:** setiap file yang dibuat, diubah, atau dihapus WAJIB tercatat di sini pada tanggal kejadian, dengan alasan singkat dan ID prompt penyebabnya.
**Protokol:** `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`

---

## 2026-09-24 (sesi P-071)

Documents category filter `T-085` DONE. Backend: `GET /documents/categories` (`document_category:read`, semua role), `ListCategories` di repository/service/handler + test `TestDocumentListCategories`. Frontend: dropdown Kategori di `Documents/index.tsx` (setelah Project, sebelum rentang), `listDocumentCategories()` + `useDocumentCategories()`, `category_id` mengalir ke query. Test: "mengirim category_id dari penyaring kategori". Backend **283 → 284 test**, frontend **299 → 300 test**. Menutup Q-016 sisi kategori.

### Added

- `backend/internal/repository/document_repository.go` — `ListCategories` (P-071)
- `backend/internal/service/document_service.go` — `ListCategories` (P-071)
- `backend/internal/handler/document_handler.go` — `ListCategories` handler (P-071)
- `backend/internal/handler/document_handler_test.go` — `TestDocumentListCategories` (P-071)

### Changed

- `backend/internal/handler/router.go` — `GET /documents/categories` (52 route total, 8 document) (P-071)
- `frontend/src/services/documents.ts` — `DocumentCategory`, `listDocumentCategories()` (P-071)
- `frontend/src/queries/documents.ts` — `useDocumentCategories()` (P-071)
- `frontend/src/pages/Documents/index.tsx` — dropdown Kategori + `category_id` ke query (P-071)
- `frontend/src/pages/Documents/Documents.test.tsx` — mock + test category filter (P-071)
- `README.md` — `7 → 8 document endpoint`, `51 → 52 route` (P-071)
- `docs/progress/STATE.md` — `283 → 284 test`, T-085 DONE (P-071)

---

## 2026-09-24 (sesi P-070)

Project Members CRUD UI `T-084` DONE. Tab Members di `ProjectDetail.tsx`: tombol "Tambah anggota" (hanya bila `project_member:manage`), dialog add member (radio user search via `GET /admin/users` + select role `SelectField`, formError 409/422 inline), hapus anggota bukan owner via `DELETE /projects/:id/members/:userId` + invalidasi kueri project. Baru: `services/admin.ts`, `queries/admin.ts`, `components/common/SelectField.tsx`. Frontend **294 → 299 test** (naik 5), menutup **C-063/Q-024**.

### Added

- `frontend/src/services/admin.ts` — `listAdminUsers`, `listAdminRoles`, `listAdminOrganizations` (P-070)
- `frontend/src/queries/admin.ts` — `useAdminUsers` (P-070)
- `frontend/src/components/common/SelectField.tsx` — select field dengan label/hint/error (P-070)

### Changed

- `frontend/src/pages/Projects/ProjectDetail.tsx` — tab Members: tombol "Tambah anggota", dialog add, kolom Aksi (Hapus untuk non-owner, "-" untuk owner), error handling 409/422 (P-070)
- `frontend/src/pages/Projects/ProjectDetail.test.tsx` — 5 test baru: manage button visible, hidden without permission, Hapus for non-owner, not for owner, dialog + search (P-070)

---

## 2026-09-24 (sesi P-069)

Administration Users/Roles/Organizations `GET /admin/users`, `POST /admin/users`, `GET /admin/roles`, `GET /admin/organizations` (user:read/create, role:read, organization:read, Admin saja) — `T-083` DONE. `UserService` ListUsers (search, page/limit, total), CreateUser (username/email/password 8+, role_ids UUID, bcrypt 12, tx + user_roles, audit USER_CREATED), ListRoles, ListOrganizations; `UserHandler` 4 handlers + validasi 401/403/422 + 201/200; `router.go` 5 admin routes (51 total, 1→5 admin), `user_handler_test.go` (401/403, list 200 meta, create 201 + search, roles 200, viewer 403).

### Added

- `backend/internal/dto/user_dto.go` — `CreateUserRequest`, `UserResponse`, `RoleResponse`, `OrganizationResponse` (P-069)
- `docs/progress/prompts/P-069-2026-09-24-administration-users-roles-orgs.md` (P-069)

### Changed

- `backend/internal/service/user_service.go` — `ListUsers`, `CreateUser`, `ListRoles`, `ListOrganizations` + `ErrUserAlreadyExists`/`ErrRoleNotFound` (P-069)
- `backend/internal/handler/user_handler.go` — `ListUsers`, `CreateUser`, `ListRoles`, `ListOrganizations` (P-069)
- `backend/internal/handler/router.go` — 5 admin routes (`GET /admin/users`, `POST /admin/users`, `GET /admin/roles`, `GET /admin/organizations`, `POST /admin/users/:id/unlock`) → 51 total (P-069)
- `backend/internal/handler/user_handler_test.go` — `TestAdminUsersListAndCreate`, `TestAdminRolesAndOrgs` (P-069)
- `README.md` — `47 → 51 route (1→5 admin)` (P-069)
- `docs/progress/STATE.md` — `281 → 283 test` (P-069)

---

## 2026-09-24 (sesi P-068)

Reports Export `GET /reports/export?type=projects|documents|tasks&format=csv` (report:export, cakupan 44-SECURITY §3.1.3, audit REPORT_EXPORTED) — `T-082` DONE. `ReportService` CSV header = kolom tabel 50-FSD (projects: code/name/status/owner/start/target/member_count/created_at; documents: document_number/title/category/status/version/owner/updated_at; tasks: title/status/priority/due_date/assignee/project) dengan `systemScope` / `taskScope` + Limit 10000, `ReportHandler` validasi `type`/`format`/`project_id` (422) + `401`/`403`, `router.go` 1 reports route (47 total), `main.go` wiring, `main_test.go` report engine, `report_handler_test.go` (401/403/422 + 200 csv header + Content-Disposition `bwdcs-<type>-<YYYYMMDD>.csv` + audit), `check-readme-facts.sh` ember reports + `README 46→47`, `STATE 279→281`.

### Added

- `backend/internal/dto/report_dto.go` — `ReportExportQuery` (P-068)
- `backend/internal/service/report_service.go` — `REPORT_EXPORTED` + `Export` CSV + audit tx (P-068)
- `backend/internal/handler/report_handler.go` — `Export` + `parseReportExportQuery` (P-068)
- `backend/internal/handler/report_handler_test.go` — `TestReportExportValidation` + `TestReportExportSuccess` (P-068)
- `docs/progress/prompts/P-068-2026-09-24-reports-export.md` (P-068)

### Changed

- `backend/internal/handler/router.go` — `Report` deps + `GET /reports/export` `report:export` (P-068)
- `backend/cmd/server/main.go` — `reportService` wiring + `Report` handler (P-068)
- `backend/internal/handler/main_test.go` — `report` engineParts + wiring (P-068)
- `scripts/check-readme-facts.sh` — `n_reports` + `known_receivers` + `actual_parts` reports (P-068)
- `README.md` — `46 → 47 route (1 reports)` (P-068)
- `docs/progress/STATE.md` — `279 → 281 test` (P-068)

---

## 2026-09-24 (sesi P-067)

Perbaikan create button, grouping chart dashboard, kontras dark mode, audit antislop — `T-090` DONE. Create button `primary` `bg-accent text-paper-000` (dark `1.74:1` FAIL) → `bg-text text-surface-raised border-text hover:opacity-90` (light `17.8:1`, dark `14.1:1` — setara `Daftar Project` `border-line-strong bg-surface-raised text-text`). Dashboard 8 chart dalam satu grid → 3 tab `Dokumen` (Sebaran, Funnel, Kategori) / `Workflow` (Volume, Approval, Activity) / `Antrian` (Aging, Avg Time) dengan `role=tablist` `aria-selected`; per chart `CartesianGrid stroke var(--line)`, `XAxis/YAxis tick var(--text-muted) stroke var(--line-strong)`, `Tooltip contentStyle bg var(--surface-raised) border var(--line)`, `Legend wrapperStyle var(--text-muted)`. Test `Dashboard.test.tsx` tab grouping + `Tasks.test.tsx` `WEB` title.

### Added

- `docs/progress/prompts/P-067-2026-09-24-perbaikan-create-button-dashboard-tabs-dan-kontras.md` (P-067)

### Changed

- `frontend/src/components/common/Button.tsx`: `primary` `bg-accent` → `bg-text text-surface-raised border-text` (P-067)
- `frontend/src/pages/Dashboard/index.tsx`: `chartTab` state + 3 tab + `chartTick/chartGridStroke/tooltipStyle` + 8 chart dibagi 3/3/2 + dark contrast per chart (P-067)
- `frontend/src/pages/Dashboard/Dashboard.test.tsx`: tab grouping (Dokumen aktif, Workflow→Volume, Antrian→Aging) + axe (P-067)
- `frontend/src/pages/Tasks/Tasks.test.tsx`: link `Website Redesign` → `WEB` + `title` (P-067)
- `docs/progress/TASKS.md`: `T-090` DONE (P-067)
- `docs/progress/STATE.md`, `CONTINUE.md`: header 2026-09-24 P-067, frontend 294/30 (P-067)
- `docs/progress/SESSION-LOG.md`: entri P-067 (P-067)

---

## 2026-09-23 (sesi P-066)

Polish filter, tabel Tasks, dan tombol Create — `T-089` DONE. Filter `flex flex-wrap gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm` + kolom `flex-1 min-w-[140px] max-w-[200px] min-w-0` (tidak blending, lebar konsisten), tabel Tasks `density comfortable` (`h-11` 44px) + kolom `due_date 190→140` `assignee 150→120` `project 190→130` (tidak penuh), tombol Create `primary` `bg-accent text-paper-000 border-accent-strong shadow-sm font-semibold` (kontras light `#0e5b63`/`#ffffff` 7:1 dan dark `#7fd1d9`/`#1c1f24`). `vite.config.ts` `optimizeDeps: { include: ["recharts"] }` + `rm -rf .vite` (recharts MIME block).

### Changed

- `frontend/src/components/common/Button.tsx`: `primary` `bg-accent` `shadow-sm` (P-066)
- `frontend/src/pages/Tasks/index.tsx` + `Documents` + `Projects`: filter `bg-surface-raised` + kolom `flex-1` (P-066)
- `frontend/src/pages/Tasks/index.tsx`: `taskColumns` `due_date`/`assignee`/`project` sempit + `DataTable density comfortable` (P-066)
- `frontend/src/pages/Dashboard/index.tsx` + `ProjectDetail.tsx`: ` - ` untuk R-02 + `var(--color-status-*)` (P-066)
- `frontend/vite.config.ts`: `optimizeDeps: { include: ["recharts"] }` (P-066)
- `docs/progress/TASKS.md`: `T-089` DONE `T-089` (P-066)
- `docs/progress/prompts/P-066-...md` (P-066)

---

## 2026-09-23 (sesi P-065)

Project Activity tab inline — `T-081` DONE. Tab Activity di `ProjectDetail` yang sebelumnya `EmptyState` `Bagian Activity belum dibangun` kini menjadi `Panel` `Aktivitas` dengan `DataTable` `GET /audit?project_id=` (`metadata->>'project_id'` / `entity=project`), `loading`/`error`/`meta`, `emptyState` `Belum ada aktivitas`. `builtTabs` `+activity`, `pendingTabs` `1→0` (activity keluar). `project_id` bukan UUID `422 field=project_id`. `42-API.md` §9 Query `?project_id=` + Activity tab `ProjectDetail`.

### Changed

- `backend/internal/repository/audit_repository.go`: `ProjectID`, `List`/`count` `project_id` (P-065)
- `backend/internal/handler/audit_handler.go`: `project_id` UUID `422`, `42-API.md` §9 (P-065)
- `docs/design/42-API.md` §9: Query `?project_id=` + Activity tab `ProjectDetail` (P-065)
- `frontend/src/services/audit.ts`: `project_id` (P-065)
- `frontend/src/queries/audit.ts`: `useAuditList` (P-065)
- `frontend/src/pages/Projects/ProjectDetail.tsx`: `builtTabs` `+activity`, `pendingTabs` `1→0`, `useAuditList` + `activityColumns` + `Panel` (P-065)
- `backend/internal/handler/audit_handler_test.go`: `TestAuditListProjectFilter` (P-065)
- `frontend/src/pages/Projects/ProjectDetail.test.tsx`: Mock `listAudit`, `activity` built (P-065)
- `docs/progress/TASKS.md`: `T-081` DONE `T-081` (P-065)
- `docs/progress/prompts/P-065-...md` (P-065)

---

## 2026-09-23 (sesi P-064)

Project Workflow tab inline — `T-080` DONE. Tab Workflow di `ProjectDetail` yang sebelumnya `EmptyState` `Bagian Workflow belum dibangun` kini menjadi `Panel` `Workflow` dengan `DataTable` `GET /workflows/instances?project_id=` (scope `projectScopePredicate` + `p.id = $8`), `loading`/`error`/`meta`, `emptyState` `Belum ada workflow`. `builtTabs` `+workflow`, `pendingTabs` `3→1` (workflow keluar). `project_id` bukan UUID `422 field=project_id`. `42-API.md` §5 Query `?project_id=` + Workflow tab `ProjectDetail`.

### Changed

- `backend/internal/repository/workflow_repository.go`: `ProjectID`, `ListInstances`/`countInstances` `project_id` (P-064)
- `backend/internal/service/workflow_service.go`: `ProjectID` (P-064)
- `backend/internal/handler/workflow_handler.go`: `project_id` UUID `422`, `42-API.md` §5 (P-064)
- `docs/design/42-API.md` §5: Query `?project_id=` + Workflow tab `ProjectDetail` (P-064)
- `frontend/src/services/workflows.ts`: `project_id` (P-064)
- `frontend/src/queries/workflows.ts`: `+ enabled` (P-064)
- `frontend/src/pages/Projects/ProjectDetail.tsx`: `builtTabs` `+workflow`, `pendingTabs` 3→1, `useWorkflowInstances` + `workflowColumns` + `Panel` (P-064)
- `backend/internal/handler/workflow_handler_test.go`: `TestWorkflowListProjectFilter` (P-064)
- `docs/progress/TASKS.md`: `T-080` DONE `T-080` (P-064)
- `docs/progress/prompts/P-064-...md` (P-064)

---

## 2026-09-23 (sesi P-063)

Project Tasks tab inline — `T-079` DONE. Tab Tasks di `ProjectDetail` yang sebelumnya `EmptyState` `Bagian Tasks belum dibangun` kini menjadi `Panel` `Tugas` dengan `DataTable` `GET /tasks?project_id=` (`taskScopePredicate`), `loading`/`error`/`meta`, `emptyState` `Belum ada tugas`. `builtTabs` `+tasks`, `pendingTabs` `3→2` (tasks keluar). `useTaskList` `enabled` ditambah (`queries/tasks.ts`).

### Changed

- `frontend/src/pages/Projects/ProjectDetail.tsx`: `builtTabs` `+tasks`, `pendingTabs` 3→2, `useTaskList` di atas + `taskColumns` + `Panel` `DataTable` (P-063)
- `frontend/src/queries/tasks.ts`: `useTaskList` `+ enabled` (P-063)
- `frontend/src/pages/Projects/ProjectDetail.test.tsx`: Mock `listTasks`, 2 test Documents/Workflow pending (P-063)
- `docs/progress/TASKS.md`: `T-079` DONE `T-079` (P-063)
- `docs/progress/prompts/P-063-...md` (P-063)

---

## 2026-09-23 (sesi P-062)

Project Documents tab inline — `T-078` DONE. Tab Documents di `ProjectDetail` yang sebelumnya hanya `EmptyState` `Bagian Documents belum dibangun` + link `Buka dokumen project ini` kini menjadi `Panel` `Dokumen` dengan `DataTable` `GET /documents?project_id=` (scope `projectScopePredicate`), `loading`/`error`/`meta`, `emptyState` `Belum ada dokumen`. `builtTabs` `+documents`, `pendingTabs` `4→3` (documents keluar), ` - ` untuk R-02. Hook `useDocumentList` dipindah ke atas (sebelum `return` awal) — sebelumnya di bawah `project` (bersyarat → crash `react_stack_bottom_frame`).

### Changed

- `frontend/src/pages/Projects/ProjectDetail.tsx`: `builtTabs` `+documents`, `pendingTabs` 4→3, `useDocumentList` di atas + `documentColumns` + `Panel` `DataTable` (P-062)
- `frontend/src/pages/Projects/ProjectDetail.test.tsx`: Mock `listDocuments`, 2 test Documents/Tasks (P-062)
- `docs/progress/TASKS.md`: `T-078` DONE `T-078` (P-062)
- `docs/progress/prompts/P-062-...md` (P-062)

---

## 2026-09-23 (sesi P-061)

Analisis penunggu 4 tab `ProjectDetail` + `Reports`/`Administration`: `Documents`/`Tasks` siap inline (`GET /documents?project_id=` & `GET /tasks?project_id=` sudah hidup — `T-078`/`T-079` READY), `Workflow` butuh `?project_id` pada `GET /workflows/instances` (`T-080`), `Activity` butuh `?project_id` pada `GET /audit` (`T-081`), `Reports` `GET /reports/export` (`T-082`) dan `Administration` `GET /admin/users` dll. (`T-083`) belum ada di `TASKS.md` TODO. `ProjectDetail.tsx` 4 `pendingTabs` alasan/reference diperbarui (`T-078`..`T-081`, ` - `).

### Changed

- `docs/progress/TASKS.md`: TODO `T-078`..`T-083` (6 task) (P-061)
- `frontend/src/pages/Projects/ProjectDetail.tsx`: `pendingTabs` 4 alasan/reference (`T-078`..`T-081`) (P-061)
- `frontend/src/pages/Projects/ProjectDetail.test.tsx`: `getByText` eksak → regex + `getAllByText(/T-078/)` (P-061)
- `docs/progress/prompts/P-061-...md` (P-061)

---

## 2026-09-23 (sesi P-060)

Audit read hidup — `T-077` DONE. `GET /audit?actor_id=&action=&entity=&entity_id=&date_from=&date_to=&page=&limit=` (`audit:read` Admin, terbaru dulu `ORDER BY created_at DESC`, `LEFT JOIN users` untuk `actor_name`, `COUNT(*) OVER()` + `count` untuk offset>0). `page`/`limit` `422` bila salah, `actor_id` bukan UUID `422`, `date_from`/`date_to` `YYYY-MM-DD`/RFC3339 `422`, `date_to < date_from` `422`, tanpa token `401`, viewer `403`, admin `200` dengan `data` array + `meta`. Route total `46` (1 audit).

### Added

- `backend/internal/{model/audit,repository/audit_repository,service/audit_read_service,handler/audit_handler}.go` (P-060, `T-077`)
- `backend/internal/handler/audit_handler_test.go` (P-060) — 2 `func Test` (6 subtest: `page` `422`, `actor_id` `422`, tanpa token `401`, viewer `403`, admin `200`)

### Changed

- `backend/internal/handler/router.go`: `Audit` deps + 1 route `audit:read` (P-060)
- `backend/cmd/server/main.go`: wiring `AuditReadService`/`Handler` (P-060)
- `backend/internal/handler/main_test.go`: `engineParts.audit` (P-060)
- `scripts/check-readme-facts.sh`: `n_audit`, `known_receivers` + audit, `actual_parts` 11 (P-060)
- `README.md`: `45 route` → `46 route` (1 audit) (P-060)
- `docs/progress/{TASKS,STATE}.md`, `docs/progress/prompts/P-060-...md` (P-060)

---

## 2026-09-23 (sesi P-059)

Notifikasi in-app hidup — `T-076` DONE. `GET /notifications?is_read=&page=&limit=` (`notification:read` semua role, cakupan `user_id = user`, `is_read` nil = tanpa filter) + `PATCH /:id/read` (`notification:update`, `id` bukan UUID `422`, bukan milik `404`) + `POST /read-all` (`notification:update`). Tabel `notifications` sudah diisi workflow, kini dapat dibaca. Route total `45` (1 health +5 auth +1 admin +8 project +7 document +5 task +5 comment +9 workflow +1 analytics +3 notifications).

### Added

- `backend/internal/{model/notification,repository/notification_repository,service/notification_service,handler/notification_handler}.go` (P-059, `T-076`)
- `backend/internal/handler/notification_handler_test.go` (P-059) — 2 `func Test` (7 subtest: `is_read` `422`, tanpa token `401`, `id` `422`/`404`, milik `200`, `read-all` `200`)

### Changed

- `backend/internal/handler/router.go`: `Notification` deps + 3 route (P-059)
- `backend/cmd/server/main.go`: wiring `NotificationService`/`Handler` (P-059)
- `backend/internal/handler/main_test.go`: `engineParts.notification` (P-059)
- `scripts/check-readme-facts.sh`: `n_notification`, `known_receivers` + notifications, `actual_parts` 10 (P-059)
- `README.md`: `42 route` → `45 route` (1 analytics +3 notifications) (P-059)
- `docs/progress/{TASKS,STATE}.md`, `docs/progress/prompts/P-059-...md` (P-059)

---

## 2026-09-23 (sesi P-058)

Kategori dokumen pada `GET /documents` (`?category_id=` — Q-016) hidup tanpa migrasi baru (`document_categories` sudah ada `41-DATABASE.md` §2.3). `category_id` bukan UUID → `422 field=category_id`; tidak ada kategori → `0`; kategori organisasi lain → `0`. `List`/`count` memakai `$9` yang sama (`COUNT(*) OVER()` + tambalan `C-048`). Frontend `services/documents.ts` mengirim `category_id`. Sisa Q-016 hanya `owner` (menunggu Q-024).

### Changed

- `backend/internal/repository/document_repository.go`: `CategoryID`, `documentListWhere` `$9`, `LIMIT $10 OFFSET $11` (P-058)
- `backend/internal/service/document_service.go`: `CategoryID` (P-058)
- `backend/internal/handler/document_handler.go`: `parseDocumentListQuery` `category_id` UUID → `422` (P-058)
- `backend/internal/handler/document_handler_test.go`: `TestDocumentListCategoryFilter` + `documentListPayload.CategoryID` (P-058)
- `frontend/src/services/documents.ts`: `category_id` di `DocumentListQuery` + `listDocuments` (P-058)
- `frontend/src/services/documents.test.ts`: `mengirim category_id bila diisi` (P-058)
- `docs/design/42-API.md` §4: Query `?category_id=` + Category kini hidup (P-058)
- `docs/design/50-FSD.md` §4.1: Category kini hidup `?category_id=` (P-058)
- `docs/progress/TASKS.md`: `T-075` DONE `T-075` (P-058)
- `docs/progress/prompts/P-058-...md` (P-058)

---

## 2026-09-23 (sesi P-057)

Frontend Dashboard MVP hidup — `T-073` DONE. Halaman `/` yang sebelumnya hanya `Panel` kosong (tanpa angka karangan R-17) kini membaca `GET /analytics/dashboard` (ADR-0026) — KPI 6 (`total_documents` → `/documents`, `active_workflows`, `pending_approvals` → `/approvals`, `overdue_workflows`, `avg_approval_time_hours`, `revised_this_month`) + 8 chart (`statusDist` donut, `volumeTrend` line, `approvalTrend` stacked bar, `funnel` 5, `pendingAging` 5, `avgTimePerStage` horizontal bar, `byCategory` horizontal bar, `activityTrend` 4 line) dari `charts` — `ResponsiveContainer` `h-64`, filter global `?from=&to=&project_id=` di URL (`project_id` dari `useProjectList`), `report:read` guard, tanpa `#hex` (statusColors `var(--color-status-*)`, chart fills `var(--color-*)`). `recharts` `3.10.1` dipasang (`--legacy-peer-deps` untuk React 19); `@testing-library/dom` dipulihkan setelah terhapus. Perbaiki `tokens.contrast.test.ts` FAIL `#hex` + `findByText "0"` ambiguitas.

### Added

- `frontend/src/services/analytics.ts` (`fetchDashboard` + `validateDashboardRange`) + `frontend/src/queries/analytics.ts` (`useDashboard`) (P-057, `T-073`)
- `frontend/src/services/analytics.test.ts` (5 test) + `frontend/src/pages/Dashboard/Dashboard.test.tsx` (5 test) (P-057)
- `docs/progress/prompts/P-057-2026-09-23-dashboard-frontend.md` (P-057)

### Changed

- `frontend/package.json`: `recharts` `3.10.1` + `@testing-library/dom` `10` (P-057)
- `frontend/src/pages/Dashboard/index.tsx`: Dashboard MVP — filter di URL, KPI 6, 8 chart `recharts`, `report:read` guard (P-057)
- `docs/progress/TASKS.md`: `T-073` DONE `T-073` (P-057)

---

## 2026-09-23 (sesi P-056)

Backend Analytics API `GET /analytics/dashboard` hidup — satu endpoint agregat (`42-API.md` §13, ADR-0026) dengan `from`/`to` (RFC3339 interval tertutup) + `project_id`, izin `report:read` (Admin/Manager), cakupan `44-SECURITY.md` §3.1.3 di kueri (non-Admin hanya project yang diikutinya). KPI 6 (total/active/pending/overdue/avg/revised) + chart 8 (Status Dist, Volume Trend, Approval Trend, Funnel, Pending Aging, Avg Time per Stage estimasi, By Category, Activity Trend) dari transaksi yang sudah ada tanpa migrasi `012`. Wiring `router.go`/`main.go` + handler parse `422` + repository 10 agregat scoped.

### Added

- `backend/internal/{dto/analytics_dto,repository/analytics_repository,service/analytics_service,handler/analytics_handler}.go` (P-056, `T-072`)
- `backend/internal/handler/analytics_handler_test.go` (P-056) — 5 validasi (`from`/`project_id`/`to<from` → `422`, tanpa token `401`, viewer `403`) + 1 sukses (manager `200` dengan `kpis`+`charts`)

### Changed

- `backend/internal/handler/router.go`: `RouterDeps.Analytics`, group `/analytics` `report:read` (P-056)
- `backend/cmd/server/main.go`: wiring `AnalyticsService` + `AnalyticsHandler` (P-056)
- `backend/internal/handler/main_test.go`: `engineParts.analytics` (P-056)
- `AGENTS.md`: `48/55` → `49/56` (endpoint baru) (P-056)
- `docs/progress/{TASKS,STATE}.md`, `docs/progress/prompts/P-056-...md` (P-056)

---

## 2026-09-23 (sesi P-055)

ADR-0026 `ACCEPTED`: Dashboard MVP diikat — KPI 6 + chart 8 dari transaksi yang sudah ada tanpa migrasi `012`, metric dictionary 16 di `52-DASHBOARD-ANALYTICS.md` §7, yang ditahan (department, SLA, review_due/expiry/published, stage history) di `52-*` §4 / Q-DASH-01..04. Kontrak `GET /analytics/dashboard` satu endpoint agregat (`42-API.md` §13, `report:read`, interval tertutup, cakupan di kueri) — alternatif 8 endpoint ditolak. Papan: `T-071` TODO → DONE.

### Added

- `docs/adr/0026-dashboard-mvp-metric-dictionary.md` (P-055) — keputusan MVP tanpa migrasi, KPI 6+chart 8, yang ditahan
- `docs/progress/prompts/P-055-2026-09-23-adr-0026-dashboard-mvp.md` (P-055)

### Changed

- `docs/adr/README.md`: baris 0026 ACCEPTED (P-055)
- `docs/design/52-DASHBOARD-ANALYTICS.md`: header v0.2.0 — diikat ADR-0026 (P-055)
- `docs/progress/TASKS.md`: `T-071` DONE `T-071` (P-055)

---

## 2026-09-23 (sesi P-054)

Telaah `Dashboard.md` vs sistem yang berjalan: `52-DASHBOARD-ANALYTICS.md` baru (telaah 60% READY / 40% ditahan, MVP KPI 6 + chart 8 tanpa migrasi, metric dictionary 16, kontrak `GET /analytics/dashboard` agregat). Seluruh peta desain selaras tanpa mengeksekusi widget: `50-FSD.md` §9 (ringkasan MVP + rujukan `52-*`), `51-UX.md` §6.1 (KPI grid + filter global di URL), `42-API.md` §13 (rencana endpoint `report:read`), `41-DATABASE.md` §2.7 (MVP tanpa kolom baru, backlog field), `30-ARCHITECTURE.md` §3.3, `00-README.md` baris 52, `20-SRS.md` FR-DASH-03/04, `80-ROADMAP.md` Phase 4/5. Papan kerja +4 TODO (`T-071` ADR-0026, `T-072` API, `T-073` frontend, `T-074` backlog penuh) + Q-DASH-01..04 (department, SLA, review_due/published, stage history).

### Added

- `docs/design/52-DASHBOARD-ANALYTICS.md` (P-054) — telaah Dashboard.md §1-§10, pemetaan READY/BUTUH, MVP tanpa migrasi (KPI 6 + chart 8), backlog field, kosakata, kontrak `GET /analytics/dashboard`, tata letak, metric dictionary 16, urutan 4 langkah
- `docs/progress/prompts/P-054-2026-09-23-telaah-dashboard-dan-desain.md` (P-054)

### Changed

- `docs/design/50-FSD.md` §9: widget 7 → KPI 6 + chart 8 MVP + filter/drill-down, rujukan `52-*` (P-054)
- `docs/design/51-UX.md` §6.1: KPI grid 4→2→1 + 4 baris chart + filter global di URL, cakupan di kueri (P-054)
- `docs/design/42-API.md` §13: rencana `GET /analytics/dashboard` (agregat, `report:read`, interval tertutup) (P-054)
- `docs/design/41-DATABASE.md` §2.7: MVP tanpa kolom baru, list field backlog (P-054)
- `docs/design/30-ARCHITECTURE.md` §3.3, `docs/design/00-README.md` baris 52, `docs/design/20-SRS.md` FR-DASH-03/04, `docs/design/80-ROADMAP.md` Phase 4/5 (P-054)
- `docs/progress/TASKS.md`: TODO `T-071`..`T-074` (P-054)
- `docs/progress/OPEN-QUESTIONS.md`: Q-DASH-01..04 (P-054)

---

## 2026-09-23 (sesi P-053)

Halaman **Approvals** (`50-FSD.md` §5.4) berdiri sebagai view dari workflow instance — bukan modul backend baru: daftar ber-tab Pending/Approved/Rejected dari `GET /workflows/instances` dan detail `/approvals/:id` dengan aksi `approve`/`reject`/`request_revision` + `409 WORKFLOW_CONFLICT`. Lapisan data `services/workflows.ts` + `queries/workflows.ts` mengikuti `42-API.md` §5 apa adanya; `responsive-evidence` mencakup `/approvals` (4 halaman × widths, `expectedRanges 0` untuk Approvals). Verifikasi: frontend **282 test / 28 berkas** hijau, backend **270 test**, enam pemeriksa hijau.

### Added

- `frontend/src/services/workflows.test.ts`: 8 test `listWorkflowInstances`/`fetch`/`act`/`resubmit` — meta fallback, scope pending (P-053, `T-070`)
- `frontend/src/pages/Approvals/Approvals.test.tsx`: 9 test tab→status kanonik, scope `assigned_to_me`, filter jeda revisi klien, link detail, meta, empty, 500, axe (P-053)
- `frontend/src/pages/Approvals/ApprovalDetail.test.tsx`: 7 test loading, overdue, jeda revisi, `409`, aksi+comment, axe (P-053)
- `docs/progress/prompts/P-053-2026-09-23-halaman-approvals.md` (P-053)

### Changed

- `docs/progress/{TASKS,STATE,SESSION-LOG,TRACEABILITY}.md`, `CONTINUE.md` (P-053)

---

## 2026-09-23 (sesi P-052)

Penyaring tanggal Documents menjadi **satu kelompok ber-label** (pola P-049) dengan kontrak backend yang sebelumnya tidak ada: `GET /documents` kini menerima `updated_from`/`updated_to` (interval tertutup, RFC 3339 ber-offset, rentang terbalik `422 field=updated_to`) — sisi tanggal Q-016 tertutup, sisa Q-016 hanya category dan owner. Pemeriksa baris penyaring digeneralisasi ke ketiga halaman pada setiap lebar dengan `expectedRanges` per halaman, dan dua cacat yang lahir dari perluasannya ditutup di sesi yang sama. **Tanpa perubahan skema, izin, atau migrasi** (versi goose tetap **11**); kontrak yang berubah hanya `GET /documents`.

### Added

- `backend`: parameter `updated_from`/`updated_to` pada `GET /documents` — handler (`parseDocumentListQuery` + `parseRFC3339Query`), service (`DocumentListFilter`), repository (placeholder `$7/$8` pada `documentListWhere`, ikut mengalir ke tambalan `COUNT(*) OVER()` C-048) (P-052, `T-069`)
- `frontend/src/services/documents.ts`: `updated_from`/`updated_to` pada `DocumentListQuery`, re-ekspor konverter `toRfc3339FromLocal`/`toLocalInputValue` dari modul task, validator `validateUpdatedAtRange` (P-052)
- `frontend/src/pages/Documents/index.tsx`: kelompok `role="group"` berlabel "Rentang pembaruan", satu aksi `applyFilters` untuk pencarian dan rentang, `role="alert"` untuk rentang tidak sah (P-052)
- `frontend/src/pages/Documents/Documents.test.tsx`: tiga test (kontrak kueri instan ber-offset, struktur kelompok ber-label, rentang terbalik ditahan di klien) (P-052)
- `docs/progress/prompts/P-052-2026-09-23-rentang-tanggal-dokumen-satu-kelompok.md` (P-052)

### Changed

- `scripts/responsive-evidence.mjs`: `measureTaskFilters` → `measureFilterRow(formLabel)` untuk ketiga halaman pada setiap lebar; kelompok rentang dicari lewat struktur (bukan id tetap) + `expectedRanges` per halaman; tombol yang kolomnya form dibedakan dari cacat "terangkat" dengan alasan tertulis; kunci laporan `taskFilters` → `filterRows` (P-052)
- `docs/design/42-API.md` §4: parameter + aturan interval tertutup `updated_from`/`updated_to`; paragraf "belum ada" direvisi (P-052)
- `docs/design/50-FSD.md` §4.1: filter Date range kini menunjuk parameternya; Category/Owner ditandai menunggu Q-016 (P-052)
- `docs/design/70-TESTING.md`: §3.14i (bukti P-052) (P-052)
- `docs/progress/{TASKS,STATE,SESSION-LOG,TRACEABILITY}.md`, `CONTINUE.md` (P-052)

### Fixed

- `backend/internal/handler/document_handler_test.go`: fixture memotong `created_at` ke detik sehingga tiga dokumen yang lahir dalam detik yang sama punya tepi identik (dan lebih awal dari `updated_at` ber-mikrodetik) — `RFC3339Nano`; kasus "sebelum semua" `2027` → `2020` (P-052)
- `docs/progress/STATE.md` §3: hitungan test per berkas dan total suite diselaraskan ke repo (15, 270) — tertangkap `check-ledger.sh` (P-052)

## 2026-09-23 (sesi P-051)

Aturan `51-UX.md` §2.1 (sidebar memuat **modul saja**) akhirnya punya penegak mesin: `scripts/check-navigation.sh` menolak entri sidebar yang membawa kueri penyaring atau menunjuk sub-halaman, halaman anak yang induknya tidak ada, path/label ganda, dan daftar menu yang menyimpang dari tabel §2.1 — **dua arah**. Halaman **Audit** (di bawah Reports) karena itu didaftarkan sebagai halaman anak (`subPages`) yang hidup di baris sub-navigasi modul induknya, bukan sebagai menu tambahan. Empat temuan lahir di sesi ini dan semuanya `FIXED`: **C-079** (aturan itu tidak dijaga apa pun — CI tidak punya job frontend, dan pengukur di peramban hanya mencari menu **berkueri**), **C-080** (penegak barunya sendiri melaporkan 25 kegagalan menyesatkan ketika bentuk berkasnya berubah, karena penjaga hampa hanya menghitung jumlah menu), **C-081** (total temuan di dalam laporan audit **sendiri** tertinggal tiga sesi — 75 padahal marker 78 — karena aturan prosa ledger hanya membaca baris yang menyebut `AUDIT-001`) dan **C-082** (suite frontend gagal berpindah-pindah antar-berkas pada kode yang sama karena jendela tunggu bawaan `findBy*` adalah klaim tentang kecepatan mesin; `--maxWorkers=1` hijau penuh pada kode yang sama). **Tidak ada perubahan backend, kontrak, izin, atau skema.**

### Added

| File | Keterangan |
|---|---|
| `scripts/check-navigation.sh` | Pemeriksa batas sidebar: entri berkueri/satu segmen, halaman anak, path/label ganda, penyimpangan dari `51-UX.md` §2.1 (dua arah), dan penjaga bentuk parser — `T-068`, temuan **C-079**/**C-080** — P-051 |
| `docs/progress/prompts/P-051-2026-09-23-pemeriksa-batas-sidebar.md` | Log sesi — P-051 |

### Changed

| File | Keterangan |
|---|---|
| `.github/workflows/ci.yml` | Langkah keenam job `ledger` (`bash scripts/check-navigation.sh`) + komentar kepala yang menjelaskan kelas cacat yang ditahannya — P-051 |
| `frontend/src/config/navigation.ts` | Daftar **`subPages`** (halaman anak ber-`parent`, bukan menu) + `visibleSubPages()` + `subNavFamily()`; sidebar tetap modul saja — `T-067` — P-051 |
| `frontend/src/App.tsx` | Rute pending dibangun dari **kedua** daftar, supaya `/reports/audit` hidup walau modul induknya belum dibangun (R-24) — P-051 |
| `frontend/src/pages/ModulePending.tsx` | Baris sub-navigasi keluarga modul, sehingga halaman anak tetap punya jalan masuk — P-051 |
| `frontend/src/config/navigation.test.ts` | Test model navigasi: kedua daftar terpisah, setiap halaman anak berinduk nyata, menu sidebar satu segmen tanpa kueri — P-051 |
| `frontend/src/components/layout/AppShell.test.tsx` | Test sidebar disesuaikan: `Reports` **tidak** menyala saat `/reports/audit` dibuka, sub-halaman tetap punya jalan masuk — P-051 |
| `frontend/src/test/setup.ts` | `configure({ asyncUtilTimeout: 5000 })` dengan alasannya: bawaan 1000ms adalah klaim tentang kecepatan mesin (**C-082**) — P-051 |
| `frontend/vite.config.ts` | `testTimeout: 20000`, menyertai jendela `findBy*` yang dinaikkan (**C-082**) — P-051 |
| `scripts/check-ledger.sh` | Aturan baru: ringkasan jumlah temuan **di dalam** laporan audit wajib sama dengan marker `audit-summary` (**C-081**), plus pagar bila pola itu hilang — P-051 |
| `README.md` | Pohon §4 memuat `check-navigation.sh` dan §12.3 memuat perintahnya — P-051 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | **§6.5** baru (batas sidebar, kelas C-079) + checklist self-check §8 — P-051 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Perintah + paragraf pemeriksa batas sidebar di §8 — P-051 |
| `docs/design/51-UX.md` | §2.1: batas itu dijaga mesin, dan halaman anak didaftarkan di `subPages` — P-051 |
| `docs/design/70-TESTING.md` | **§3.14h** baru: tabel aturan pemeriksa, tujuh cacat yang disuntikkan, dan pengukuran flakiness suite — P-051 |
| `AGENTS.md` | Blok aturan **batas sidebar** + daftar pemeriksa yang wajib hijau + prosa angka audit (`C-079`..`C-082`) — P-051 |
| `docs/progress/TASKS.md` | `T-067` (halaman anak di sub-navigasi) dan `T-068` (pemeriksa batas sidebar) `DONE` bertanggal dan ber-bukti — P-051 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | **C-079**, **C-080**, **C-081**, dan **C-082** ditambahkan (semuanya `FIXED`); marker `audit-summary` menjadi `82 / 80 / 0 / 2 / 0 / 37` dan paragraf §1 dikoreksi (ia sendiri menulis 75) — P-051 |
| `docs/progress/audits/README.md`, `AGENTS.md`, `docs/progress/STATE.md`, `CONTINUE.md` | Marker `audit-summary` dan prosa hitungannya diselaraskan ke **82/80/0/2** — P-051 |
| `docs/progress/STATE.md`, `docs/progress/SESSION-LOG.md`, `docs/progress/CHANGELOG.md`, `docs/progress/TRACEABILITY.md`, `CONTINUE.md` | Ledger sesi P-051 — P-051 |

## 2026-09-23 (sesi P-050)

Sapuan tata letak per lebar kini berjalan untuk **setiap** halaman, bukan hanya `/projects`: `projects`/`tasks`/`documents` × `375/640/768/1024/1440` = 15 pengukuran tata letak dan 18 pengukuran tema, dengan sidebar, bilah tab, dan baris penyaring Task ikut diperiksa di **setiap** lebar. Perluasan cakupan itu langsung menemukan tiga kelas cacat. **C-076:** kolom penyaring tidak dapat menyusut — min-width otomatis item flex adalah min-content anaknya, dan min-content sebuah `<select>` ditentukan teks pilihan terpanjang (yaitu data pengguna), sehingga `#penyaring-project-dokumen` selebar 403px pada 375px membuat halaman menggulir mendatar 44px, dan halaman Projects menyimpan cacat yang sama meski pilihannya pendek (65px). **C-077:** `tap-target` hanya menetapkan tinggi, sehingga tab sub-halaman "Tim" terukur 43x44px. **C-078:** lima cacat pada alat ukurnya sendiri, termasuk cakupan yang tidak pernah meluas, argumen CLI berspasi yang diam-diam diabaikan, dan jeda tetap yang kalah balapan sehingga pernah melaporkan positif palsu. Perbaikannya: `min-w-0` pada setiap kolom ber-`select`, `tap-target` menetapkan kedua sisi, penantian tata letak yang tenang, dan probe pilihan sengaja panjang agar aturannya tidak bergantung pada data. **Tanpa perubahan backend, kontrak API, izin, atau skema.**

### Added

| File | Keterangan |
|---|---|
| `frontend/src/styles/tap-target.test.ts` | 3 test membaca `tokens.css`: utility menetapkan `min-height` **dan** `min-width` pada kedua rentang lebar, dan tidak mengunci `width`/`height` tetap — P-050 |
| `docs/progress/prompts/P-050-2026-09-23-sapuan-per-lebar-setiap-halaman.md` | Log sesi — P-050 |

### Changed

| File | Keterangan |
|---|---|
| `scripts/responsive-evidence.mjs` | Sapuan per lebar & tema untuk **setiap** halaman; sidebar/tab/penyaring diperiksa di setiap lebar; ambang kesebarisan rentang diturunkan dari kelas `sm:`; probe pilihan panjang pada setiap `select` penyaring (melaporkan pertambahan, bukan luas); penantian "tata letak berhenti berubah" menggantikan jeda tetap; kendali majemuk dipisahkan dan dilaporkan; argumen berspasi diterima; fase yang hampa dilewati dengan alasan tertulis — P-050 |
| `frontend/src/styles/tokens.css` | `@utility tap-target` menetapkan `min-width` juga (44px, 36px di ≥1024px), karena ambangnya kotak dan bukan garis — P-050 |
| `frontend/src/pages/Projects/index.tsx`, `frontend/src/pages/Documents/index.tsx`, `frontend/src/pages/Tasks/index.tsx` | `min-w-0` pada setiap kolom penyaring ber-`select` (kolom rentang tenggat sengaja **tidak**), beserta alasan mengapa ukuran minimum datang dari data pengguna — P-050 |
| `frontend/src/pages/Projects/Projects.test.tsx`, `frontend/src/pages/Documents/Documents.test.tsx`, `frontend/src/pages/Tasks/Tasks.test.tsx` | Satu test per halaman mengunci `min-w-0` pada setiap kolom ber-`select`, dengan penjaga hampa supaya tidak lulus saat tidak ada kolom yang ditemukan — P-050 |
| `docs/design/51-UX.md` | §2.1: kolom penyaring wajib dapat menyusut dan ambang sentuh berupa kotak, beserta pengukuran yang melatarbelakanginya — P-050 |
| `docs/design/70-TESTING.md` | §3.14b (cakupan sapuan yang diperluas), §3.14f (tuntutan rentang diukur di setiap lebar), dan **§3.14g** baru: pengukuran, tiga temuan, lima cacat alat ukur, dan bukti giginya — P-050 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Tiga temuan baru **C-076**, **C-077**, **C-078** (`FIXED`); hitungan **78/76/0/2** — P-050 |
| `docs/progress/audits/README.md`, `AGENTS.md`, `docs/progress/STATE.md`, `CONTINUE.md` | Marker `audit-summary` dan prosa hitungannya diselaraskan ke 78/76/0/2 — P-050 |

---

## 2026-09-23 (sesi P-049)

Sesi ini menutup kekurangan yang dinyatakan sendiri sesi P-047: penyaring **rentang tenggat** di halaman Tasks berdiri sebagai dua kolom terpisah, sehingga saat baris penyaring melipat (`flex-wrap`) batas awalnya dapat jatuh ke baris berbeda dari batas akhirnya dan rentangnya terbaca sebagai dua penyaring yang tidak berhubungan. Kedua isian kini berada di **satu kelompok ber-peran `group` berlabel "Rentang tenggat"** — berdampingan pada lebar lebar, menumpuk di dalam kelompok yang sama pada lebar sempit. Bentuk terakhir itu bukan tafsiran melainkan hasil ukuran: pasangan berdampingan pada 375px menuntut ~400px dan mendorong halaman menggulir mendatar 71px, jadi yang dipertahankan hubungannya, bukan posisinya. Klaimnya berhenti menjadi klaim — `scripts/responsive-evidence.mjs` mengukur pasangan itu per kelompok pada lebar lebar **dan** sempit, dan tiga butirnya terbukti bergigi dengan memasang kembali bentuk lamanya. Satu cacat pada alat ukur itu sendiri ikut ditutup (butir itu mencari isian lewat `aria-label` sehingga berhenti mengukur pada bentuk yang harus ditangkapnya — kelas C-075). **Tanpa perubahan backend, kontrak API, izin, atau skema.**

### Changed

| File | Keterangan |
|---|---|
| `frontend/src/pages/Tasks/index.tsx` | Kedua batas rentang tenggat berpindah ke satu kelompok berlabel "Rentang tenggat": berdampingan pada lebar lebar, menumpuk di dalam kelompok yang sama pada lebar sempit; pemisah "sampai" tetap `aria-hidden` — P-049 |
| `frontend/src/pages/Tasks/Tasks.test.tsx` | Test baru "menempatkan kedua batas rentang di satu kolom yang tidak melipat": keduanya di dalam `role="group"` bernama "Rentang tenggat", barisnya `flex` **tanpa** `flex-wrap`, tiga anak — P-049 |
| `scripts/responsive-evidence.mjs` | Bagian `navigation` mengukur pasangan rentang per kelompok (`sameLine`, `sameGroup`, `contained`) pada lebar lebar **dan** sempit plus gulir mendatar halaman ini; pencarian isian dipindah dari `aria-label` ke `id` dan pesan "tidak ditemukan" dikoreksi — P-049 |
| `docs/design/51-UX.md` | §2.1: aturan bahwa dua kendali yang membentuk satu nilai berdiri sebagai satu kelompok berlabel, beserta alasan mengapa pada lebar sempit yang dituntut kelompoknya, bukan garisnya — P-049 |
| `docs/design/70-TESTING.md` | §3.14f: pengukuran bentuk lama vs bentuk baru pada 1440px/375px dan bukti gigi tiga butir — P-049 |
| `README.md` | Ringkasan frontend menyebut penyaring rentang tenggat sebagai satu kelompok berlabel — P-049 |

---

## 2026-09-23 (sesi P-048)

Sesi ini menutup modul terakhir yang menggantung di Phase 2: **Workflow** hidup dengan sembilan endpoint `42-API.md` §5 — definisi + step, submit, tiga aksi keputusan, re-submit setelah revisi, dan daftar/detail instance ber-cakupan. Guard optimistic ADR-0015 ditegakkan **di kueri** (bukan di memori), jeda revisi dibaca dari **status dokumen** (bukan status instance), dan setiap penolakan terjadi sebelum satu baris pun ditulis sehingga tidak meninggalkan jejak. 29 test menguncinya, dan `scripts/probe-workflow-module.py` membuktikan 46 perilakunya di server nyata dengan lima aktor login sungguhan. Tiga temuan audit ditutup: **C-073** (pseudokode §4.1 memberi Administrator jalan pintas `OR be admin` yang tidak ada di dokumen pengikat), **C-074** (`42-API.md` §5 menyebut "satu-satunya route" untuk himpunan yang dokumen lain sebut dua), dan **C-075** (`check-ledger.sh` melewatkan dua klaim hitungan test §3 karena polanya tidak mengenali bentuk `(N test: …)` — sekaligus mengungkap versi skema yang tertulis 10 padahal 11). **Tidak ada perubahan skema**: versi goose tetap 11.

### Added

| File | Keterangan |
|---|---|
| `backend/internal/model/workflow.go` | Kosakata kanonik modul (status instance, tiga aksi, empat role step), navigasi step maju/mundur dengan batas bawah step 1, deadline dan keterlambatan sebagai turunan — P-048 |
| `backend/internal/model/workflow_test.go` | 6 test kosakata, navigasi step, deadline, dan penanda keterlambatan — P-048 |
| `backend/internal/dto/workflow_dto.go` | Bentuk request/response `42-API.md` §5 (termasuk `responsible_user_ids` pada submit dan `actions` pada detail) — P-048 |
| `backend/internal/repository/workflow_repository.go` | Definisi + step, instance ber-cakupan `WHERE`, `ApplyTransition` ber-guard empat kondisi, riwayat aksi, penanggung jawab per role — P-048 |
| `backend/internal/service/workflow_service.go` | `Submit`, `ExecuteAction` (izin dari isi body + penunjukan step + jeda revisi), `Resubmit`, `List`/`Get`, definisi + step — P-048 |
| `backend/internal/service/workflow_service_test.go` | 18 test: definisi, submit, aksi, konkurensi §3.3, rollback §3.4, rollback satu step, re-submit, cakupan — P-048 |
| `backend/internal/handler/workflow_handler.go` | Sembilan handler + validasi per field dengan pesan `422` berpola modul lain — P-048 |
| `backend/internal/handler/workflow_handler_test.go` | 5 test HTTP: auth, izin definisi, validasi body, validasi kueri, siklus penuh — P-048 |
| `scripts/probe-workflow-module.py` | Probe HTTP nyata 46 asersi (lima aktor, baseline pulih otomatis) — P-048 |
| `docs/progress/prompts/P-048-2026-09-23-modul-workflow-dan-probe-46-pass.md` | Log prompt sesi — P-048 |

### Changed

| File | Keterangan |
|---|---|
| `backend/internal/handler/router.go` | Group `/workflows` + sembilan route beserta izinnya (route aksi hanya `workflow_instance:read`; izin aksi dipilih service) — P-048 |
| `backend/cmd/server/main.go` | Wiring repository/service/handler workflow — P-048 |
| `backend/internal/handler/main_test.go` | Fixture HTTP menyediakan service workflow — P-048 |
| `backend/internal/repository/db.go` | Helper transaksi yang dipakai alur aksi workflow — P-048 |
| `backend/internal/pkg/response/response.go` | Amplop konflik ber-`details` **objek** untuk `409 WORKFLOW_CONFLICT` — P-048 |
| `docs/design/43-WORKFLOW.md` | §4.1: syarat aksi menjadi **izin DAN penunjukan step** (bukan `OR be admin`) — temuan **C-073** — P-048 |
| `docs/design/42-API.md` | §5: route aksi adalah "route **pertama** dari **dua**" yang izinnya bergantung pada isi body (bukan "satu-satunya") — temuan **C-074**; §6 tetap sumber pasangannya — P-048 |
| `docs/design/50-FSD.md` | §8.1: jenis notifikasi `REVIEW_REQUIRED_AGAIN` (siklus re-submit ADR-0016), berbeda dari `APPROVAL_REQUIRED` dan `REVISION_REQUESTED` — P-048 |
| `docs/design/40-TSD.md` | Modul workflow masuk daftar struktur/rute + alasan mengapa izin aksi diperiksa di service, bukan middleware — P-048 |
| `docs/design/70-TESTING.md` | §3.15: bukti `T-064` — 29 test, bukti mutasi guard `version` (beserta koreksi atribusinya), dan ringkasan 46 asersi probe — P-048 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`, `docs/progress/audits/README.md` | **C-073**, **C-074**, dan **C-075** ditambahkan (ketiganya `FIXED`); hitungan temuan **72/70 → 75/73** — P-048 |
| `docs/progress/TRACEABILITY.md` | Tujuh baris `FR-WF-*` → `DONE` dengan berkas, test, dan bukti server nyata; `FR-WF-08` menyatakan batasnya (`reject` belum diuji lewat probe) — P-048 |
| `docs/progress/TASKS.md` | Baris `T-064` (DONE) — P-048 |
| `docs/progress/STATE.md` | Paragraf sesi P-048, baris keadaan (workflow hidup, halaman Approvals menyusul), dan angka suite **239 → 268**; angka audit **75/73**; dua hitungan test §3 dan versi skema dikoreksi (**C-075**) — P-048 |
| `CONTINUE.md` | Header, snapshot, dan papan task menyebut modul Workflow selesai + angka audit 74/72 — P-048 |
| `AGENTS.md` | Modul workflow dinyatakan hidup; blok **aturan modul workflow** yang mengikat (guard empat kondisi, izin DAN penunjukan step, jeda revisi dari status dokumen, dua jalur masuk review, `is_overdue` turunan) — P-048 |
| `README.md` | 41 route (+9 workflow), baris modul Workflow "Selesai", dua skrip probe di pohon dan perintah verifikasi — P-048 |
| `scripts/check-readme-facts.sh` | Ember route `workflows` + baris modul Workflow pada tabel status (tanpa itu, group baru menggagalkan pemeriksaan) — P-048 |
| `scripts/check-ledger.sh` | Pola klaim per-berkas §3 meluas ke bentuk `(N test: …)` yang sebelumnya tidak dikenali sehingga dua klaim lolos tanpa diperiksa — temuan **C-075** — P-048 |
| `scripts/check-api-contract.sh` | Butir baru: klaim jumlah anotasi izin di `AGENTS.md` (`N/M`) dibandingkan dengan hitungan skrip sendiri, sehingga angka itu tidak dapat basi lagi — temuan **C-075** — P-048 |
| `AGENTS.md` | Klaim anotasi izin `40/51` → **`48/55`**; blok **aturan modul workflow** ditambahkan — P-048 |

---

## 2026-09-23 (sesi P-047)

Sesi ini menutup dua keluhan halaman dengan **sebabnya**, dan mengubah ketiga klaimnya menjadi **ukuran di peramban sungguhan**: kolom penyaring Task tidak lagi lebih tinggi dari kolom lain, dan sidebar kembali menjadi daftar modul dengan sub-navigasi di halaman yang memilikinya (`T-063`). Tanpa perubahan backend, kontrak, izin, atau skema.

### Added

| File | Keterangan |
|---|---|
| `docs/progress/prompts/P-047-2026-09-23-sidebar-modul-dan-penyaring-sebaris.md` | Log prompt sesi — P-047 |

### Changed

| File | Keterangan |
|---|---|
| `frontend/src/pages/Tasks/index.tsx` | Catatan sumber pilihan penanggung jawab (dan batas C-063) dipindah dari dalam kolom penyaring ke blok catatan di bawah baris, sehingga kolomnya tidak lagi lebih tinggi dan kontrolnya tidak terangkat dari baris ber-`items-end` — P-047 |
| `frontend/src/config/navigation.ts` | `subItems`/`NavSubItem` dibuang (sidebar memuat modul saja); `requiresExactMatch` diturunkan dari daftar path sehingga `/reports` tidak menyala saat `/reports/audit` dibuka — P-047 |
| `frontend/src/components/layout/Sidebar.tsx` | Kembali merender satu tautan per modul; `sameFilters`/`useLocation`/`Link` dan blok `<ul>` sub-menu dihapus; `end` dihitung dari `requiresExactMatch` — P-047 |
| `frontend/src/pages/Documents/index.tsx` | Baris tab sub-halaman (`Semua`, `Milik saya`, `Pending Review`, `Revision Required`, `Approved`) supaya sub-navigasi tinggal di halaman dan `?view=mine` tetap dapat dibuka (alasan Q-016 tetap terbaca) — P-047 |
| `frontend/src/pages/Projects/index.tsx` | Komentar: Projects tidak butuh baris tab (List = halaman, Create = tombol header) — P-047 |
| `frontend/src/components/layout/AppShell.test.tsx` | Describe `penanda menu aktif` diganti `menu sidebar`: satu tautan per modul, tepat satu menu bertanda aktif, dan menu induk tidak menyala pada path bersarang — P-047 |
| `frontend/src/config/navigation.test.ts` | Describe `bentuk menu`: tanpa `subItems`, aturan pencocokan persis, dan tidak ada dua menu yang saling menutupi — P-047 |
| `frontend/src/pages/Documents/Documents.test.tsx` | Dua test: tab memetakan label ke status kanonik, dan `Milik saya` dapat dibuka dari tab sehingga alasannya terbaca — P-047 |
| `frontend/src/pages/Tasks/Tasks.test.tsx` | Menyesuaikan teks catatan penyaring yang berpindah tempat — P-047 |
| `scripts/responsive-evidence.mjs` | Bagian `navigation`: sidebar (jumlah tautan, tautan berkueri, tautan sub-halaman, satu menu aktif), kesebarisan baris penyaring Task (kolom tidak melanjutkan di bawah kontrolnya), dan tab Documents (dibuka lewat klik sungguhan) — P-047 |
| `docs/design/51-UX.md` | §2.1 ditulis ulang: sidebar memuat modul saja, kolom `Sub-items` diganti kolom halaman + sub-navigasinya, beserta alasan keputusan P-047 — P-047 |
| `docs/design/70-TESTING.md` | §3.14e: bukti navigasi & kesebarisan penyaring sesi ini, termasuk dua pengukuran yang salah rancang sebelum menjadi ukuran yang benar — P-047 |
| `docs/progress/TASKS.md` | Baris `T-063` (DONE) — P-047 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `SESSION-LOG.md`, `AGENTS.md`, `README.md` | Ledger sesi P-047; angka frontend menjadi **240 test / 24 berkas** — P-047 |
| `.freebuff/run.md` | Baseline `versi skema` dikoreksi ke **11**; bagian skrip bukti menyebut bagian `navigation` — P-047 |

### Fixed

| File | Keterangan |
|---|---|
| `frontend/src/pages/Tasks/index.tsx` (tata letak) | Kolom penyaring Penanggung jawab melanjutkan **23px** di bawah kontrolnya, sehingga kontrolnya terangkat dari barisnya; kini **0** kontrol terangkat (terukur) — P-047 |
| `frontend/src/components/layout/Sidebar.tsx` + `frontend/src/config/navigation.ts` | Delapan menu menghasilkan **15** tautan sub-item; kini **8** tautan tanpa satu pun yang membawa kueri penyaring — P-047 |
| `frontend/src/components/layout/Sidebar.tsx` (penanda aktif) | `/reports` dan `/reports/audit` sama-sama memasang `aria-current="page"` karena pencocokan awalan `NavLink` — P-047 |

---

## 2026-09-22 (sesi P-046)

Sesi ini membangun **halaman bisnis ketiga — Tasks (`T-062`)** mengikuti pola yang dua kali terbukti (Projects P-041, Documents P-044), dengan penyaring **tri-state overdue** dan **transisi status** sesuai `42-API.md` §6, dibuktikan probe **41/41 PASS** di server nyata. Tanpa temuan baru.

### Added

| File | Keterangan |
|---|---|
| `frontend/src/services/tasks.ts` | Lapisan layanan modul task (`listTasks`/`fetchTask`/`createTask`/`updateTask`/`completeTask`, kosakata prioritas + label, `validateDueRange` batas inklusif) — P-045/P-046 |
| `frontend/src/queries/tasks.ts` | Kunci kueri terpusat + hook; mutasi detail diambil ulang karena `is_overdue`/`updated_at` dihitung server — P-045/P-046 |
| `frontend/src/pages/Tasks/index.tsx` | Task List `50-FSD.md` §6.1: sub-halaman sebagai penyaring, tri-state overdue, rentang tenggat RFC 3339, paginasi dari `meta` — P-045/P-046 |
| `frontend/src/pages/Tasks/TaskDetail.tsx` | Task Detail §6.3: tabel transisi endpoint+izin, aksi menurut status berjalan, bagian "belum dibangun" — P-045/P-046 |
| `frontend/src/pages/Tasks/CreateTaskDialog.tsx` | Dialog buat task: project → anggota + dokumen project — P-045/P-046 |
| `frontend/src/services/tasks.test.ts`, `frontend/src/pages/Tasks/{Tasks,TaskDetail}.test.tsx` | Test lapisan layanan, halaman daftar (tiga keadaan penyaring), dan detail (ketiga transisi + `409`) — P-045/P-046 |
| `scripts/probe-task-module.py` | Sesi probe HTTP nyata modul task (16 kelompok, 41 asersi, aktor kedua viewer, baseline pulih otomatis) — P-045/P-046 |
| `docs/progress/prompts/P-046-2026-09-22-halaman-tasks-dan-probe-41-pass.md` | Log prompt sesi — P-046 |

### Changed

| File | Keterangan |
|---|---|
| `frontend/src/config/navigation.ts`, `frontend/src/App.tsx` | `Tasks` menjadi `ready`; rute `/tasks` + `/tasks/:id` — P-045/P-046 |
| `frontend/src/queries/projects.ts`, `frontend/src/queries/documents.ts` | Hook pendukung dialog buat task (anggota project, dokumen project) — P-045/P-046 |
| `frontend/src/components/layout/Sidebar.tsx`, `frontend/src/components/layout/AppShell.test.tsx` | Penyesuaian laci menu kecil + testnya — P-045/P-046 |
| `frontend/src/App.test.tsx`, `README.md` | Rute & status halaman diperbarui (tiga halaman bisnis) — P-045/P-046 |
| `docs/progress/TASKS.md` | Baris `DONE` `T-062` — P-046 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Paragraf sesi P-046, angka 234 test/24 berkas, keadaan halaman — P-046 |
| `docs/progress/SESSION-LOG.md`, `docs/progress/TRACEABILITY.md` | Entri P-046; baris UI halaman Tasks — P-046 |
| `docs/design/70-TESTING.md` | §3.14d: bukti probe 41 asersi + test halaman Tasks — P-046 |
| `docs/progress/CHANGELOG.md` | Entri ini — P-046 |

---

## 2026-09-22 (sesi P-045)

Sesi ini menyelesaikan **ledger P-044** yang tertinggal dan, dari verifikasi jalur nyatanya, menemukan
**C-072**: unggahan `.txt` dan `.csv` — dua dari delapan jenis berkas yang dijanjikan `50-FSD.md` §4.2 —
**selalu** dibalas `422`, karena MIME hasil `http.DetectContentType` (`text/plain; charset=utf-8`)
dibandingkan utuh terhadap daftar tertutup yang menulis `text/plain`.

### Added

| Berkas | Isi | Prompt |
|---|---|---|
| `docs/progress/prompts/P-044-2026-09-22-halaman-documents-dan-lapisan-data.md` | Log prompt P-044 yang tertinggal (ditulis menyusul, dengan catatan provenans bahwa isinya diambil dari berkas di disk dan bukti yang dijalankan ulang) | P-045 |
| `docs/progress/prompts/P-045-2026-09-22-mime-berkas-dan-ledger-p044.md` | Log sesi ini | P-045 |
| `backend/internal/service/document_service_test.go::TestDocumentUploadAcceptsDetectedMimeWithParameters` | Lima golongan berkas diunggah dengan MIME **hasil deteksi byte**, bukan MIME yang ditulis test | P-045 |
| `backend/internal/handler/document_handler_test.go::TestUploadAcceptsDocumentedTextTypesHTTP` | Unggahan multipart `.txt`/`.csv` sungguhan → `201`, `mime_type`, `Content-Type` unduhan, isi utuh | P-045 |

### Changed

| Berkas | Perubahan | Prompt |
|---|---|---|
| `backend/internal/service/document_service_upload.go` | `normalizeMimeType` membuang parameter media type sebelum pencocokan; `validateUpload` memakainya; komentar menunjuk **C-072** | P-045 |
| `backend/internal/handler/document_handler_test.go` | Kasus `palsu.pdf` berisi teks yang dituntut `422` diganti `.pdf` berisi ZIP + `.sh`, dengan alasan penggantiannya di komentar test | P-045 |
| `frontend/src/services/documents.ts`, `frontend/src/services/documents.test.ts` | Rujukan temuan pada komentar arsip dikoreksi dari `C-067` menjadi **`C-070`** | P-045 |
| `docs/design/42-API.md` §4 | Bentuk amplop `POST /documents`, `POST /documents/:id/archive` (dan perbedaan `POST /documents/:id/upload`) ditulis eksplisit beserta contoh JSON — klaim yang sebelumnya hanya ada di baris audit **C-070** | P-044/P-045 |
| `docs/design/44-SECURITY.md` §4.2 | Butir MIME: hasil deteksi dibandingkan **setelah** parameternya dibuang; dinyatakan bahwa kedua penjaga tidak berpasangan (**C-072**) | P-045 |
| `docs/design/40-TSD.md` | Paragraf modul dokumen: normalisasi MIME dan dua penjaga yang berdiri sendiri | P-045 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Baris **C-072** + catatan kelasnya; marker & ringkasan `72 temuan / 70 FIXED / 0 APPROVED / 2 OPEN`; riwayat sesi P-043/P-044/P-045 | P-045 |
| `docs/progress/audits/README.md` | Marker & prosa diselaraskan ke tabel audit (`72`, `70 FIXED`, rentang `C-038..C-072`) | P-045 |
| `docs/progress/TASKS.md` | Baris `DONE` `T-059` (halaman Documents), `T-060` (C-070/C-071), `T-061` (C-072) | P-044/P-045 |
| `docs/progress/STATE.md` | Paragraf sesi P-044 & P-045; marker audit; versi skema **11**; baris modul dokumen (uji `.txt`/`.csv`); baris frontend (halaman Documents, 21 berkas / 188 test); total suite **239** | P-045 |
| `CONTINUE.md` (root) | Blok §0 (task terakhir, task aktif, audit, next action) + pointer `P-046` | P-045 |
| `AGENTS.md` | Angka audit; versi skema & rentang migrasi (`001`-`011`, berikutnya `012`); aturan MIME **C-072**; keadaan halaman frontend | P-045 |
| `docs/progress/TRACEABILITY.md` | Baris UI halaman Documents terhadap requirement modul dokumen | P-044 |
| `docs/design/70-TESTING.md` | §3.14c bukti halaman Documents + C-070/C-071/C-072 | P-044/P-045 |
| `.freebuff/run.md` | Rentang migrasi `001`-`011`; catatan bahwa server backend dijalankan lewat `launchctl` karena `nohup … disown` di-reap runner | P-045 |

## 2026-09-22 (sesi P-044)

Sesi ini membangun **halaman bisnis kedua** di frontend (`Documents`) dan menemukan dua cacat dari
menjalankannya: **C-070** (kontrak arsip tanpa bentuk amplop) dan **C-071** (pesan penjaga append-only
menyebut tabel yang salah). Ledger sesi ini diselesaikan pada P-045.

### Added

| Berkas | Isi | Prompt |
|---|---|---|
| `frontend/src/services/documents.ts`, `frontend/src/queries/documents.ts` | Lapisan data `42-API.md` §4: daftar, buat, unggah multipart, unduh blob, daftar versi, arsip | P-044 |
| `frontend/src/pages/Documents/index.tsx`, `CreateDocumentDialog.tsx`, `UploadVersionDialog.tsx`, `FilePicker.tsx`, `DocumentDetail.tsx` | Daftar ber-penyaring (status/project/search di URL), dialog unggah dua langkah, unggah versi baru, detail dokumen | P-044 |
| `frontend/src/utils/download.ts` | Penyimpan blob + nama berkas dari `Content-Disposition` | P-044 |
| `frontend/src/services/documents.test.ts`, `src/utils/download.test.ts`, `src/pages/Documents/Documents.test.tsx`, `src/pages/Documents/DocumentDetail.test.tsx` | Test lapisan data, util, dan halaman | P-044 |
| `backend/internal/migration/011_append_only_message_names_table.sql` | Fungsi penjaga append-only ditulis ulang memakai `TG_TABLE_NAME` (**C-071**) | P-044 |

### Changed

| Berkas | Perubahan | Prompt |
|---|---|---|
| `frontend/src/config/navigation.ts`, `src/App.tsx` | `Documents` menjadi `ready` (task `T-059`) + rute `/documents` dan `/documents/:id` | P-044 |
| `frontend/src/types/api.ts`, `src/utils/format.ts`, `src/services/http.ts` | Tipe dokumen & versi, util ukuran berkas dan nilai kosong (`EMPTY_VALUE`/`EMPTY_DATE`), penyesuaian pemetaan galat | P-044 |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Tab Documents mengarah ke halaman nyata, bukan `ModulePending` | P-044 |
| `backend/internal/migration/audit_append_only_test.go` | `TestAppendOnlyMessageNamesTheOffendingTable` mengunci pesan trigger per tabel (**C-071**) | P-044 |
| `docs/design/42-API.md` §4 | Amplop arsip ditulis eksplisit (**C-070**) | P-044 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Baris **C-070** dan **C-071** | P-044 |

## 2026-09-22 (sesi P-043)

Aturan dan skill **antislop diterapkan pada frontend yang sudah berdiri**, dan yang pertama kali
muncul dari penerapan itu bukan tambahan fitur melainkan **tiga ketidakbenaran**: dokumen tata letak
menjanjikan bentuk sidebar yang tidak ada di layar (C-067), dua test baru mengklaim menguji perilaku
yang tidak mereka kunci (C-068), dan keputusan R-02 lebih luas daripada yang diperiksa mesin sehingga
tiga teks pengguna memang masih memuat em dash (C-069). Ketiganya ditutup di sesi yang sama. Menu di
layar sempit diperbaiki menjadi laci modal yang sungguhan (menutupi, bukan mendorong), dan klaim tata
letaknya kini **diukur** di Chrome sungguhan oleh `scripts/responsive-evidence.mjs` — alat baru yang
giginya dibuktikan dengan enam cacat yang disuntikkan sementara.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `frontend/src/hooks/useMediaQuery.ts` | Tiga titik henti bernama dibaca lewat `useSyncExternalStore` (bukan state yang disalin di effect) + `useMediaQueryEnter` untuk state yang hanya berlaku di layar sempit | P-043 |
| `frontend/src/hooks/useModalLayer.ts` | Perilaku lapisan modal (fokus awal, jebakan Tab, Escape, kunci gulir, fokus kembali) dipakai bersama laci menu dan `Dialog` | P-043 |
| `frontend/src/components/layout/AppShell.test.tsx` | Perilaku laci menu: kunci gulir, `inert`, fokus, Escape, latar penutup, dan pelebaran jendela saat laci terbuka | P-043 |
| `frontend/src/components/common/tapTarget.test.tsx` | Setiap kontrol antarmuka memakai utility `tap-target` (ukuran 44/36px tidak dapat diukur jsdom, jadi yang dikunci adalah pemakaiannya) | P-043 |
| `scripts/responsive-evidence.mjs` | Pengukur tata letak di Chrome sungguhan lewat protokol DevTools (tanpa dependensi baru, tidak di CI): gulir mendatar, ukuran target sentuh per ambang lebar, state sidebar, perilaku laci, kedua tema | P-043 |
| `docs/progress/prompts/P-043-2026-09-22-antislop-frontend-dan-laci-menu.md` | Log sesi + laporan Delivery Gate | P-043 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `frontend/src/components/layout/AppShell.tsx` | Tiga state lebar (laci < 768px, kolom kompak 768-1023px, kolom penuh ≥ 1024px); laci menjadi lapisan modal dengan latar ber-penanda `data-drawer-backdrop`; perbaikan tipe `panelRef` (error `tsc`) dan state yang tidak lagi diset di dalam effect (dua error `eslint`) | P-043 |
| `frontend/src/components/layout/Header.tsx` | Target sentuh pada tombol Menu/tema/akun, label tema pendek di layar sempit, nama akun dipotong agar header tidak melebar | P-043 |
| `frontend/src/components/layout/Sidebar.tsx` | `tap-target` pada seluruh tautan dan sub-tautan, `data-autofocus` untuk item pertama saat laci dibuka, dan tutup laci saat item dipilih | P-043 |
| `frontend/src/components/common/Dialog.tsx` | Logika modal yang terduplikasi diganti `useModalLayer` | P-043 |
| `frontend/src/components/common/{Button,Field,DataTable,States}.tsx`, `frontend/src/styles/tokens.css` | Utility `tap-target` dipakai seragam oleh kontrol dan utility-nya didokumentasikan (44px layar sentuh, 36px desktop) | P-043 |
| `frontend/src/pages/{Login/Dashboard/Projects}/*` | Penyesuaian tata letak sempit dan pemakaian placeholder yang menerangkan keadaan | P-043 |
| `frontend/src/utils/format.ts` | `EMPTY_VALUE`/`EMPTY_DATE` + `formatTimestamp` mengembalikan kalimat, bukan em dash (R-02/R-27) | P-043 |
| `frontend/src/test/a11y.test.tsx` | axe dijalankan juga atas laci menu yang **terbuka** | P-043 |
| `scripts/check-antislop-refs.sh` | Butir 8 baru: em dash pada teks yang dibaca pengguna (`frontend/src`, tanpa komentar) | P-043 |
| `scripts/check-readme-facts.sh` | Pohon folder diperiksa untuk **semua** berkas `scripts/` (bukan hanya `*.sh`), supaya pemeriksa berbasis Node tidak jadi lubang baru kelas C-066 | P-043 |
| `docs/design/51-UX.md` | §8: tiga state lebar yang berlaku + perilaku laci + alasan penyimpangan state tengah (C-067); §9: dua register target sentuh beserta pemeriksanya | P-043 |
| `docs/design/70-TESTING.md` | §3.14b baru: bukti tata letak per lebar, perilaku laci, pengukuran tema, dan gigi pemeriksa yang dibuktikan | P-043 |
| `docs/design/01-AGENT-WORKFRAME.md`, `AGENTS.md`, `README.md`, `.freebuff/run.md` | Keputusan R-02/R-03 disempitkan ke yang benar-benar diperiksa; perintah pengukur masuk daftar verifikasi; run doc ditulis ulang (frontend + backend + cara membersihkan `login_attempts`) | P-043 |
| `docs/progress/{TASKS,OPEN-QUESTIONS,CHANGELOG,SESSION-LOG,STATE}.md`, `CONTINUE.md`, `docs/progress/audits/{AUDIT-001-…,README}.md` | Ledger sesi: `T-056`/`T-057`/`T-058` DONE, **Q-025**/**Q-026** dicatat, dan temuan **C-067/C-068/C-069** (`FIXED`) — angka audit menjadi **69 / 67 FIXED / 0 APPROVED / 2 OPEN** | P-043 |

### Removed

| File | Alasan | Prompt |
|---|---|---|
| (tidak ada berkas dihapus) | Sesi ini menyunting berkas yang ada dan menambah lima berkas baru | P-043 |

---

## 2026-09-22 (sesi P-042)

Sistem aturan **antislop** diselaraskan dengan rilis resminya dan berhenti disalin: lima skill yang
sudah lama didaftarkan `AGENTS.md` **benar-benar dipasang** (C-064), core di root diganti berkas
pristine dari **tag rilis `v3.2.12`** (C-065), salinan daftar aturan dan Delivery Gate di dokumen
desaian **dihapus** dan diganti penunjuk, dan sebuah pemeriksa baru menahan kelas cacatnya di CI.
Tidak ada kode backend/frontend maupun migrasi yang disentuh. Atas **izin eksplisit user** (Q-003),
berkas pihak ketiga diunduh agen dari repo `miqdadbadjuber/anti-slop`; izin itu dicatat sebagai
**satu kali** di ADR-0025 (butir 2 ADR-0006 diamandemen: agen tetap dilarang mengunduh atas
inisiatif sendiri).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `skills/README.md` | Provenans berkas pihak ketiga: sumber, tag rilis, `sha256` 9 berkas, lisensi MIT, cara memperbarui, dan apa yang diperiksa mesin | P-042 |
| `skills/antislop/SKILL.md`, `skills/antislop-ui/SKILL.md`, `skills/antislop-copywriting/SKILL.md`, `skills/antislop-layoutmobile/SKILL.md`, `skills/antislop-code/SKILL.md` | Lima skill yang didaftarkan `AGENTS.md` sejak P-037 tetapi tidak ada di disk (C-064); disalin byte-identik dari tag `v3.2.12` | P-042 |
| `skills/antislop-human/SKILL.md`, `skills/antislop-human/contrast-check.py`, `skills/LICENSE-antislop` | Skill aksesibilitas + alat pemeriksa kontras upstream + salinan lisensi MIT pihak ketiga | P-042 |
| `scripts/check-antislop-refs.sh` | Pemeriksa rujukan antislop (7 pemeriksaan: daftar aturan, rujukan `R-XX`, path skill dua arah, `sha256` vs tabel provenans, salinan core identik, klaim rentang, sidik jari aturan tersalin) | P-042 |
| `docs/adr/0025-skill-antislop-terpasang-dan-dipin.md` | Keputusan: pin ke tag rilis, lisensi MIT, sumber aturan tunggal `antislop.md`, dan amandemen butir 2 ADR-0006 | P-042 |
| `docs/progress/prompts/P-042-2026-09-22-skill-antislop-terpasang-dan-dipin.md` | Log sesi | P-042 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `antislop.md` | Diganti berkas pristine dari tag `v3.2.12`: berkas sebelumnya varian lama yang berbeda dari salinan Gate di dokumen desain (C-065) | P-042 |
| `AGENTS.md` | Blok penunjuk antislop kini sesuai kenyataan (path nyata + versi dipin), ditambah aturan anti-drift dan perintah pemeriksa kelima; salinan angka audit diselaraskan ke `65 / 63 FIXED / 0 APPROVED / 2 OPEN` | P-042 |
| `.github/workflows/ci.yml` | Langkah kelima di job `ledger` | P-042 |
| `docs/design/01-AGENT-WORKFRAME.md` | §3.2 dan §5.3: tabel 23 aturan + blok empat Gate **dihapus**, diganti penunjuk ke `antislop.md` dan keputusan proyek per nomor; §6 pohon folder memuat `skills/` dan lima `scripts/` yang sebenarnya | P-042 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | §6.4 baru (rujukan antislop & berkas pihak ketiga — kelas cacat C-064/C-065) dan checklist §8 | P-042 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md`, `docs/design/90-AGENT-GUIDE.md`, `README.md` | Perintah `bash scripts/check-antislop-refs.sh` masuk daftar verifikasi wajib, dengan penjelasan apa yang ditahan | P-042 |
| `docs/adr/README.md` | Indeks ADR-0025; baris ADR-0006 diberi penanda amandemen butir 2 | P-042 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-003 `RESOLVED`: skill terpasang dan dipin; izin unduh bersifat satu kali | P-042 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`, `docs/progress/audits/README.md` | **C-064** dan **C-065** ditambahkan (keduanya `FIXED`); marker `audit-summary` menjadi `65 / 63 / 0 / 2 / 0 / 37` | P-042 |
| `docs/progress/TASKS.md` | `T-054` `DONE` | P-042 |
| `README.md` §4/§12.3/§13 | Changed (**P-042**): pohon folder memuat direktori **`skills/`** dan **kelima** skrip beserta fungsinya (sebelumnya dua — temuan **C-066**), §12.3 menambah `check-antislop-refs.sh` sebagai perintah verifikasi wajib, dan §13 mencatat satu pengecualian yang mudah membingungkan: `skills/LICENSE-antislop` **ada**, tetapi itu salinan izin MIT pihak ketiga (ADR-0025), bukan lisensi proyek ini yang masih belum ditetapkan | P-042 |
| `scripts/check-readme-facts.sh` | Changed (**P-042**): **§6** baru membandingkan daftar berkas di pohon README dengan isi `scripts/` dan `skills/` — dua arah (yang ada wajib disebut, yang disebut wajib ada). Pemeriksa itu kini **37 fakta**, naik dari 25; gigi dibuktikan dengan dua cacat sementara | P-042 |
| `scripts/check-doc-links.sh` | Changed (**P-042**): `guide.md` masuk `ABSENT_DOCS` (panduan upstream yang sengaja tidak disalin) dan isi `skills/` selain `README.md` dilewati — berkas pihak ketiga yang rujukannya menunjuk tata letak repo upstream (varian Claude, Gemini) dan keasliannya dikunci `sha256`, sehingga tidak boleh ditambal dengan menyunting salinannya | P-042 |
| `docs/progress/STATE.md`, `docs/progress/SESSION-LOG.md`, `CONTINUE.md` | Ledger sesi: angka audit, file penting, posisi, dan snapshot §0 | P-042 |

## 2026-09-21 (sesi P-041)

Halaman bisnis **pertama** berdiri: **Projects** (`50-FSD.md` §3 di atas endpoint `42-API.md` §3 yang
sudah hidup), beserta lapisan data **TanStack Query** yang `30-ARCHITECTURE.md` §2.1 janjikan untuk
"halaman data pertama". Tidak ada berkas backend, migrasi, maupun dependency yang disentuh —
`@tanstack/react-query` sudah terpasang sejak kerangka P-037 (Q-022 butir 6), jadi sesi ini tidak
menyentuh jaringan. Verifikasi menyeluruh dijalankan lebih dulu dan hijau, sehingga yang dikerjakan
adalah pekerjaan baru, bukan perbaikan. Sesi ini juga menemukan **C-063** (`OPEN`): `50-FSD.md` §3.2
menuntut dropdown `Owner` yang wajib, padahal tidak ada satu pun endpoint yang dapat menyebutkan
daftar pengguna — batas itu **ditampilkan di layar** dan dikunci test, bukan dikarang; pertanyaannya
**Q-024**.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `frontend/src/services/projects.ts` | Lapisan API modul project (list/detail/create/update/archive/anggota) — satu tempat yang tahu bentuk `42-API.md` §3 | P-041 |
| `frontend/src/services/projects.test.ts` | 9 test bentuk permintaan dan penguraian galat validasi server | P-041 |
| `frontend/src/queries/client.ts` | Konfigurasi klien TanStack Query di satu tempat | P-041 |
| `frontend/src/queries/projects.ts` | Kunci kueri terpusat + hook baca/tulis dengan invalidasi cache | P-041 |
| `frontend/src/components/common/Dialog.tsx` | Primitive dialog aksesibel (fokus awal, jebakan fokus, Escape, `aria-modal`) dipakai dua dialog | P-041 |
| `frontend/src/pages/Projects/index.tsx` | Daftar project: penyaring dari URL, paginasi dari `meta`, tiga keadaan, aksi arsip berkonfirmasi | P-041 |
| `frontend/src/pages/Projects/CreateProjectDialog.tsx` | Form buat project dengan validasi §3.2 dan pemetaan galat server per-field | P-041 |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Detail project: metadata dari server, anggota, tab yang menyatakan dirinya belum dibangun | P-041 |
| `frontend/src/pages/Projects/Projects.test.tsx` (15), `CreateProjectDialog.test.tsx` (8), `ProjectDetail.test.tsx` (9) | Perilaku halaman dikunci mesin, termasuk `axe-core` atas halaman dan atas dialog arsip yang terbuka | P-041 |
| `frontend/src/test/render.tsx` | Helper render ber-provider supaya test halaman tidak menyalin penyiapan | P-041 |
| `docs/progress/prompts/P-041-2026-09-21-halaman-projects-dan-lapisan-data.md` | Log sesi, termasuk Design Read halaman dan bukti server nyata | P-041 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `frontend/src/App.tsx`, `frontend/src/main.tsx`, `frontend/src/config/navigation.ts` | Rute `/projects` + `/projects/:id` hidup, provider kueri dipasang, menu Projects tidak lagi menunjuk halaman yang belum ada | P-041 |
| `frontend/src/pages/Dashboard/index.tsx` | Modul Projects dihitung sebagai modul yang **sudah** ada (R-18: tanpa klaim kosong) | P-041 |
| `frontend/src/components/common/DataTable.tsx`, `frontend/src/components/common/Field.tsx`, `frontend/src/utils/format.ts` | Dukungan kolom aksi/keadaan kosong, pesan galat per-field dari server, dan formatter tanggal untuk kolom tabel | P-041 |
| `frontend/src/App.test.tsx`, `frontend/src/test/a11y.test.tsx` | Menyesuaikan rute baru dan menambah pemeriksaan `axe` atas halaman Projects | P-041 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`, `docs/progress/audits/README.md`, `AGENTS.md`, `docs/progress/STATE.md`, `CONTINUE.md` | Temuan **C-063** ditambahkan (`OPEN`) dan seluruh angka audit diselaraskan ke `63 / 61 FIXED / 0 APPROVED / 2 OPEN` (marker `audit-summary`) | P-041 |
| `docs/progress/TASKS.md` | **`T-053`** `DONE` + catatan `C-063` pada baris `T-017` | P-041 |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-024** dicatat (cara klien memilih pengguna untuk field `Owner`) | P-041 |
| `docs/progress/TRACEABILITY.md` | `FR-PROJ-01`..`FR-PROJ-07` mendapat kolom bukti **lapisan UI** + catatan bahwa P-041 tidak menambah requirement baru | P-041 |
| `docs/design/70-TESTING.md` | **§3.14a** baru: yang dikunci test halaman Projects, bukti server nyata, dan batas jujurnya | P-041 |
| `docs/progress/SESSION-LOG.md` | Entri sesi ini | P-041 |

## 2026-09-21 (sesi P-040)

Gap terakhir yang dapat dikerjakan agen tanpa keputusan siapa pun ditutup: **`T-024`** — 10 endpoint
yang belum punya izin di `42-API.md` §5/§8/§11 (workflow definitions, notifications, admin) — dan
sekaligus **mekanisme rawatnya diganti**. Angka "45/55 endpoint" selama enam sesi ditulis ulang dari
ingatan dan pernah salah (C-055), dan pasangan izin karangan pernah lolos ke draf §4
(`document_version:read`, Q-016). Sejak sesi ini `scripts/check-api-contract.sh` membaca matriks
§3.1.2 `44-SECURITY.md` sebagai sumber kebenaran dan menahan tiga kelas cacat: endpoint tanpa izin,
pasangan yang tidak ada di matriks, dan route yang memakai pasangan di luar matriks. Empat
pemeriksa dokumen kini berjalan di CI. Tidak ada kode produksi yang berubah; `44-SECURITY.md` juga
**tidak** diubah — tidak ada pasangan izin baru yang ditambahkan.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `scripts/check-api-contract.sh` | Pemeriksa anotasi izin (110 pemeriksaan): setiap endpoint di `42-API.md` punya izin terbaca (baris `Izin:` atau baris tabel izin bab), setiap pasangan pada anotasi/tabel ada di matriks `44-SECURITY.md` §3.1.2 (44 pasangan), dan setiap `RequirePermission` di `router.go` memakai pasangan matriks (18 pasangan) | P-040 |
| `docs/progress/prompts/P-040-2026-09-21-anotasi-izin-endpoint-dan-pemeriksanya.md` | Log sesi, termasuk verifikasi menyeluruh dan tiga cacat yang disuntikkan sementara | P-040 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `docs/design/42-API.md` §5/§8/§11 | 10 baris `Izin:` ditambahkan dari matriks §3.1.2 (`workflow_definition:read`/`manage`, `notification:read`/`update`, `user:read`/`create`/`update`, `role:read`, `setting:manage`) beserta alasan pasangan yang tidak jelas; butir prosa `PATCH /admin/users/:id` diangkat menjadi baris `Izin:`; kalimat penjelas `comment:update` ditulis ulang agar tidak terbaca sebagai klaim pasangan — temuan `T-024`; log `prompts/P-040-2026-09-21-anotasi-izin-endpoint-dan-pemeriksanya.md` | P-040 |
| `.github/workflows/ci.yml` | Langkah keempat job `ledger` + komentar kepala diperluas (empat pemeriksa) | P-040 |
| `AGENTS.md`, `README.md` §12.3, `docs/design/12-DEVELOPMENT-WORKFLOW.md` §8 | Perintah verifikasi baru + pernyataan bahwa izin endpoint tidak ditulis dari ingatan | P-040 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` §6.3/§8 | Aturan baru: izin endpoint berasal dari matriks, tiga aturan pemeriksa, dan cara menulis kalimat tentang pasangan yang **tidak** ada (tanpa token `resource:action`) | P-040 |
| `docs/progress/TASKS.md` | `T-024` dipindah dari `TODO` ke `DONE` dengan bukti; **`T-052`** baru (pemeriksa kontrak izin) `DONE` | P-040 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md`, `docs/progress/TRACEABILITY.md` | Ledger diselaraskan: status `T-024`/`T-052`, ringkasan sesi, bukti `FR-ROLE-03` | P-040 |

---

## 2026-09-21 (sesi P-039)

Angka & versi di `README.md` kini diperiksa mesin terhadap repo. Pemicunya dua temuan berturut-turut di
berkas yang paling sering dibaca orang luar: jumlah route yang ditulis **30** padahal router memuat **32**
(**C-061**, P-038), lalu baris "Teknologi" yang masih menulis **React 18** dan "Belum diinisialisasi"
sesudah `frontend/` berdiri (**C-062**). Keduanya lolos dari proses manual — ingatan, `grep` sesekali,
bahkan daftar berkas per sesi — sehingga yang ditambahkan bukan kebiasaan baru melainkan
**pemeriksa**: `scripts/check-readme-facts.sh`. Pemeriksa itu **gagal pada percobaan pertama** terhadap
README yang sudah ada, dan kegagalan itu sendiri yang menemukan C-062. Tidak ada perubahan kode
produksi; hanya skrip, README, dan dokumen aturan.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `scripts/check-readme-facts.sh` | Menghitung fakta README dari sumbernya: jumlah route + rincian per modul + endpoint per modul (`backend/internal/handler/router.go`), versi Go/Gin/pgx/viper/goose (`backend/go.mod`), versi React/Vite/Tailwind/TypeScript (`frontend/package.json`), rentang migrasi (`backend/internal/migration/`), dan klaim "belum ada" atas path yang sudah berdiri. 25 fakta; `readme-facts OK` (exit 0) atau `FAIL` beserta nomor baris (exit 1) | P-039 |
| `docs/progress/prompts/P-039-2026-09-21-pemeriksa-kesegaran-angka-readme.md` | Log sesi, termasuk lima cacat yang disuntikkan sementara untuk membuktikan pemeriksanya berpunya gigi | P-039 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `README.md` §2/§12.3 | Baris "Frontend (rencana) | React 18 + TypeScript + Vite + TailwindCSS | Belum diinisialisasi" diganti keadaan sebenarnya (`React 19 + TypeScript 5.9 + Vite 8 + Tailwind v4`) — temuan **C-062**; blok perintah verifikasi §12.3 menambah `check-readme-facts.sh` beserta alasan mengapa angkanya dibaca dari sumber | P-039 |
| `.github/workflows/ci.yml` | Job `ledger` menjalankan `check-readme-facts.sh` sebagai langkah ketiga, dan komentar kepala menjelaskan kelas cacat yang ditahannya (C-044/C-055/C-057/C-061) | P-039 |
| `AGENTS.md` | Baris kewajiban "sebelum menutup sesi" memuat perintah baru; paragraf baru menjelaskan bahwa angka README diperiksa `check-readme-facts.sh` dan bahwa skripnya **tidak** boleh dilunakkan agar lulus | P-039 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` §6.2/§8 | Aturan baru §6.2 (tabel fakta README → sumber kebenarannya) + butir checklist penutup sesi | P-039 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` §8 | Perintah baru pada blok verifikasi dokumen + paragraf penjelasan hasil/kelas cacat yang ditahan | P-039 |
| `docs/design/90-AGENT-GUIDE.md` §verifikasi | Perintah `check-readme-facts.sh` beserta ringkasan yang diperiksanya | P-039 |
| `docs/progress/README.md` | Butir 7 ("ledger diperiksa mesin") menunjuk pemeriksa kedua untuk kelas serupa di luar ledger | P-039 |
| `docs/progress/audits/AUDIT-001-...md` + `audits/README.md`, `TASKS.md`, `STATE.md`, `CONTINUE.md`, `AGENTS.md`, `CHANGELOG.md`, `SESSION-LOG.md` | Ledger diselaraskan: **C-062** baru (FIXED di sesi yang sama), marker audit 61 → **62** temuan (61 FIXED / 0 APPROVED / 1 OPEN), **`T-051`** DONE | P-039 |

---

## 2026-09-21 (sesi P-038)

`README.md` mendapat bagian **Kontribusi**, dan status lisensi dicatat sebagai keputusan pemilik yang
masih tertunda. User meminta berkas `LICENSE` + bagian kontribusi "setelah pemilik proyek memutuskan
lisensinya"; karena lisensi tidak pernah ditetapkan di dokumen mana pun (dan `README.md` sendiri
melarang menganggap proyek ini open source), agen bertanya lebih dulu, dan pemilik memilih **menunda**
keputusan lisensi sambil tetap menjawab pemegang hak ciptanya (**BSA**). **Berkas `LICENSE` tidak
dibuat**, `frontend/package.json` tetap tanpa field `license`, dan ketidakadaan itu ditampilkan di pohon
`README.md` §4 serta dicatat di `OPEN-QUESTIONS.md` **Q-023** dengan status jujur (BLOCKING untuk
distribusi, NON-BLOCKING untuk pengembangan). Sesi yang sama menemukan dan menutup temuan **C-061**:
enam klaim basi di README, termasuk `frontend/` yang masih disebut kosong dan jumlah route **30** yang
seharusnya **32**. Tidak ada perubahan kode maupun skema.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-038-2026-09-21-bagian-kontribusi-dan-status-lisensi.md` | Log sesi, termasuk alasan berkas `LICENSE` sengaja tidak dibuat | P-038 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `README.md` §1/§3/§4/§12/§13 | Bagian **12 Kontribusi** (baru, enam subbagian: persiapan, alur satu perubahan, verifikasi wajib, sepuluh aturan **dengan sumber masing-masing**, konvensi commit, kanal usulan) dan **§13 Lisensi** sebagai tabel status + peringatan "jangan anggap open source"; enam klaim basi diperbaiki, termasuk baris status Frontend, diagram §3, pohon §4 (baris `LICENSE` bertanda belum ada), baris auth yang masih "Belum ada", dan jumlah route **30 → 32** yang dihitung dari `internal/handler/router.go` — temuan **C-061** | P-038 |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-023** (baru): lisensi proyek + kebijakan kontribusi pihak ketiga, beserta butir yang masih terbuka | P-038 |
| `docs/progress/TASKS.md` | `T-049` (bagian Kontribusi) → **DONE**; **`T-050`** (lisensi + `LICENSE`) → **BLOCKED** menunggu Q-023; catatan papan disesuaikan | P-038 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`, `docs/progress/audits/README.md` | **C-061** (baru, FIXED): enam klaim basi README; marker audit 60 → **61** temuan (60 FIXED / 0 APPROVED / 1 OPEN) dan daftar FIXED pada baris ringkasan ikut diperbarui | P-038 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Ledger diselaraskan: keadaan sesudah P-038, `Q-023` pada tabel keputusan pending, dan lima baris "File Penting dari Sesi Terakhir" ditambahkan | P-038 |
| `scripts/check-doc-links.sh` | Daftar `ABSENT_DOCS` (kini hanya `CONTRIBUTING.md`): dokumen yang **sengaja tidak dibuat** dilaporkan `ABSENT`, bukan `BROKEN`. Sebelum ini setiap penyebutan berkas kontribusi terpisah — yang memang tidak ada karena aturannya ada di `README.md` §12 — membuat skrip gagal. Klasifikasi baru ini **tidak** melonggarkan pemeriksaan: dokumen hilang yang tidak terdaftar tetap `BROKEN` (dibuktikan dengan sisipan sementara) | P-038 |
| `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md` | Entri sesi ini (kewajiban protokol) | P-038 |

### Not Created (sengaja)

| File | Alasan | Prompt |
|---|---|---|
| `LICENSE` | Pemilik memilih **menunda** jenis lisensinya (Q-023); menulis teks lisensi tanpa keputusan itu berarti mengarang keputusan hukum — `CONTINUE.md` §1 butir 4 melarangnya. Pemegang hak cipta yang akan ditulis sudah diketahui: **BSA** | P-038 |

---

## 2026-09-21 (sesi P-037)

`DESIGN.md` diisi dan kerangka frontend berdiri — blocker UI yang sejak P-008 menahan Phase 4 akhirnya
dibuka. User menjawab **Q-001** (mode antislop **`during`**) dan **Q-002** (**jalur 2** ADR-0007: agen
menyusun arah desain atas izin eksplisit), sehingga **ADR-0006**/**ADR-0007** naik ke `ACCEPTED` dan
temuan **C-015** `FIXED`. Temuan **C-060** ditemukan sekaligus ditutup di sesi yang sama: `51-UX.md`
§3/§4 masih memuat palet biru-slate dan `H1 28px` — dokumen yang paling sering dibuka untuk kerja
halaman justru menyimpang dari arah desain yang baru ditetapkan. **ADR-0024** mengunci versi & tooling
frontend. Tidak ada perubahan pada `backend/` maupun skema.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `frontend/**` (51 berkas: `package.json`, `tsconfig.json`, `vite.config.ts`, `eslint.config.js`, `index.html`, `.env.example`, `.gitignore`, `src/**`) | Kerangka Vite 8 + React 19 + TS 5.9 + Tailwind v4: token, shell, primitives, lapisan API, store, halaman Login/Dashboard/ModulePending/NotFound | P-037 |
| `frontend/src/styles/tokens.css` | Token `DESIGN.md` sebagai CSS: `@theme` (Paper/Ink/Signal/status/teks/radius/bayangan/gerak) + lapisan semantik `@theme inline` + tema terang & gelap. Satu sumber nilai; komponen dilarang menulis warna langsung | P-037 |
| `frontend/src/styles/tokens.contrast.test.ts` | 31 test kontras WCAG 2.2 AA yang **membaca token dan menghitung sendiri** rasio tiap pasangan (teks, batas kontrol, focus ring, badge status, kedua tema) + larangan `#hex` di berkas komponen | P-037 |
| `frontend/src/test/{setup.ts,a11y.ts,a11y.test.tsx}` | `axe-core` atas shell/Login/Dashboard, sekaligus membuktikan pemeriksaannya menemukan pelanggaran (bukan lulus kosong) | P-037 |
| `docs/adr/0024-versi-dan-tooling-frontend.md` | ADR baru: React 19, React Router 7 (library mode), Tailwind v4 CSS-first tanpa `tailwind.config.js`, TanStack Query menyusul, Vitest + `axe-core` | P-037 |
| `docs/progress/prompts/P-037-2026-09-21-design-md-dan-kerangka-frontend.md` | Log sesi (termasuk laporan Delivery Gate antislop PASS/FAIL) | P-037 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `DESIGN.md` | **Terisi penuh** dari placeholder: identitas & kepribadian, palet + alasan & angka kontras per token, tipografi, mood/kepadatan, dials resmi (ENERGY 1 / RHYTHM 2 / MOTION 1), motif "punggung rekam", tema terang/gelap, refs & anti-refs, Design Read | P-037 |
| `docs/adr/0006-antislop-usage-mode.md`, `docs/adr/0007-design-direction-source.md` | `PROPOSED` → **`ACCEPTED`** (mode `during`; jalur 2 arah desain) | P-037 |
| `docs/adr/README.md` | Baris 0006/0007 menjadi `ACCEPTED` + tanggal keputusan; baris **0024** ditambahkan | P-037 |
| `docs/design/51-UX.md` §1/§3/§4 | Palet & tipografi yang bertentangan dengan `DESIGN.md` diganti penunjuk ke `DESIGN.md` §2/§3 + `tokens.css` (tabel token semantik + empat aturan wajib); ambang aksesibilitas dikoreksi 2.1 → **2.2** (temuan C-060) | P-037 |
| `docs/design/30-ARCHITECTURE.md` §2.1/§2.2/§2.3 | Stack frontend diselaraskan ke versi nyata (React 19, Router 7, Tailwind v4 tanpa berkas konfigurasi, TanStack Query menyusul) + tabel versi & alasan; pohon `frontend/` diganti struktur nyata beserta catatan apa yang sengaja tidak ada; teks CJK `Sesuai antislop原则` dibersihkan | P-037 |
| `docs/design/60-DEPLOYMENT.md` §2.1/§3.2 | Tabel variabel frontend (`VITE_API_BASE_URL`, `VITE_API_PROXY_TARGET`, `VITE_DEV_PORT`, `VITE_PREVIEW_PORT`) yang **dibaca dari berkas nyatanya** + skrip build yang sebenarnya + alur verifikasi frontend | P-037 |
| `docs/design/01-AGENT-WORKFRAME.md` §3.3/§5.2/§7 | Dial 1/2/1 dinyatakan **resmi** (bukan usulan); checklist frontend bertambah (Design Read, larangan warna langsung); tabel gap menandai Q-001/Q-002 selesai | P-037 |
| `docs/design/00-README.md`, `11-DESIGN-DIRECTION.md`, `12-DEVELOPMENT-WORKFLOW.md` §10 | Status `DESIGN.md` (terisi); kuesioner `11` menjadi **`SUPERSEDED`**; pitfall UI disesuaikan | P-037 |
| `AGENTS.md` | Status `DESIGN.md`/`frontend/`, aturan UI yang mengikat (token, 51-UX vs DESIGN, plafon 26px, accent sekali per layar), perintah verifikasi frontend, routing halaman, blok antislop (`during`) | P-037 |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | **C-015** → `FIXED`, **C-060** dicatat & `FIXED`; marker menjadi `total=60 fixed=59 approved=0 open=1 rejected=0 rinci=37`; paragraf ringkasan, catatan naratif, dan indeks temuan diselaraskan | P-037 |
| `docs/progress/STATE.md`, `TASKS.md`, `OPEN-QUESTIONS.md`, `TRACEABILITY.md`, `CONTINUE.md` | Ledger sesi: `T-006`/`T-007` `DONE` + `T-048` baru (papan kosong), Q-001/Q-002 `RESOLVED` + Q-021/Q-022 baru, `NFR-USABLE-03` `TODO` → `PARTIAL` dengan ambang WCAG 2.2, blok snapshot `CONTINUE.md` §0 | P-037 |
| `docs/design/70-TESTING.md` §3.14 + §7 | Bagian frontend: tiga kelompok suite (kontras token 31, `axe-core` 4, komponen/store/service), bukti gigi test kontras, dan batas jujurnya; baris frontend di tabel cakupan | P-037 |
| `scripts/check-doc-links.sh` | `is_planned` memuat `*.js`/`*.jsx`/`*.mjs`/`*.cjs`; sebelum ini setiap penyebutan berkas JS dalam backtick — termasuk `tailwind.config.js` yang **sengaja tidak ada** — dilaporkan BROKEN (13 baris) padahal tidak ada tautan dokumen yang rusak | P-037 |

---

## 2026-09-20 (sesi P-036)

Pemeriksa konsistensi ledger otomatis, dijalankan juga di CI. Seluruh cacat yang ditemukannya
sudah diperbaiki di sesi ini (rujukan test mati di `TRACEABILITY.md` dan ADR-0022, papan `TASKS.md`
yang bertentangan dengan dirinya sendiri), dan dua temuan audit baru dicatat: **C-058** dan **C-059**.
Angka audit kini dikunci marker `<!-- audit-summary ... -->` yang diperiksa mesin, bukan disalin antar dokumen.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `scripts/check-ledger.sh` | Pemeriksa konsistensi ledger: hitungan audit (+ marker & prosa), papan kerja, rujukan test/task/temuan, hitungan test `STATE.md` §3 | P-036 |
| `.github/workflows/ci.yml` | CI: job `ledger` (`check-ledger.sh` + `check-doc-links.sh`) dan job `backend` (build, vet, gofmt, `make test`) | P-036 |
| `docs/progress/prompts/P-036-2026-09-20-pemeriksa-konsistensi-ledger.md` | Log sesi | P-036 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | §6.1 baru (angka tidak ditulis dari ingatan, marker audit, satu task satu kolom, rujukan test hidup) + checklist §8 | P-036 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §8: perintah dan arti `check-ledger.sh` | P-036 |
| `docs/design/90-AGENT-GUIDE.md` | §7: perintah `check-ledger.sh` di quick reference | P-036 |
| `AGENTS.md` | Marker audit + aturan menjalankan pemeriksa | P-036 |
| `CONTINUE.md` | Marker audit, baris audit, snapshot §0 | P-036 |
| `docs/progress/STATE.md` | Marker, tanggal & blok "Diperbarui oleh", baris audit §1, dua baris §5 diberi penanda historis | P-036 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Marker + **C-058**/**C-059** (FIXED) + prosa §1 (59 temuan) | P-036 |
| `docs/progress/audits/README.md` | Marker, hitungan 59/57, rincian `C-038..C-059` | P-036 |
| `docs/progress/TASKS.md` | `T-047` DONE; `T-044` pindah ke DONE; `T-006` hanya di BLOCKED; `T-008`→`T-046` untuk kewajiban berulang | P-036 |
| `docs/progress/TRACEABILITY.md` | Tiga rujukan test mati dikoreksi (FR-AUTH-02, FR-AUTH-06, FR-VER-06) | P-036 |
| `docs/adr/0022-login-attempts-dan-auto-lock.md` | Nama dua test di §Konsekuensi dikoreksi | P-036 |

---

## 2026-09-20 (sesi P-035)

Bukti `70-TESTING.md` §3.12b dijalankan **ulang** terhadap binari terkini. Bukti lama berasal dari P-030,
sebelum `change-password` (P-034) dan `POST /auth/refresh` (ADR-0023) ada, jadi angka dan perilakunya sudah
tidak mewakili kode sekarang. Eksekusi baru membuktikan seluruh klaim `T-040`/`T-041` bertahan, **dan**
menambahkan dua baris yang belum pernah diuji di HTTP: refresh saat akun terkunci (`200`) dan refresh
sesudah `logout_all` (`401 TOKEN_REVOKED`). Tidak ada perubahan kode, dan tidak ada temuan baru.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-035-2026-09-20-probe-ulang-alur-sesi-dan-lock.md` | Log sesi, termasuk perintah lengkap probe di §6 | P-035 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `docs/design/70-TESTING.md` §3.12b | Tabel bukti diganti hasil eksekusi P-035 (termasuk dua baris interaksi refresh), judul ditandai "dijalankan ulang P-035", dan selisih angka dari P-030 dijelaskan (`false` 6 → 7 karena satu permintaan tambahan untuk membaca header `Retry-After`) | P-035 |
| `docs/progress/{TASKS,STATE,CONTINUE,SESSION-LOG}.md` | Penunjuk bukti segar pada `T-040`/`T-041`, snapshot sesi terakhir, dan entri log | P-035 |

## 2026-09-20 (sesi P-034)

Sesi P-034 punya dua bagian. **Bagian pertama:** `POST /api/v1/auth/change-password` dijalankan —
`old_password` diverifikasi lebih dulu, lalu hash baru + `users.tokens_invalid_before` + audit
`PASSWORD_CHANGED` ditulis dalam **satu** transaksi, dan token pengganti diterbitkan **sesudah** commit;
`POST /auth/refresh` sempat ditahan karena bentuk tokennya belum diputuskan (dicatat sebagai `Q-020` +
task `T-045`). **Bagian kedua:** user menjawab Q-020 dengan Opsi A, lalu `POST /auth/refresh`
dikerjakan mengikuti **ADR-0023** (JWT bertanda `typ`, 7 hari, tanpa penyimpanan di server). Satu temuan
ditutup di sesi yang sama: **C-057** (angka test per berkas di `STATE.md` §3 menyimpang dari isi repo).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-034-2026-09-20-change-password-dan-status-refresh.md` | Log sesi | P-034 |
| `docs/adr/0023-bentuk-token-refresh.md` | ADR baru `ACCEPTED`: bentuk token refresh (JWT bertanda `typ`, 7 hari, tanpa penyimpanan) beserta opsi yang ditolak dan batas yang diterima sadar | P-034, Q-020 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/service/auth_service.go` | `ChangePassword`: urutan mengikat (verifikasi → hash + pencabutan + audit satu transaksi → token pengganti sesudah commit) | P-034 |
| `backend/internal/service/audit_service.go` | Konstanta `ActionPasswordChanged` | P-034 |
| `backend/internal/handler/auth_handler.go` | Handler `ChangePassword` + `writeChangePasswordError` + `missingPasswordFields`; pemetaan `400`/`422`/`404`/`401` | P-034 |
| `backend/internal/dto/auth_dto.go` | `ChangePasswordRequest`/`ChangePasswordResponse` (response memuat token pengganti) | P-034 |
| `backend/internal/handler/router.go` | Route `POST /auth/change-password` (hanya autentikasi); catatan mengapa `POST /auth/refresh` tidak didaftarkan | P-034 |
| `backend/internal/service/auth_service_test.go`, `internal/handler/auth_handler_test.go` | Lima test baru: tiga service (`TestChangePasswordKeepsCurrentSession`, `TestChangePasswordRejectsWrongCurrentPassword`, `TestChangePasswordEnforcesNewPasswordRules`) dan dua handler (`TestChangePasswordEndToEnd`, `TestChangePasswordErrorMapping`) | P-034 |
| `docs/design/42-API.md` §2 | `change-password` menjadi kontrak yang berjalan (response + aturan tiap error); butir `keepJTI` yang usang dihapus; bagian `refresh` dipisah tegas antara "sudah diputuskan" (pemeriksaan pencabutan) dan "belum" (bentuk token, Q-020) | P-034, Q-020 |
| `docs/design/40-TSD.md` §2.4/§6 | Sketsa interface `AuthService` menandai `Refresh` belum ada; sketsa route auth menambah `change-password` + alasan `refresh` tidak ikut | P-034 |
| `docs/design/70-TESTING.md` §3.12/§3.12c | Ringkasan bukti `T-034` (tabel test + dua alasan test tidak rapuh terhadap waktu); catatan "belum dapat diuji pada `T-040`" ditutup | P-034 |
| `docs/progress/TASKS.md` | `T-034` → **DONE** (sebagian, sisanya dipindahkan eksplisit); **`T-045`** baru di papan `BLOCKED` | P-034 |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-020** baru (bentuk token `POST /auth/refresh`, tiga opsi + rekomendasi); status `Q-013` diperbarui | P-034 |
| `docs/progress/TRACEABILITY.md` | `FR-AUTH-09` → **DONE** dengan file dan nama test yang nyata | P-034 |
| `docs/progress/STATE.md` | Baris auth (§3), §4 (Q-013, Q-020, baris T-039/T-040/T-041 ditutup), §5, dan angka test per berkas dikoreksi | P-034, C-057 |
| `backend/internal/pkg/jwt/jwt.go`, `internal/pkg/jwt/jwt_test.go` | Klaim `typ` **wajib** (`access`/`refresh`), `GenerateRefresh` + `ValidateRefresh` + `RefreshExpiry` (7 hari), dan inti pemeriksaan bersama `validate`; enam test baru, termasuk refresh-ditolak-sebagai-bearer dan token-tanpa-`typ` | P-034, T-045 |
| `backend/internal/service/auth_service.go`, `internal/service/auth_service_test.go` | `Refresh` (tipe → pencabutan lewat jalur yang sama → akun aktif → pasangan token baru), `Login` kini juga menerbitkan refresh token, `Refreshed` + `ErrInvalidRefreshToken` + `ErrSessionRevoked`; lima test baru, termasuk batasan stateless yang dikunci terbuka | P-034, T-045 |
| `backend/internal/dto/auth_dto.go`, `internal/handler/auth_handler.go`, `internal/handler/router.go`, `internal/handler/auth_handler_test.go`, `internal/handler/main_test.go` | `RefreshRequest`/`RefreshResponse`, `refresh_token` + `refresh_expires_at` pada response login, handler `Refresh` dengan pemetaan `422`/`401`/`401 TOKEN_REVOKED`/`403`, route `POST /api/v1/auth/refresh` **tanpa** `AuthMiddleware`; tiga test baru | P-034, T-045 |
| `docs/design/42-API.md` §2, `44-SECURITY.md` §2.2, `40-TSD.md` §2.4/§6, `70-TESTING.md` §3.12d | Kontrak `login` (menyerahkan refresh token) dan `refresh` ditulis penuh; bentuk refresh token di dokumen keamanan; sketsa interface/route tidak lagi menandai `Refresh` sebagai belum ada; ringkasan bukti §3.12d | P-034, T-045 |
| `docs/adr/README.md`, `AGENTS.md` | Baris ADR-0023 di indeks; aturan refresh yang mengikat di `AGENTS.md` (klaim `typ` wajib, 7 hari, tanpa tabel, tanpa `AuthMiddleware`, tanpa audit) + daftar modul selesai memuat comment | P-034, T-045 |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md`, `AGENTS.md`, `CONTINUE.md`, `docs/progress/SESSION-LOG.md` | **C-057** ditambahkan & ditutup; hitungan audit menjadi **57 / 55 FIXED / 0 APPROVED / 2 OPEN**; status `C-015` dicetak tebal supaya hitungan status bertemu; aturan `change-password` + larangan mengarang refresh dicatat di `AGENTS.md`; `CONTINUE.md` naik ke P-034 | P-034 |

## 2026-09-20 (sesi P-033)

Test kontrak HTTP untuk semantik batas rentang `?due_from=&due_to=`: mengunci seluruh kombinasi
(hanya `due_from`, hanya `due_to`, kedua batas sama, rentang tertutup, rentang kosong, rentang terbalik)
sehingga perubahan semantik berikutnya tidak dapat lolos tanpa mengubah test. Tanpa perubahan kode
produksi; hanya test dan dokumentasi.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-033-2026-09-20-kontrak-batas-rentang-http.md` | Log sesi | P-033 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/handler/task_handler_test.go` | `TestTaskListDueRangeContractAtHTTP` (16 subtest + kasus rentang terbalik sampai `details.field = due_to`) | P-033 |
| `docs/design/70-TESTING.md` §3.11a | Catatan kontrak HTTP yang mengunci semantik inklusif | P-033 |
| `docs/progress/{STATE,CONTINUE,CHANGELOG,SESSION-LOG}.md` | Ledger | P-033 |

## 2026-09-20 (sesi P-032)

`T-043` — temuan **C-048** ditutup: `meta.total` kini berlaku seragam di **seluruh** endpoint daftar
(project, document, task), memakai tambalan yang sudah dipakai modul komentar. Predikat daftar diangkat
jadi variabel bersama di tiap repository sehingga jalur cepat `COUNT(*) OVER()` dan kueri hitung
membangun kueri dari sumber yang sama, dan kueri hitung hanya dijalankan pada halaman kosong di luar
halaman pertama. Tanpa migrasi; tanpa perubahan kontrak selain jaminan yang kini berlaku penuh.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-032-2026-09-20-meta-total-seragam.md` | Log sesi | P-032 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/repository/project_repository.go` | `projectListWhere` diangkat; `List` memakai jalur cepat + fallback `count` saat halaman kosong non-pertama | P-032 |
| `backend/internal/repository/document_repository.go` | `documentListWhere` diangkat; `count` memakai aturan arsip yang sama; fallback di `List` | P-032 |
| `backend/internal/repository/task_repository.go` | `taskListWhere` diangkat; `count` memakai cakupan baca yang sama; fallback di `List` | P-032 |
| `backend/internal/repository/comment_repository.go` | Komentar usang di `CommentRepository.List` yang menyebut tiga endpoint lain belum ditambal diperbarui (kini seragam, C-048 `FIXED`) | P-032 |
| `backend/internal/service/project_service_test.go` | `TestProjectListOutOfRangePageKeepsTotal` | P-032 |
| `backend/internal/service/document_service_test.go` | `TestDocumentListOutOfRangePageKeepsTotal` | P-032 |
| `backend/internal/service/task_service_test.go` | `TestTaskListOutOfRangePageKeepsTotal` | P-032 |
| `docs/design/42-API.md` §7 | Catatan `meta.total` dijelaskan berlaku di seluruh endpoint daftar (C-048 `FIXED`) | P-032 |
| `docs/design/70-TESTING.md` §3.13/§3.13a | Ringkasan bukti `T-043` (tabel test + demonstrasi test gagal saat tambalan dinonaktifkan) | P-032 |
| `docs/progress/{STATE,CONTINUE,TASKS,CHANGELOG,SESSION-LOG}.md`, `AGENTS.md`, `docs/progress/audits/*` | Ledger: `T-043` DONE, C-048 `FIXED`, hitungan audit 54 FIXED / 2 OPEN | P-032 |

## 2026-09-20 (sesi P-031)

`README.md` root repo dibuat. Sebelumnya repo tidak punya berkas README sama sekali, sehingga orang
pertama yang membuka proyek hanya melihat berkas aturan agen (`AGENTS.md`, `antislop.md`) dan direktori
dokumen. README merangkum produk, status nyata per bagian, teknologi, arsitektur, struktur repo,
prasyarat, cara menjalankan, cara test, daftar endpoint yang hidup, peran dan izin, migrasi, dan
navigasi dokumen. Penulisannya mengikuti aturan copywriting `antislop.md` (tanpa em dash, tanpa
buzzword, tanpa klaim atau angka yang tidak bersumber, dan bagian yang belum ada ditulis jujur).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `README.md` | Berkas masuk proyek: gambaran produk, status jujur, cara menyiapkan dan menjalankan, serta navigasi dokumentasi | P-031 |
| `docs/progress/prompts/P-031-2026-09-20-readme-komprehensif.md` | Log sesi | P-031 |

## 2026-09-19 (sesi P-030)

`T-040` + `T-041` — implementasi **ADR-0021** (temuan **C-033**) dan **ADR-0022** (temuan **C-009** +
**C-035**). Pencabutan seluruh sesi kini nyata lewat `users.tokens_invalid_before`, dan batas percobaan
login berujung pada **lock akun** (`423 LOCKED`) yang dapat dibuka Administrator. Penghitung di memori
`internal/service/login_guard.go` **dihapus**. Tidak ada migrasi baru: skema ADR-0021/0022 sudah dipasang
di berkas `010` pada P-029.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/repository/login_attempt_repository.go` | Menulis **setiap** percobaan login (berhasil/gagal, termasuk username tak ada) dan menghitung kegagalan menurut jam database — sumber auto-lock ADR-0022 | P-030 |
| `backend/internal/service/user_service.go`, `user_service_test.go` | `Unlock` akun: membersihkan `locked_until`, idempoten, mengaudit `USER_UNLOCKED` hanya saat ada penanda lock yang dibersihkan | P-030 |
| `backend/internal/handler/user_handler.go`, `user_handler_test.go` | Endpoint `POST /api/v1/admin/users/:id/unlock` (izin `user:update`) + test HTTP | P-030 |
| `docs/progress/prompts/P-030-2026-09-19-pencabutan-sesi-dan-auto-lock.md` | Log sesi | P-030 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/model/user.go` | Kolom kanonik `TokensInvalidBefore`, `LockedUntil` pada model user | P-030 |
| `backend/internal/repository/user_repository.go` | Membaca kolom baru; `LockUntil`/`ClearLock` untuk auto-lock | P-030 |
| `backend/internal/repository/token_revocation_repository.go` | `SessionRevoked(jti, userID, issuedAt)` menjawab dua sebab dalam satu kueri (cache per `jti`, TTL 30 detik); `RevokeAllForUser` menulis `tokens_invalid_before` + membuang cache user | P-030 |
| `backend/internal/middleware/auth.go`, `rate_limit.go` | `AuthConfig.Sessions` (interface `SessionChecker`) memeriksa `iat < tokens_invalid_before` di samping `jti`; komentar batas laju diperjelas | P-030 |
| `backend/internal/service/auth_service.go` | Login menulis `login_attempts` + `applyLockout` (`423`); `Logout` mencabut `jti` + seluruh token lama + audit `LOGOUT_ALL`; `logout_all` dijalankan | P-030 |
| `backend/internal/service/audit_service.go` | Aksi `USER_UNLOCKED` | P-030 |
| `backend/internal/service/audit_service.go`, `dto/auth_dto.go` | — (`dto.LockedDetails`: `retry_after_seconds`, `locked_until`) | P-030 |
| `backend/internal/handler/auth_handler.go`, `router.go` | Pemetaan `423 LOCKED` + header `Retry-After`; route `/admin/users/:id/unlock` | P-030 |
| `backend/internal/pkg/response/response.go` | Kode `NOT_IMPLEMENTED` dihapus (tidak ada lagi endpoint yang memakainya) | P-030 |
| `backend/internal/pkg/jwt/jwt.go` | `Validate` menolak token tanpa `iat` — syarat pencabutan massal | P-030 |
| `backend/cmd/server/main.go` | Merakit `LoginAttemptRepository`, `UserService`, kebijakan lock dari `system_settings` | P-030 |
| `backend/internal/service/auth_service_test.go`, `main_test.go`, `internal/middleware/auth_test.go`, `internal/pkg/jwt/jwt_test.go`, `internal/handler/auth_handler_test.go`, `main_test.go`, `internal/migration/migration_test.go`, `internal/handler/project_handler_test.go` | Test sesi & lock; `projectHTTPFixture.clean()` membersihkan `login_attempts` sebelum `DELETE FROM users` (C-056) | P-030 |
| `docs/design/42-API.md` §2/§11/§12 | `logout_all` → `200`; bentuk `details` `423` menjadi **objek** (C-052); `429` murni batas per alamat klien (C-054); endpoint `unlock` | P-030 |
| `docs/design/44-SECURITY.md` §2.2/§2.3 | Pencabutan per user + auto-lock; `429` vs `423` | P-030 |
| `docs/design/40-TSD.md` §5.2.2 | Perbedaan `429` (laju) vs `423` (keadaan akun) | P-030 |
| `docs/design/41-DATABASE.md` §2.1 | Makna `tokens_invalid_before` (presisi detik, C-053) dan tabel `login_attempts` | P-030 |
| `docs/design/50-FSD.md` | Perilaku lock akun + tombol Unlock di halaman pengguna | P-030 |
| `docs/design/70-TESTING.md` §3.12/§3.12b | Ringkasan bukti `T-040`/`T-041` pada server nyata | P-030 |
| `docs/adr/0021-...md`, `docs/adr/0022-...md` | Catatan "sudah dijalankan pada P-030" pada konsekuensinya | P-030 |
| `AGENTS.md`, `CONTINUE.md`, `docs/progress/{STATE,TASKS,TRACEABILITY,OPEN-QUESTIONS,CHANGELOG,SESSION-LOG}.md`, `docs/progress/audits/*` | Ledger diselaraskan: `T-040`/`T-041` DONE, C-009/C-033/C-035 `FIXED`, C-052..C-056 ditambahkan, hitungan audit dikoreksi (C-055) | P-030 |

### Removed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/service/login_guard.go`, `login_guard_test.go` | Penghitung percobaan di memori digantikan `login_attempts` + `users.locked_until` (ADR-0022 butir 6); tidak boleh ada dua pembatas yang berbeda pendapat | P-030 |
| `backend/cmd/hashtmp/main.go` | Binari sementara pembuat hash bcrypt untuk user uji; dihapus setelah bukti | P-030 |

## 2026-09-19 (sesi P-029)

`T-039` — implementasi **ADR-0019** (temuan **C-004**): `DELETE /documents/:id` diganti
`POST /documents/:id/archive`, status kanonik bertambah `archived`, dan `document_versions` menjadi
**append-only**. Arsip hanya mengubah `status` + `archived_at`: baris, versi, berkas, dan jejak auditnya
tetap; terarsip keluar dari daftar default, tetap dapat dibaca/diunduh, dan menolak unggahan versi baru
(`409`) maupun arsip ulang (`409`, supaya `archived_at` tidak bergeser).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/migration/010_session_revocation_login_attempts_document_archive.sql` | Migrasi `010` sesuai `41-DATABASE.md` §4: `documents.archived_at` + `CHECK` enam nilai + dua trigger `document_versions`; **sekaligus** skema ADR-0021/0022 (`users.tokens_invalid_before`, `users.locked_until`, `login_attempts`) karena berkas migrasi tidak boleh disunting setelah diterapkan | P-029 |
| `docs/progress/prompts/P-029-2026-09-19-arsip-dokumen-dan-trigger-versi.md` | Log sesi | P-029 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/model/document.go` | Nilai kanonik `archived`, kolom `ArchivedAt`, `IsDocumentArchived`; imutabilitas versi kini tiga lapis | P-029 |
| `backend/internal/repository/document_repository.go` | `archived_at` ikut dibaca; daftar default menyembunyikan arsip; `Archive` menggantikan `Delete` (penjaga `status <> 'archived'` di `WHERE`); `VersionKeys` dihapus (tidak dipakai lagi) | P-029 |
| `backend/internal/service/document_service.go`, `document_service_upload.go` | `Archive` + audit `DOCUMENT_ARCHIVED`; `ActionDocumentDeleted` dihapus; unggahan versi ditolak pada dokumen terarsip sebelum berkas ditulis | P-029 |
| `backend/internal/dto/document_dto.go` | `archived_at` pada response dokumen | P-029 |
| `backend/internal/handler/document_handler.go`, `router.go` | Handler `Archive`; route `DELETE /documents/:id` diganti `POST /documents/:id/archive` (izin `document:update`); pemetaan `409` untuk unggahan/arsip ulang | P-029 |
| `backend/internal/service/document_service_test.go` | Lima test arsip (`TestArchiveDocumentKeepsVersionsAndFiles`, `TestArchivedDocumentLeavesDefaultListButStaysInFilter`, `TestArchiveRejectedWhileWorkflowRunning`, `TestArchivedDocumentRejectsNewVersion`, `TestArchiveSecondTimeIsRejected`) menggantikan dua test hapus | P-029 |
| `backend/internal/handler/document_handler_test.go` | `TestArchiveDocumentEndToEnd`, izin arsip (Viewer `403` / Contributor `200`), 401 endpoint arsip; blok hapus diganti pemeriksaan berkas tetap ada | P-029 |
| `backend/internal/migration/migration_test.go`, `audit_append_only_test.go`, `main_test.go` | Daftar tabel memuat `login_attempts`; `TestDocumentStatusVocabularyIncludesArchived`, `TestLoginAttemptsSchemaExists`; tujuh test `TestDocumentVersions_*` + helper pasangan trigger + helper `seedDocumentVersion` | P-029 |
| `docs/design/42-API.md` §4 | Tabel izin (arsip = `document:update`), enam nilai `status`, kolom `archived_at`, aturan daftar default, catatan "kontrak ini berjalan" menggantikan blok "Status implementasi" | P-029 |
| `docs/design/44-SECURITY.md` §3.1.3/§6 | Pengecualian trigger dihapus; `document_versions` append-only sejak migrasi `010` | P-029 |
| `docs/design/40-TSD.md` §2.4 | `Delete` → `Archive`; aksi audit `DOCUMENT_ARCHIVED` | P-029 |
| `docs/design/50-FSD.md` §11.1 | Catatan bahwa nilai `archived` berlaku di kode sejak `T-039` | P-029 |
| `docs/design/70-TESTING.md` §3.12/§3.12a/§4.3 | Status `T-039` selesai + nama test nyata, ringkasan bukti server nyata (§3.12a), dan catatan helper append-only `document_versions` | P-029 |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | C-004 `APPROVED` → `FIXED`; **C-051** baru (FIXED: FK kaskade `document_versions` tetap ada tetapi tertahan trigger); hitungan **51 / 45 FIXED / 3 APPROVED / 3 OPEN** | P-029 |
| `docs/design/41-DATABASE.md` §2.3 | Catatan C-051: FK kaskade `document_versions` tetap di skema, tetapi kaskadenya ditahan trigger append-only — pembersihan dokumen berversi wajib lewat `SET LOCAL bwdcs.audit_maintenance = 'on'` | P-029 |
| `docs/progress/TASKS.md`, `STATE.md`, `CONTINUE.md`, `AGENTS.md`, `TRACEABILITY.md` | `T-039` DONE; posisi, aturan modul dokumen, "migrasi `010` sudah terpasang", FR-DOC-03 & FR-VER-03 `DONE`, FR-AUDIT-01 menyebut `DOCUMENT_ARCHIVED` | P-029 |

### Removed

| File | Alasan | Prompt |
|---|---|---|
| route `DELETE /api/v1/documents/:id` | Diganti arsip (ADR-0019 butir 1); penghapusan permanen tidak ada di MVP | P-029 |
| konstanta `service.ActionDocumentDeleted` + metode `DocumentRepository.Delete`/`VersionKeys` | Tidak ada lagi operasi hapus dokumen | P-029 |

---

## 2026-09-19 (sesi P-028)

**Q-017 butir (10) dikoreksi user**: penyaring rentang `GET /tasks?due_from=&due_to=` kini memakai interval
**tertutup** `[due_from, due_to]` — **kedua batas inklusif** — menggantikan setengah terbuka `[from, to)`
yang diusulkan agen pada P-026. Akibatnya `due_to == due_from` **sah** (satu instan) dan hanya rentang
terbalik yang dijawab `422`; konsekuensinya (rentang bersebelahan dapat tumpang tindih) dinyatakan di
`42-API.md` §6, bukan disembunyikan. Perubahan menyentuh tiga baris kode di tiga lapisan (kueri `<=`,
validasi, komentar) tanpa migrasi.

**Q-017 butir (11) diverifikasi ulang, tanpa perubahan kode**: atribusi field pada `422` untuk UUID/tanggal
bukan-UUID di body (temuan C-045 yang ditutup P-026) dijalankan lagi pada binari yang baru dibangun —
`document_id`, `title`, `due_date`, `body`, dan `owner_id` (`POST /projects`) semuanya disebut dengan
alasan yang benar. Yang diminta prompt ini memang sudah ada di repo; mengerjakannya ulang berarti
menulis ulang kode yang sudah berjalan.

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/repository/task_repository.go` | Predikat batas atas rentang `due_date` menjadi `<=`, dengan komentar yang menyebut keputusan user P-028 | P-028 |
| `backend/internal/service/task_service.go`, `backend/internal/handler/task_handler.go` | Validasi rentang: `due_to < due_from` → `422`; `due_to == due_from` sah. Komentar semantik diperbarui di kedua lapisan | P-028 |
| `backend/internal/service/task_service_test.go`, `backend/internal/handler/task_handler_test.go` | `TestTaskListDueRangeFilterIsHalfOpen` → `TestTaskListDueRangeFilterIsInclusiveBothEnds` (7 kasus); kasus pembeda pada `due_to` yang tepat sama dengan `due_date` task; `due_to == due_from` pindah ke daftar sah | P-028 |
| `docs/design/42-API.md` §6, `docs/design/50-FSD.md` §6.1 | Semantik rentang dinyatakan tertutup, lengkap dengan konsekuensi dan catatan siapa yang memutuskan | P-028 |
| `docs/design/70-TESTING.md` §3.11/§3.11a | Tabel sebelum/sesudah + tanda bahwa bukti P-026 memakai semantik lama | P-028 |
| `docs/progress/OPEN-QUESTIONS.md` Q-017 butir (10) & Status | Opsi (a) ditandai "dipilih agen, dikoreksi user P-028"; butir (11) ditandai masih berlaku | P-028 |
| `docs/progress/TASKS.md` | `T-044` baru (DONE); baris `T-038` diberi pointer koreksi semantik | P-028 |
| `docs/progress/STATE.md`, `CONTINUE.md` | Baris "Penyaring rentang tanggal" dan baris Q-017 diselaraskan | P-028 |
| `docs/progress/prompts/P-028-2026-09-19-rentang-tanggal-inklusif-q017.md` | Added — log sesi ini | P-028 |

---

## 2026-09-19 (sesi P-027)

**Modul Comment** (`T-042`): lima endpoint `42-API.md` §7 hidup — `GET|POST /api/v1/comments`,
`GET|PATCH|DELETE /api/v1/comments/:id`. Dua aturan `44-SECURITY.md` §3.1.3 dijalankan sekaligus dan
keduanya **di dalam kueri**: cakupan baca diturunkan dari entitas yang dikomentari (tabel `comments`
tidak menyimpan `project_id`, jadi pemetaannya tinggal satu di `commentEntityProjectCase`), sedangkan
edit/hapus dibatasi **kepemilikan** (`WHERE created_by_id = actor`) — bukan izin, karena matriks ADR-0014
tidak memuat `comment:update`/`comment:delete` dan kelima route cukup dijaga `comment:read`/
`comment:create`. Phase 3 (Task & Comment) tertutup.

**Tiga temuan baru, semuanya lahir dari menjalankan sesuatu** (pola yang sama dengan C-039..C-050):

- **C-048** — `meta.total` melaporkan `0` pada halaman di luar rentang karena `COUNT(*) OVER()` tidak
dievaluasi tanpa baris. Ketahuan dari test paginasi sendiri. Modul komentar menambalnya (kueri hitung
terpisah hanya saat halaman kosong dan bukan halaman pertama; server nyata: `?page=5` → `data: []`,
`total: 2`); project/document/task menyusul di **`T-043`** — `OPEN`.
- **C-049** — `GET /comments/:entityType/:entityId` (draf §7) **tidak dapat** dipasang bersama
  `GET /comments/:id`: Gin panik saat registrasi rute. Kontrak §7 memakai bentuk kueri → `FIXED`.
- **C-050** — `50-FSD.md` §7 menjanjikan balasan ber-thread sedangkan tabel `comments` tidak punya kolom
  induk. FSD dinyatakan "belum didukung" dan perilakunya dikunci test → `OPEN`, **Q-019** (rekomendasi:
  tunda).

Hitungan audit diambil ulang dari tabel: **50 temuan / 43 FIXED / 4 APPROVED / 3 OPEN** (43 + 4 + 3 = 50).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/model/comment.go`, `internal/model/comment_test.go` | Model komentar + kosakata tertutup `entity_type` (sama dengan `CHECK` kolomnya) dan batas isi 2000 karakter | P-027 |
| `backend/internal/repository/comment_repository.go` | Pemetaan entitas → project (satu tempat), predikat cakupan di `WHERE`, jalur kepemilikan tanpa JOIN project, paginasi + tambalan total C-048 | P-027 |
| `backend/internal/service/comment_service.go`, `internal/service/comment_service_test.go` | Cakupan (`systemScope`), kepemilikan, validasi isi satu tempat, audit `COMMENT_*` di transaksi yang sama | P-027 |
| `backend/internal/dto/comment_dto.go`, `internal/handler/comment_handler.go`, `internal/handler/comment_handler_test.go` | Bentuk response + lima endpoint lewat HTTP, termasuk `parseCommentListQuery` yang mewajibkan `entity_type`/`entity_id` | P-027 |
| `docs/progress/prompts/P-027-2026-09-19-modul-comment-dan-kepemilikan.md` | Log sesi ini | P-027 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/handler/router.go`, `backend/cmd/server/main.go`, `backend/internal/handler/main_test.go` | Pemasangan lima route komentar (`comment:read`/`comment:create`) + wiring service/handler | P-027 |
| `backend/internal/handler/project_handler_test.go`, `backend/internal/service/project_service_test.go` | Pembersihan fixture menghapus komentar lebih dulu — `comments.created_by_id` `ON DELETE RESTRICT` | P-027 |
| `docs/design/42-API.md` §7 | Kontrak penuh lima endpoint: pemetaan `entity_type` → project, aturan input, bentuk kueri daftar (beserta alasan C-049), kepemilikan, kode error, catatan C-048 | P-027 |
| `docs/design/40-TSD.md` §3/§5.3/§6 | `CommentService`, `CommentRepository`, kolom turunan `created_by_username`, blok route komentar | P-027 |
| `docs/design/44-SECURITY.md` §3.1.3 | Rujukan implementasi keempat: pemetaan entitas → project, kepemilikan, alasan `PATCH`/`DELETE` memakai `comment:read` | P-027 |
| `docs/design/50-FSD.md` §7 | `Reply (threaded)` dinyatakan belum didukung; aturan cakupan/kepemilikan/urutan ditulis eksplisit; `@mention` dinyatakan milik modul Notification | P-027 |
| `docs/design/70-TESTING.md` §3.13 | Inventaris test modul Comment + bukti server nyata + dua temuan yang hanya terlihat saat dijalankan | P-027 |
| `docs/design/80-ROADMAP.md` §3 | Phase 3 → selesai (Exit Criteria dicentang), dengan catatan threading | P-027 |
| `docs/progress/TRACEABILITY.md` | Baris `FR-CMT-01`..`FR-CMT-03` (`DONE` + bukti); `FR-AUDIT-01` diperluas untuk tiga aksi komentar | P-027 |
| `docs/progress/audits/AUDIT-001-...md`, `docs/progress/audits/README.md` | C-048/C-049/C-050 ditambahkan; hitungan menjadi **50 / 43 FIXED / 4 APPROVED / 3 OPEN** | P-027 |
| `docs/progress/TASKS.md` | `T-042` → DONE dengan bukti; `T-043` baru (C-048); `T-024` 45/55 (total endpoint 55, dihitung ulang); `T-017` diperjelas | P-027 |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-019** baru (threading); Q-018 diberi hasil P-027 | P-027 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Ledger diselaraskan: hitungan audit, tabel modul Comment, aturan modul komentar yang mengikat, next action | P-027 |

---

## 2026-09-19 (sesi P-026)

Sesi ini punya **dua bagian**, keduanya di bawah satu prompt yang sama.

**Bagian 1 — perbaikan gap hasil pemeriksaan ulang.** **C-045** ditutup (`bindJSON` kini menamai field
untuk UUID/waktu/tipe/JSON rusak — berlaku lintas modul), **C-046** ditutup (penyaring rentang
`?due_from=`/`?due_to=` sebagai interval **setengah terbuka** ber-batas RFC 3339), dan **C-047**
ditemukan sekaligus ditutup (tabel status fase `80-ROADMAP.md` tertinggal; label fase modul Task
diselaraskan ke Phase 3).

**Bagian 2 — sembilan temuan terbuka dijawab dengan best practice**, setelah user meminta pertanyaan
terbuka **dijawab** (bukan diserahkan kembali tanpa data). Hasilnya: **C-006** ("Reviewer" = peran
fungsional, bukan role kelima), **C-007** (dua hierarki role dipisahkan dan tidak pernah digabung),
**C-010** (`responsible_user_id` ditunda; MVP role-based), dan **C-028** (retensi audit = operasi
pemeliharaan berlantai 12 bulan, **ADR-0020**) → `FIXED`; serta **C-004** (**ADR-0019** arsip sebagai
default), **C-033** (**ADR-0021** `users.tokens_invalid_before`), **C-009** + **C-035** (**ADR-0022**
`login_attempts` + auto-lock) → `APPROVED` dengan tugas implementasi `T-039`/`T-040`/`T-041`. Satu-satunya
yang masih `OPEN` adalah **C-015** (arah desain — milik user). Hitungan audit dikembalikan ke tabel dan
dapat diperiksa silang: **47 temuan / 42 FIXED / 4 APPROVED / 1 OPEN**.

> **Koreksi angka sesi sebelumnya (append-only).** Baris `P-025` di bawah mencatat "46 temuan / 35 FIXED /
> 11 OPEN". Angka itu benar **pada saat itu** (C-047 belum ditemukan dan C-045/C-046 belum diperbaiki);
> angka yang berlaku sekarang adalah **47 / 42 / 4 / 1** di bagian atas ini dan di `AUDIT-001`.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0019-arsip-sebagai-default-penghapusan-dokumen.md` | Keputusan C-004: `DELETE /documents/:id` diganti arsip, versi tetap immutable, penghapusan permanen ditunda, dan karena kaskade hilang `document_versions` dapat diberi trigger append-only | P-026 |
| `docs/adr/0020-retensi-audit-log-sebagai-operasi-pemeliharaan.md` | Keputusan C-028: retensi = operasi pemeliharaan berlantai 12 bulan, **bukan** setting UI; jalur `bwdcs.audit_maintenance`, prosedur di runbook | P-026 |
| `docs/adr/0021-pencabutan-seluruh-sesi-tokens-invalid-before.md` | Keputusan C-033: satu kolom `users.tokens_invalid_before` + pemeriksaan `iat` di middleware; `token_revocations` tetap untuk satu token | P-026 |
| `docs/adr/0022-login-attempts-dan-auto-lock.md` | Keputusan C-009 + C-035: tabel `login_attempts` tanpa FK pada username, `users.locked_until`, `423 LOCKED`, `unlock` oleh admin, retensi 90 hari | P-026 |
| `docs/progress/prompts/P-026-2026-09-19-perbaikan-c045-c046-dan-riset-best-practice.md` | Log sesi — berkas ini sudah dirujuk banyak ledger sebelum ada (gap yang diperbaiki di sesi yang sama) | P-026 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/handler/project_handler.go` | `bindJSON` membedakan tiga sebab kegagalan decode dan menamai field; helper `undecodableField`/`messageForUndecodable` (C-045) | P-026 bagian 1 |
| `backend/internal/handler/project_handler_test.go`, `task_handler_test.go`, `internal/service/task_service_test.go` | Test C-045 (4 sebab, lintas modul) dan C-046 (6 kasus service + 7 kasus HTTP) | P-026 bagian 1 |
| `backend/internal/repository/task_repository.go`, `internal/service/task_service.go`, `internal/handler/task_handler.go` | Penyaring `due_from`/`due_to` setengah terbuka + `parseRFC3339Query` yang menerima `+07:00` maupun `%2B07:00` (C-046) | P-026 bagian 1 |
| `docs/design/42-API.md` §4/§6/§12 | Kontrak arsip dokumen menggantikan `DELETE`; penyaring rentang; tabel pemetaan error body; `TOKEN_REVOKED` menyebut kedua sebab | P-026 |
| `docs/design/42-API.md` §2/§11 | `logout_all` + pencabutan sesi (ADR-0021), `423 LOCKED` (ADR-0022), `POST /admin/users/:id/unlock`, `PATCH /admin/users/:id` mematikan sesi saat akun dinonaktifkan | P-026 bagian 2 |
| `docs/design/44-SECURITY.md` §2.3/§3.1/§3.3/§6.1/§6.2/§8 | Auto-lock + `login_attempts` (C-009/C-035), dua hierarki role terpisah (C-007), izin arsip vs `document:delete` (C-004), trigger `document_versions`, retensi (C-028), empat butir checklist baru | P-026 bagian 2 |
| `docs/design/41-DATABASE.md` §2.1/§2.3/§3/§4 | `users.tokens_invalid_before`, `users.locked_until`, tabel `login_attempts` + indeks, `documents.archived_at` + nilai kanonik `archived`, dan migrasi **`010`** beserta alasan mengapa perubahan tidak disunting ke berkas `001`-`009` yang sudah terpasang | P-026 bagian 2 |
| `docs/design/50-FSD.md` §4.3/§10.5/§11.1 | Arsip menggantikan hapus; halaman Settings hanya memuat kunci yang benar-benar ada (masa berlaku token & password policy dinyatakan bukan setting); label `Archived` | P-026 bagian 2 |
| `docs/design/20-SRS.md` (FR-AUTH-06/FR-AUTH-08/FR-AUTH-09/FR-DOC-03/FR-VER-03/FR-WF-03/FR-AUDIT-01) + §2.3 | Requirement diselaraskan dengan empat ADR; "Reviewer" dinyatakan peran fungsional; FR-AUDIT-01 "login" = login **berhasil** | P-026 bagian 2 |
| `docs/design/40-TSD.md` §5.2.1/§5.2.2 | Middleware memeriksa **dua** sebab pencabutan; `LoginGuard` di memori digantikan `login_attempts` + `locked_until` | P-026 bagian 2 |
| `docs/design/10-BRD.md` §4/BR-07, `51-UX.md` §2.1, `43-WORKFLOW.md` §5 | "Reviewer" = peran fungsional (C-006); "Reviewer+" = role berwenang bertindak pada step itu; `responsible_user_id` ditunda (C-010) | P-026 bagian 2 |
| `docs/design/60-DEPLOYMENT.md` §6.4 (baru) | Prosedur retensi audit & `login_attempts`: langkah bernomor, perintah, larangan melepas trigger / menjalankannya tanpa cadangan | P-026 bagian 2 |
| `docs/design/70-TESTING.md` §3.10/§3.11/§3.12 | Inventaris test task, aturan "bangun ulang binari sebelum membuktikan perubahan kode", dan **test wajib** untuk tiga ADR yang belum diimplementasikan | P-026 |
| `docs/design/80-ROADMAP.md` §3 | Tabel status fase diperbaiki + catatan koreksi bahwa dokumen itu sumber urutan fase (C-047) | P-026 bagian 1 |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | C-006/C-007/C-010/C-028 → `FIXED`; C-004/C-009/C-033/C-035 → `APPROVED` (keputusan ada, kode `T-039`/`T-040`/`T-041`); footer dan catatan kaki dikoreksi; hitungan **47 / 42 / 4 / 1** | P-026 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-012/Q-013/Q-014 `RESOLVED` dengan ADR-nya; Q-010 ditutup; §3 diberi tabel "keputusan akhir vs rekomendasi" termasuk dua tempat yang sengaja **menyimpang** (C-009: ambang tidak dinaikkan; C-028: retensi bukan kunci `system_settings`) | P-026 |
| `docs/progress/TASKS.md` | `T-039`/`T-040`/`T-041` baru (masing-masing dengan bukti selesai yang harus dipenuhi); `T-017` menunggu ketiganya; `T-034` tidak lagi terblokir keputusan | P-026 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Ledger diselaraskan: hitungan audit, fase roadmap, tabel modul Task (sebelumnya masih "Belum"), aturan auth/`logout_all`, aturan versi `goose`, dan next action | P-026 |
| `docs/progress/TRACEABILITY.md` | Baris requirement yang tersentuh empat ADR diberi penunjuk ADR + status `APPROVED` yang harus dibaca apa adanya | P-026 |

---

## 2026-09-19 (sesi P-025)

Mengerjakan **`T-038`**: modul **Task** — lima endpoint `42-API.md` §6 dengan **cakupan baris kedua**
`44-SECURITY.md` §3.1.3 (baca mengikuti keanggotaan project untuk Contributor/Viewer, seluruh organisasi
untuk Manager/Administrator; tulis hanya task milik Contributor), penanda overdue tetap turunan
(ADR-0012) tetapi kini juga **penyaring `?overdue=` tri-state** di `WHERE`, dan `?priority=` mengikuti
`50-FSD.md` §6.1. Kontrak §6 ditulis ulang dari tiga baris menjadi kontrak penuh; TRACEABILITY
FR-TASK-01..07 diisi. Dua temuan baru dicatat (**C-045**, **C-046**) bersama sebelas keputusan modul di
**Q-017**. Bukti: `make test` delapan paket `ok` (3 + 12 + 10 test task, 0 FAIL/SKIP) dan dua rangkaian
probe HTTP pada server nyata (201/403/401/404/409/422 + tujuh entri audit task).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/model/task.go` | Kosakata tertutup status/prioritas, normalisasi nilai klien, `CanTransitionTaskStatus` (tiga aksi `50-FSD.md` §6.3), dan `IsTaskOverdue` sebagai **satu tempat** rumus FR-TASK-06 | P-025 |
| `backend/internal/model/task_test.go` | Tiga test: himpunan tertutup, transisi status, overdue turunan (`due_date` kosong/`completed` tidak pernah overdue) | P-025 |
| `backend/internal/repository/task_repository.go` | `TaskScope` + `taskReadPredicate`/`taskWritePredicate` (**parameterized**, perbedaan role dikirim sebagai parameter), daftar ber-penyaring, `FindByIDForUpdate` untuk cakupan tulis, `DocumentInProject` (FR-TASK-05) | P-025 |
| `backend/internal/service/task_service.go` | Aturan domain task + audit dalam transaksi pemanggil (ADR-0011); penolakan pindah project dan transisi status terlarang | P-025 |
| `backend/internal/service/task_service_test.go` | Dua belas test: cakupan baca per role (FR-TASK-07), cakupan tulis contributor, field+audit, transisi, penjaga `PATCH`, penyaring daftar, dan `TestTaskListOverdueFilterMatchesDerivedFlag` yang mengikat penyaring SQL dengan penanda kode | P-025 |
| `backend/internal/dto/task_dto.go` | Bentuk request/response task + kolom turunan (`project_code`, `assignee_username`, `document_number`, `is_overdue`, `project_archived`) | P-025 |
| `backend/internal/handler/task_handler.go` | Lima handler, validasi `422` ber-`details.field`, pemetaan error domain ke `404`/`403`/`409`/`422`, dan pemeriksaan `task:assign` **hanya bila** body memuat `assignee_id` | P-025 |
| `backend/internal/handler/task_handler_test.go` | Sepuluh test HTTP dengan token hasil login nyata | P-025 |
| `docs/progress/prompts/P-025-2026-09-19-modul-task-dan-cakupan-baris-kedua.md` | Log sesi: interpretasi, aksi, bukti, dan sisa pekerjaan | P-025 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/service/scope.go` | Menambah `taskScope`: modul pertama dengan cakupan **baca ≠ tulis** (`systemScope` tidak cukup), tetap satu tempat agar aturan role tidak disalin per modul | P-025 |
| `backend/internal/handler/router.go`, `backend/cmd/server/main.go` | Group `/tasks` dengan `RequirePermission` dari matriks `44-SECURITY.md` §3.1.2 + wiring service/handler | P-025 |
| `backend/internal/handler/main_test.go` | Fixture test ikut membersihkan `tasks` dan `project_members` supaya suite tetap terisolasi (C-036/C-038) | P-025 |
| `backend/internal/handler/project_handler.go`, `backend/internal/handler/document_handler.go` | Helper `actorFrom` dipakai bersama oleh tiga handler (tanpa perubahan perilaku) | P-025 |
| `docs/design/42-API.md` §6 | Dari tiga baris menjadi kontrak penuh: izin per endpoint, aturan tiap field, tabel transisi status → endpoint+izin, penyaring (`?priority=`, `?overdue=` tri-state), kode error, dan contoh response dengan kolom turunan | P-025 |
| `docs/design/40-TSD.md` §2.3/§2.4/§2.5/§6 | Catatan model task (kosakata + turunan + transisi), `TaskService`, `TaskScope`/`TaskRepository`, route `tasks`, dan aturan 3 (izin bergantung isi body) kini menyebut **dua** route: aksi workflow dan `PATCH /tasks/:id` | P-025 |
| `docs/design/44-SECURITY.md` §3.1.3 | Rujukan implementasi **ketiga**: dua predikat task dan catatan bahwa penyaring daftar tidak menambah izin | P-025 |
| `docs/design/50-FSD.md` §6.1/§6.3 | Penyaring dan sub-halaman dipetakan ke parameter API yang benar-benar ada; rentang tanggal dinyatakan **belum** ada (Q-017); tiga aksi dipetakan ke endpoint/izin yang berbeda | P-025 |
| `docs/design/70-TESTING.md` §3.10 (baru) + §4.1 | Inventaris test modul task, bukti server nyata, dan aturan hasil temuan: **bangun ulang binari sebelum membuktikan perubahan kode** | P-025 |
| `docs/progress/TRACEABILITY.md` | Tujuh baris `FR-TASK-01..07` diisi (dua di antaranya sebelumnya `TODO`), `FR-AUDIT-01` diperbarui untuk empat aksi task, catatan sesi ditambahkan | P-025 |
| `docs/progress/audits/AUDIT-001-...md`, `docs/progress/audits/README.md` | **C-045** (pesan `422` menyesatkan untuk UUID tidak sah di body) dan **C-046** (penyaring "Due date range" tanpa kontrak) ditambahkan sebagai `OPEN`; ringkasan menjadi **46 temuan / 35 FIXED / 11 OPEN** | P-025 |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-017**: sembilan kontrak modul task yang diputuskan agen + dua keputusan yang menunggu user (rentang tanggal, atribusi error body) dengan rekomendasi | P-025 |
| `docs/progress/TASKS.md` | `T-038` → DONE dengan bukti; `T-017` 9 → **11** temuan OPEN; `T-024` 35 → **40/51** endpoint beranotasi | P-025 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Ledger diselaraskan: 20 endpoint hidup, aturan modul task yang mengikat, hitungan audit, next action (Comment), dan catatan operasional cara membuktikan di server dev | P-025 |
| `.freebuff/run.md` | Cara menjalankan server untuk bukti (binari, port 8081, cara melepas) | P-025 |

---

## 2026-09-19 (sesi P-024)

Mengerjakan **`T-036`** dengan izin user: menyiapkan **database test terpisah `bwdcs_test`** dan mengarahkan
`TEST_DATABASE_URL` ke sana, sehingga suite tidak lagi bergantung pada keadaan database dev yang kosong —
menutup temuan **C-038**. Sekalian menutup dua temuan baru yang muncul saat kerja: **C-043**
(`60-DEPLOYMENT.md` §3.1 menyalin `Makefile` yang sudah menyimpang) dan **C-044** (hitungan audit menulis
31 FIXED padahal tabelnya memuat 32). Bukti utama: dengan project nyata hidup di database dev, `make test`
hijau dua kali berturut-turut sementara perintah lama yang menunjuk database dev gagal `SQLSTATE 23503`.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/Makefile` target `test-dsn` | Melihat database mana yang akan dipakai test **tanpa** mencetak sandi (penting agar verifikasi tidak membocorkan kredensial ke log) | P-024 |
| Database `bwdcs_test` (bukan berkas) | Database test terpisah (owner role `bwdcs`, dibuat lewat peran superuser lokal karena role aplikasi tidak diberi `CREATEDB`); skema dimigrasikan otomatis `TestMain` (ADR-0018) | P-024 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/Makefile` target `test` | Dulu `go test ./... -cover -p 1` telanjang: tanpa `TEST_DATABASE_URL` test integrasi `t.Skip` sementara paket tetap `ok` (hijau palsu), dan bila variabelnya diisi ke database dev, suite merusak/menabrak data dev (C-038). Kini memuat `.env`, **menurunkan** DSN ke `bwdcs_test` (`TEST_DB_NAME`), memakai `-count=1`, mencetak nama database yang dipakai (sandi disamarkan), dan **berhenti dengan pesan** bila DSN menunjuk database dev | P-024 |
| `backend/internal/bootstrap/bootstrap_test.go` | Pesan kegagalan `DELETE FROM users` dulu hanya menyebut akibatnya ("database test harus bersih dari modul lain"); kini menyebut penyebab dan jalan keluarnya (`TEST_DATABASE_URL` harus menunjuk database test terpisah) | P-024 |
| `docs/design/70-TESTING.md` §8/§8.1 | Database test terpisah berubah dari rekomendasi menjadi **praktik wajib** + prosedur pembuatan `bwdcs_test`; ditambah peringatan bahwa `go test` tanpa `TEST_DATABASE_URL` bukan bukti apa pun; catatan C-038 ditutup | P-024 |
| `docs/design/60-DEPLOYMENT.md` §3.1 | Cuplikan `Makefile` yang sudah menyimpang (tanpa `-p 1`, tanpa pemuatan `.env`, DSN migrasi tanpa variabel) **dihapus** dan diganti tabel target + penunjuk ke `backend/Makefile` sebagai sumber tunggal (temuan **C-043**, kelas C-014) | P-024 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` §8, `docs/design/90-AGENT-GUIDE.md` §7 | Perintah verifikasi standar memakai `make test` (yang menyiapkan database test), bukan `go test ./... -count=1` telanjang | P-024 |
| `docs/progress/audits/AUDIT-001-...md` + `audits/README.md` | **C-038** OPEN → FIXED; **C-043** dan **C-044** ditambahkan (FIXED di sesi yang sama); hitungan diambil ulang dari tabel: **44 temuan / 35 FIXED / 9 OPEN** (35 + 9 = 44) | P-024 |
| `docs/progress/TASKS.md` | `T-036` pindah ke DONE dengan bukti; `T-017` 10 → **9** temuan OPEN dan tanpa lagi menyebut C-038 sebagai butuh izin | P-024 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Ledger diselaraskan: hitungan audit, perintah test kanonik (`make test`), baris environment database test, dan catatan bahwa `go test` telanjang bukan bukti | P-024 |

---

## 2026-09-19 (sesi P-023)

Mengerjakan **`T-037`**: modul **Document** — metadata dokumen (`POST /documents`), unggah versi
berkas (`POST /documents/:id/upload`), riwayat versi, unduhan ber-audit, dan hapus berkaskade, dengan
generator `document_number` `{PROJECT_CODE}-{NNN}` **di dalam transaksi yang sama** (ADR-0017) dan
cakupan data yang **sama** dengan project (`44-SECURITY.md` §3.1.3). Bukti utamanya dijalankan pada
server nyata: nomor `DOC-UJI-001`/`DOC-UJI-002`, versi `1.0` → `1.1`, unduhan yang isinya identik
dengan berkas asli, dan cakupan yang menyembunyikan dokumen dari non-anggota (`404`, bukan `403`).
Empat temuan baru (**C-039**–**C-042**) muncul dari **menjalankan** test dan dari memeriksa sumber
sebelum menerapkan aturan; semuanya ditutup di sesi yang sama. Delapan kontrak yang diputuskan agen
dicatat di `OPEN-QUESTIONS.md` **Q-016**.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/model/document.go` | `Document`/`DocumentVersion`/`DocumentCategory`, status kanonik (FR-DOC-03), dan aturan versi `ParseVersion`/`FormatVersion`/`NextVersion` (FR-VER-02) | P-023 |
| `backend/internal/repository/document_repository.go` | Dokumen bercakupan: `List`/`FindByID` memakai `projectScopePredicate` di `WHERE`, `NextNumber` (`INSERT … ON CONFLICT … RETURNING`), `CategoryExists`, `AddVersion`/`SetCurrentVersion`/`Versions`/`LatestVersion`/`FindVersion`, `VersionKeys`, `Delete` berkaskade, `HasRunningWorkflow` | P-023 |
| `backend/internal/service/document_service.go` | `Create` (nomor + INSERT + audit dalam satu transaksi), `List`, `Get`, `Delete` (penjaga workflow + kaskade berkas), `Scope` | P-023 |
| `backend/internal/service/document_service_upload.go` | `UploadVersion` (checksum SHA-256, versi berikutnya, pembersihan berkas bila transaksi gagal), `Versions`, `Download` (audit transaksi tersendiri), validasi tipe/ukuran, `limitedReader` | P-023 |
| `backend/internal/service/scope.go` | `Actor` + `systemScope` diekstrak dari `ProjectService` supaya project dan document memakai **satu** aturan cakupan | P-023 |
| `backend/internal/dto/document_dto.go` | Bentuk request/response `42-API.md` §4 (termasuk `current_version` null-able) | P-023 |
| `backend/internal/handler/document_handler.go` | Tujuh handler: daftar berfilter, detail, buat, unggah multipart (magic bytes + `Seek(0,0)`), versi, unduh streaming ber-`Content-Disposition`, hapus; pemetaan error terpusat | P-023 |
| `backend/internal/service/document_service_test.go` | 9 test integrasi: metadata + audit, kategori asing, checksum/`file_key`, versi minor → major, penolakan tipe/ukuran, unduh + audit, hapus berkaskade, penjaga workflow, cakupan | P-023 |
| `backend/internal/service/document_number_test.go` | 5 test penomoran ADR-0017: tiga test `70-TESTING.md` §3.5 (berurutan, rollback tidak menghabiskan nomor, konkurensi) + nomor per project + project di luar cakupan tidak membangkitkan nomor (test keempat §3.5 — tolak nomor dari klien — ada di lapisan HTTP karena `422` hanya ada di sana) | P-023 |
| `backend/internal/service/document_upload_limit_internal_test.go` | Unit `limitedReader`: berhenti tepat di batas dan menolak byte kelebihan | P-023 |
| `backend/internal/handler/document_handler_test.go` | 11 test HTTP: 401 ketujuh endpoint, matriks izin, 201 + nomor, nomor dari klien → `422`, unggah `1.0`/`1.1`, unduh + `Content-Disposition` + isi identik (termasuk berkas besar), 422 field `file`, cakupan 404, hapus 200 → 404, `entity_id` audit = nomor dokumen | P-023 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/service/project_service.go` | `normalizeProjectCode` dijalankan juga di service: kode huruf kecil dapat membuat nomor dokumen `webdocs-001` (temuan **C-041**) | P-023 |
| `backend/internal/model/project.go` | Komentar `IsArchived()` menjanjikan aturan "project arsip tidak menerima dokumen baru" yang tidak ada di dokumen desain (temuan **C-042**) | P-023 |
| `backend/internal/pkg/jwt/jwt_test.go` | `tamperSignature` mengubah byte tanda tangan, bukan karakter terakhir base64url — test tidak lagi gagal acak (temuan **C-039**) | P-023 |
| `backend/internal/handler/router.go` | Tujuh route `/documents` dengan `RequirePermission` dari matriks ADR-0014; daftar versi memakai `document:read` (matriks tidak memuat `document_version:read`) | P-023 |
| `backend/cmd/server/main.go` | Wiring `DocumentService` beserta `FileStorage` yang sama dengan health check | P-023 |
| `backend/internal/handler/main_test.go` | Harness dipecah: `newEngineParts` mengembalikan engine + storage + service, supaya test dokumen mengunggah berkas sungguhan | P-023 |
| `docs/design/42-API.md` §4 | Ditulis ulang: tabel izin 7 endpoint, aturan query, bentuk `current_version`, tabel versi berikutnya, validasi berkas, semantik unduhan, syarat hapus; contoh `checksum` diselaraskan ke 64 heksadesimal (C-040) | P-023 |
| `docs/design/40-TSD.md` §2.3-§2.6/§6 | Model/document service/repository diselaraskan dengan implementasi; pola handler multipart + streaming; tujuh route dokumen | P-023 |
| `docs/design/44-SECURITY.md` §3.1.3/§4.2 | Rujukan implementasi kedua (cakupan satu fungsi); daftar kebijakan unggahan menjadi bentuk yang berjalan (`limitedReader`, `422` bukan `413`) | P-023 |
| `docs/design/50-FSD.md` §4.2/§4.3 | Tabel versi minor/major, owner = pembuat, syarat "no workflow running" ditegakkan `409` | P-023 |
| `docs/design/41-DATABASE.md` §2.3 | Komentar kolom `checksum`: heksadesimal 64 karakter tanpa prefiks | P-023 |
| `docs/design/70-TESTING.md` §3.5/§3.8/§3.9 | Status §3.5 (dijalankan) + letak test penolakan nomor; §3.8 daftar test modul dokumen; §3.9 aturan test deterministik (C-039) | P-023 |
| `docs/progress/TRACEABILITY.md` | FR-DOC-01..07 → DONE (FR-DOC-07 PARTIAL), enam baris FR-VER-01..06 ditambahkan, FR-AUDIT-01 diperluas | P-023 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-016 baru (delapan kontrak modul document) | P-023 |
| `docs/progress/audits/AUDIT-001-...md` + `audits/README.md` | C-039..C-042 (FIXED) dan hitungan 42 temuan / 31 FIXED / 10 OPEN | P-023 |
| `docs/progress/TASKS.md` | `T-037` DONE; `T-024` 28/51 → **35/51** | P-023 |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `CONTINUE.md`, `AGENTS.md` | Ledger diselaraskan dengan modul document + aturan yang kini mengikat | P-023 |
| Database dev `bwdcs` (bukan berkas) | Data uji (project `DOC-UJI`, dua dokumen, dua versi, satu user `scopedoc-uji`) dihapus sesudah dipakai supaya suite tetap hijau (C-038); entri `audit_logs` milik **admin** tetap ada (append-only), sedangkan entri milik user uji dihapus lewat jalur pemeliharaan `bwdcs.audit_maintenance` seperti teardown test | P-023 |

#### Penutup P-023 (lanjutan turn setelah restart Freebuff)

Turn P-023 terputus dua kali, sehingga kode dan dokumen sudah ada di disk tetapi **log promptnya belum
terbentuk** padahal `TASKS.md`, `STATE.md`, `CONTINUE.md`, `70-TESTING.md`, dan footer `AUDIT-001` sudah
merujuk namanya. Penutup ini **tidak mengubah kode**; yang dilakukan adalah memverifikasi ulang apa yang
tertulis dan merapikan ledger yang belum sinkron.

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-023-2026-09-19-modul-document-unggah-versi-dan-penomoran.md` | **Dibuat**: log prompt yang dirujuk lima dokumen tetapi tidak ada berkasnya. Memuat ringkasan aksi, daftar berkas, dan §6.1 verifikasi ulang pada penutup sesi (bukan klaim ulang) | P-023 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Footer "Riwayat" menyebut nama berkas log P-023 yang **berbeda** dari empat dokumen lain (`…-modul-document-dan-penomoran.md`) — dijadikan satu nama kanonik | P-023 |
| `docs/progress/SESSION-LOG.md` | Entri P-023 memuat klaim `is_current=true`/`false` pada `document_versions`; **klaim itu salah** — kolom itu tidak ada di skema, kode, maupun dokumen mana pun (versi "terkini" turunan `documents.current_version`). Dikoreksi lewat entri baru (log ini append-only), bukan dengan menulis ulang entri lama | P-023 |

**Verifikasi ulang penutup (semua dijalankan pada kondisi repo apa adanya):** `gofmt -l .` bersih, `go vet ./...`
bisu, `go build ./...` sukses, `find internal cmd -name '*.go' -newer bin/bwdcs` kosong (server hidup memuat
kode ini), `GET /health` → `200 healthy`. Suite dijalankan **dengan `TEST_DATABASE_URL`** — `go test ./... -p 1 -count=1`
seluruh paket `ok`, dan tanpa variabel itu test integrasi `SKIP` sehingga hijau sebelumnya bukan bukti: 14 test
dokumen di `internal/service` dan 11 di `internal/handler` PASS (0 SKIP). Alur HTTP ulang penuh pada server nyata
(`DOC-UJI-001`/`DOC-UJI-002`, versi `1.0` → `1.1`, unduhan `cmp` identik, non-anggota `404` pada detail/versi/unduh,
`viewer` unggah `403`, `DELETE` `200` + kaskade) memberi hasil yang sama, lalu data uji dibersihkan
(`projects=0 documents=0 versions=0 seq=0 users=1`, storage kosong). Catatan operasional baru: user uji **tidak
dapat** dihapus selama ada entri `audit_logs` miliknya (`actor_id` → `ON DELETE RESTRICT`), jadi pembersihannya
memakai jalur pemeliharaan `SET LOCAL bwdcs.audit_maintenance = 'on'` — sama seperti teardown test.

---

## 2026-09-19 (sesi P-022)

Mengerjakan **`T-035`**: modul **Project** — `POST`/`GET`/`PATCH`/`archive` project, anggota project, dan
**cakupan data anggota diterapkan di dalam kueri** (`44-SECURITY.md` §3.1.3). Delapan endpoint `42-API.md` §3
kini hidup; bukti utamanya dijalankan pada server nyata: user ber-izin `project:read` yang **bukan anggota**
melihat daftar kosong dan detail **404**, lalu melihat project itu begitu dijadikan anggota; Administrator
organisasi lain tetap **404**. Keputusan agen yang perlu konfirmasi Anda dicatat di `OPEN-QUESTIONS.md` **Q-015**.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/model/project.go` | Model `Project`/`ProjectMember` + konstanta status (`active`/`archived`, ADR-0012) dan role project (`owner`/`manager`/`contributor`/`viewer`, FR-PROJ-05) | P-022 |
| `backend/internal/repository/project_repository.go` | Repository project: `ProjectScope` diterapkan **di dalam `WHERE`** (`projectScopePredicate`), daftar berfilter, `FindByID` dalam cakupan, `Create`/`Update` (whitelist kolom)/`Archive`, keanggotaan (`Members`/`AddMember`/`UpsertMember`/`RemoveMember`/`MemberRole`) | P-022 |
| `backend/internal/service/project_service.go` | `Scope` (role sistem → cakupan), `Create`/`Update`/`Archive`/`AddMember`/`RemoveMember` dalam satu transaksi bersama audit (ADR-0011); invariant "owner selalu anggota"; kode permanen (ADR-0017) | P-022 |
| `backend/internal/dto/project_dto.go` | Bentuk request/response `42-API.md` §3 + tipe `Date` (`YYYY-MM-DD`) + normalisasi/pola `code` (ADR-0017) | P-022 |
| `backend/internal/handler/project_handler.go` | Delapan handler + validasi 422 berstruktur + pemetaan error domain → status HTTP di satu fungsi | P-022 |
| `backend/internal/service/project_service_test.go` | 14 test integrasi: cakupan anggota vs Administrator vs tenant lain, konflik kode, kode permanen, urutan tanggal, arsip idempotent, siklus anggota, pemindahan owner, audit per aksi | P-022 |
| `backend/internal/handler/project_handler_test.go` | 10 test end-to-end HTTP: 401 untuk kedelapan endpoint tanpa token, 403 viewer/contributor, 201 + normalisasi kode, 7 kasus 422, 409 (kode & anggota & owner), cakupan (0/404/200), validasi query | P-022 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/repository/db.go` | Tambah `ErrDuplicate` (pelanggaran constraint unik → 409) dan `ErrNoUpdateFields` (PATCH tanpa field → 422) sebagai error domain repository | P-022 |
| `backend/internal/pkg/response/response.go` | Tambah `OKWithMeta` supaya endpoint daftar tidak menyusun blok `meta` sendiri (`42-API.md` §1) | P-022 |
| `backend/internal/handler/router.go` | Daftarkan 8 route `/projects` dengan `RequirePermission` dari matriks ADR-0014; `RouterDeps.Project` (boleh nil untuk engine tanpa modul itu) | P-022 |
| `backend/internal/handler/main_test.go` | Rakit `ProjectService` di engine test supaya test auth lama tetap memakai jalan yang sama dengan produksi | P-022 |
| `backend/cmd/server/main.go` | Rakit `repository.NewProjectRepository` + `service.NewProjectService` dan serahkan ke `handler.Setup` | P-022 |
| `docs/design/42-API.md` | §1: `limit` 1–100 → 422 (tidak dipotong diam-diam). §3 ditulis ulang: bentuk kanonik project, aturan query, `Izin:` per endpoint (8/8), cakupan data, aturan `code`/owner/tanggal, 404 vs 403, 409 (kode, anggota duplikat, owner). §12: sebab `404` diperluas ke "di luar cakupan" dan daftar `409` menyebut kasus project | P-022 |
| `docs/design/40-TSD.md` | §2.3 catatan bahwa tag `validate:` di sketsa tidak dipakai implementasi + kolom turunan; §2.4 `ProjectService`; §2.5 `ProjectScope` + `ProjectRepository`; §2.6 pola handler (pemetaan error terpusat, cakupan di service); §6 route project lengkap 8 endpoint | P-022 |
| `docs/design/44-SECURITY.md` | §3.1.3: cakupan diterapkan di kueri, pelanggaran cakupan → `404` (bukan `403`), rujukan implementasi pertama (`ProjectScope`) — tetap tanpa bypass izin | P-022 |
| `docs/design/50-FSD.md` | §3.1: arsip = status (bukan hapus) + cakupan daftar; §3.2: aturan server (normalisasi `Code`, owner langsung menjadi anggota); §3.3: role anggota, duplikat & owner tidak dapat dihapus | P-022 |
| `docs/design/70-TESTING.md` | §3.7 baru: daftar test modul project (service + handler) beserta catatan urutan pembersihan fixture (FK `RESTRICT` + audit append-only); §4.1 menunjuk test cakupan project yang sudah ada | P-022 |
| `docs/progress/TRACEABILITY.md` | FR-PROJ-01..07 diisi (DONE) dan FR-AUDIT-01 naik dari TODO → PARTIAL dengan bukti per aksi | P-022 |
| `docs/progress/TASKS.md` | `T-035` masuk DONE dengan bukti; `T-024` diperbarui 20/51 → **28/51** (bab projects beranotasi) | P-022 |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-015** baru: tiga kontrak yang diputuskan agen (owner selalu anggota, 404 untuk pelanggaran cakupan, batas `limit`) — NON-BLOCKING, minta konfirmasi | P-022 |
| `docs/progress/STATE.md`, `docs/progress/SESSION-LOG.md`, `CONTINUE.md`, `AGENTS.md` | Ledger diselaraskan dengan kondisi sesudah modul project | P-022 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` + `audits/README.md` | Temuan baru **C-038** (OPEN): `TEST_DATABASE_URL` menunjuk database dev yang sama dengan server, sehingga project nyata membuat lima test `internal/bootstrap` gagal `23503`; hitungan diperbarui 37→38 temuan, 28 FIXED, 9→10 OPEN | P-022 |
| `docs/design/70-TESTING.md` §8.1 (catatan) | Menyebut C-038 di samping aturan `TEST_DATABASE_URL` supaya agen berikutnya tidak mengejar gejala yang sama | P-022 |
| Database dev `bwdcs` (bukan berkas) | Dua project demo hasil verifikasi (`DEMO-PRJ`, `DEMO-2`) dihapus sesudah dipakai, karena keberadaannya membuat suite gagal (C-038); entri `audit_logs` miliknya tetap ada (append-only, memang tidak dapat dihapus) | P-022 |
| `.freebuff/run.md`, `.freebuff/preview.html` | Daftar endpoint yang hidup ditambah bab Projects; prosedur menjalankan server tetap sama | P-022 |

### Removed

| File | Alasan | Prompt |
|---|---|---|
| `backend/tmp-probe/main.go` | Alat diagnosis sementara (menirukan urutan startup server untuk mencari penyebab preview tidak ter-spawn di bawah launchd) — dibuat dan dihapus dalam sesi yang sama; tidak ada artefak yang tersisa | sesi preview (thread yang sama, sebelum P-022) |

#### Penutup P-022 (lanjutan turn setelah restart Freebuff)

Sesi P-022 dilanjutkan untuk memastikan kode di disk benar-benar sesuai ledger dan tidak ada hitungan yang usang. Tidak ada berkas kode yang diubah; yang berubah hanya ledger.

| File | Alasan | Prompt |
|---|---|---|
| `AGENTS.md` | Hitungan audit masih lama (37 temuan, 9 OPEN) padahal `AUDIT-001` sudah 38 temuan / 10 OPEN; ditambah aturan modul yang kini mengikat: cakupan data di `WHERE` (bukan middleware), pelanggaran cakupan → `404`, owner selalu anggota, `code` permanen → `409` | P-022 |
| `docs/progress/STATE.md` | Menghapus baris `Project` ganda di §3 (satu baris "Selesai", satu sisa "Belum"); Q-006/Q-007 ditandai `RESOLVED` seperti di `OPEN-QUESTIONS.md`; Q-010 9 → **10** temuan (C-038 butuh izin, bukan keputusan); **Q-015** ditambahkan ke tabel keputusan pending | P-022 |
| `docs/progress/TASKS.md` | `T-017` diperbarui dari 9 → **10** temuan OPEN (menyertakan C-038 → `T-036`) supaya tidak tampak lebih sempit dari kenyataan | P-022 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Footer "Riwayat" belum menyebut `P-022` (sesi yang menambahkan C-038) | P-022 |
| `docs/progress/SESSION-LOG.md` | Entri P-022 ditambah penutup: verifikasi ulang persis seperti §6 (build/vet/test + bukti HTTP `SMOKE-01`) dan daftar koreksi ledger di atas | P-022 |

---

## 2026-09-18 (sesi P-021)

Mengerjakan **`T-005`**: modul auth lengkap — login (`FR-AUTH-01/02`), JWT ber-`jti` (`FR-AUTH-03`), middleware RBAC dari matriks ADR-0014 (`FR-ROLE-03`), rate limit `FR-AUTH-06`, dan logout + daftar revokasi `token_revocations` (ADR-0009, `FR-AUTH-04`). Bukti utamanya bukan test unit: **admin pertama login lewat HTTP** (`POST /api/v1/auth/login` → 200 + token), memakai token di `/api/v1/auth/me` (200, 44 izin), logout, lalu token yang sama ditolak `401 TOKEN_REVOKED`. Lima temuan audit baru (**C-033**–**C-037**) muncul karena modul ini benar-benar **ditulis dan dijalankan**; dua di antaranya menandai janji kontrak yang tidak dapat dijalankan tanpa keputusan Anda, bukan salah tulis. Sekaligus menutup utang **`T-033`** (test config hermetis).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `backend/internal/pkg/jwt/jwt.go` + `jwt_test.go` | Penerbit/validator token HS256 dengan `jti` wajib (ADR-0009); 9 test termasuk alg none, issuer lain, tanpa `exp`, tanpa `jti`, kedaluwarsa | P-021 |
| `backend/internal/pkg/response/response.go` | Amplop response + kode error kanonik `42-API.md` §12 supaya handler tidak menulis JSON sendiri | P-021 |
| `backend/internal/model/user.go` | Model `User` + `Permission` (`40-TSD.md` §2.3) | P-021 |
| `backend/internal/repository/db.go` | `DBTX` (pool **atau** transaksi) sesuai ADR-0011 butir 3 + `ErrNotFound` | P-021 |
| `backend/internal/repository/user_repository.go` | Baca user/role/permission, `HasPermission` langsung dari `role_permissions`, `UpdatePasswordHash`, `WithTx` | P-021 |
| `backend/internal/repository/token_revocation_repository.go` | `Revoke`/`IsRevoked` (cache TTL 30 detik, invalidasi seketika saat logout)/`CleanupExpired` (ADR-0009) | P-021 |
| `backend/internal/repository/setting_repository.go` | Kebijakan login dari `system_settings` (`auth.max_login_attempts`, `auth.lockout_duration_minutes`) — tanpa menambah environment variable | P-021 |
| `backend/internal/service/{auth_service,login_guard,permission_checker,audit_service}.go` + test | Login/logout/profil, batas percobaan gagal per username (FR-AUTH-06), pemeriksa izin tanpa bypass Administrator, penulisan audit di transaksi pemanggil (ADR-0011) | P-021 |
| `backend/internal/middleware/{context,auth,permission,rate_limit,correlation,logger,cors}.go` + `auth_test.go` | Enam middleware `40-TSD.md` §2.2; interface kecil untuk dependensinya sehingga dapat diuji tanpa database (10 test) | P-021 |
| `backend/internal/dto/auth_dto.go`, `backend/internal/handler/{auth_handler,router}.go` + `main_test.go`, `auth_handler_test.go` | Endpoint `42-API.md` §2 + pemasangan route; 9 test end-to-end lewat engine nyata | P-021 |
| `docs/progress/prompts/P-021-2026-09-18-auth-login-jwt-rbac-dan-revokasi-token.md` | Log prompt sesi T-005 | P-021 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/cmd/server/main.go` | Merakit modul auth (JWT, repository, service, middleware, route) + pembersihan `token_revocations` saat startup dan setiap 1 jam (ADR-0009 butir 5) | P-021 |
| `backend/go.mod`, `go.sum` | Tambah `golang-jwt/jwt/v5` v5.3.1 dan `google/uuid` v1.6.0 (tidak menaikkan direktif `go`) | P-021 |
| `backend/Makefile` | `test` memakai `-p 1`: paket test integrasi berbagi satu database ( **C-036** ) | P-021 |
| `backend/internal/bootstrap/bootstrap_test.go` | `newCleanTx` membersihkan lewat jalur pemeliharaan `SET LOCAL bwdcs.audit_maintenance = 'on'` di dalam transaksi yang digulung balik — tanpa itu `DELETE FROM users` ditolak `audit_logs_actor_id_fkey` begitu ada entri audit nyata (login admin `T-005`), dan seluruh paket test gagal ( **C-036** ) | P-021 |
| `backend/internal/config/{config_test,envfile_test}.go` | Helper `clearEnv(t)` membuat test hermetis terhadap environment pemanggil (viper `AutomaticEnv` selalu menang atas `.env`) — menutup `T-033` | P-021 |
| `docs/design/42-API.md` | §2: anotasi `Izin:` kelima endpoint auth (T-024: 15/51 → 20/51), perilaku `logout_all` (`501`), dan penanda `refresh`/`change-password` belum dijalankan; §12: kode `401`/`403`/`429`/`501` diberi tabel kode eksplisit | P-021 |
| `docs/design/40-TSD.md` | §2.2: interface middleware nyata (package `auth` hantu dihapus — **C-034**) + pembagian rate limit; §2.4: `AuditService.Log(ctx, tx pgx.Tx, …)`; §5.2.2 baru: batas percobaan login + batas auto-lock (C-009); §6: lokasi `Setup` | P-021 |
| `docs/design/44-SECURITY.md` | §2.3: batas auto-lock dan audit login gagal dinyatakan eksplisit; §3.2: butir "admin bypass" diganti larangan tegas (**C-037**) | P-021 |
| `docs/design/70-TESTING.md` | §8: peringatan bahwa test integrasi berbagi satu database → jalankan serial, plus catatan pembersihan data yang sudah commit (`-p 1`, **C-036**) | P-021 |
| `docs/progress/audits/AUDIT-001-...md`, `docs/progress/audits/README.md` | Tambah C-033..C-037 dan perbarui hitungan: 37 temuan, 28 FIXED, 9 OPEN | P-021 |
| `docs/progress/OPEN-QUESTIONS.md` | Tambah **Q-013** (mekanisme pencabutan sesi) dan **Q-014** (audit login gagal) | P-021 |
| `docs/progress/TASKS.md` | `T-005` dan `T-033` pindah ke DONE dengan bukti; `T-034` baru (refresh + change-password, menunggu Q-013); `T-024` diperbarui ke 20/51 | P-021 |
| `docs/progress/TRACEABILITY.md` | `FR-AUTH-01`..`FR-AUTH-06` dan `FR-ROLE-03` → DONE; baris `FR-AUTH-03`, `FR-AUTH-04`, `FR-AUTH-07` ditambahkan; `FR-AUTH-07` PARTIAL | P-021 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md`, `docs/progress/SESSION-LOG.md` | Ledger diselaraskan: Phase 0 selesai, audit 37/28/9, aturan auth yang mengikat, next action Phase 1 | P-021 |

---

## 2026-09-18 (sesi P-020)

Mengerjakan **`T-004`**: sembilan migrasi `001`-`009` (skema `41-DATABASE.md` §2 + trigger append-only `007` + seed permission `008`), runner migrasi yang di-embed (ADR-0018), dan **bootstrap organisasi + admin pertama** (ADR-0010) yang kini berjalan dalam urutan startup. Empat temuan audit baru (**C-029**–**C-032**) muncul karena migrasi benar-benar **dijalankan** — bukan dibaca — dan semuanya ditutup di sesi yang sama.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0018-migrasi-di-embed-dan-dijalankan-saat-startup.md` | ADR baru (ACCEPTED): migrasi di-embed & dijalankan saat startup, `internal/migration` menjadi package (memperbarui ADR-0013 butir 1), goose dipin v3.24.1 library+CLI | P-020 |
| `backend/internal/migration/001_create_organizations.sql` … `009_create_token_revocations.sql` | Sembilan migrasi skema + seed; sumber DDL `41-DATABASE.md` §2/§4 | P-020 |
| `backend/internal/migration/migration.go` | Embed (`//go:embed *.sql`) + runner goose; mencatat versi skema ke log (ADR-0018) | P-020 |
| `backend/internal/migration/main_test.go`, `migration_test.go`, `audit_append_only_test.go` | 13 test: kelengkapan 21 tabel, FK tertunda C-029, seed 44/30/18/12, kosakata tertutup, spot check matriks, tanpa duplikat, `system_settings`, dan enam test append-only `70-TESTING.md` §4.3 | P-020 |
| `backend/internal/bootstrap/bootstrap.go` | `EnsureAdminFirstRun` (satu transaksi, idempotent, validasi sebelum menulis), `ValidatePassword`, `Querier` (agar test dapat memakai transaksi yang digulung balik) | P-020 |
| `backend/internal/bootstrap/bootstrap_test.go` | 10 test: password lemah/contoh ditolak, org+admin+role dibuat, bcrypt cost 12 dan hash ≠ password, idempotent, pesan bila seed `008` hilang, dilewati saat database sudah berisi user | P-020 |
| `docs/progress/prompts/P-020-2026-09-18-migrasi-001-009-dan-bootstrap-admin.md` | Log prompt sesi T-004 | P-020 |

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `backend/cmd/server/main.go` | Urutan startup diisi sesuai ADR-0010 butir 1: migrasi (embed) → bootstrap → HTTP; komentar TODO `T-004` dihapus | P-020 |
| `backend/go.mod`, `backend/go.sum` | `pressly/goose/v3` v3.24.1 + `golang.org/x/crypto` v0.31.0 sebagai dependensi langsung (bcrypt); pin karena toolchain Go 1.22.5 | P-020 |
| `docs/design/41-DATABASE.md` | §2.3 komentar FK tertunda; §2.6 penunjuk rumah migrasi; §4 tabel pemetaan isi sembilan berkas + catatan urutan FK (C-029), penempatan `system_settings` (C-030), dan pembungkus anotasi goose (C-031) | P-020 |
| `docs/design/44-SECURITY.md` | §6: butir baru — badan fungsi wajib dibungkus `StatementBegin`/`StatementEnd`, dan larangan menulis penanda anotasi goose di komentar biasa (C-031) | P-020 |
| `docs/design/40-TSD.md` | §2.0: `migration/` kini memuat package kecil (ADR-0018) + catatan pin `pgx` **dan** `goose` v3.24.1 | P-020 |
| `docs/design/60-DEPLOYMENT.md` | §4.2: berkas migrasi tidak perlu ikut di-deploy karena di-embed (ADR-0018) | P-020 |
| `docs/design/70-TESTING.md` | §4.3: path test menjadi `internal/migration/audit_append_only_test.go` (yang diuji objek skema) + catatan SAVEPOINT wajib saat menguji penolakan; §4.1: penunjuk implementasi `TestSeedRolePermissions_RowCounts` | P-020 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §3 catatan status langkah 4 (bukti "admin pertama dapat login" menunggu `T-005`, sisanya terbukti pada P-020); §3.1 pre-flight item 2, 3, dan 6a dari "sebagian/menunggu izin/belum dibuat" menjadi selesai | P-020 |
| `docs/adr/README.md` | Baris ADR-0018 ditambahkan; ADR-0013 ditandai "butir 1 diperbarui ADR-0018" | P-020 |
| `docs/progress/audits/AUDIT-001-...md`, `docs/progress/audits/README.md` | C-029–C-032 ditambahkan dan `FIXED`; ringkasan 32 temuan / 25 FIXED / 7 OPEN | P-020 |
| `docs/progress/TASKS.md` | `T-004` pindah ke DONE; `T-017` diperbarui; `T-012` diberi catatan pin CLI; **`T-033` baru** (test config hermetis) | P-020 |
| `docs/progress/TRACEABILITY.md` | `FR-AUTH-05`, `FR-ROLE-01`/`FR-ROLE-02`/`FR-ROLE-03`, `FR-DOC-02` → `PARTIAL` dengan implementasi + test; **baris baru `FR-ORG-02`** (sebelumnya tidak ada) | P-020 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md`, `OPEN-QUESTIONS.md`, `SESSION-LOG.md` | Ledger sesi P-020: skema terpasang, pin goose, audit 32/25/7, next action `T-005` | P-020 |

---

## 2026-09-18 (sesi P-019)

Memperbaiki temuan **C-020**: cuplikan trigger "audit log immutable" di `44-SECURITY.md` §6 memanggil `raise_exception()` yang tidak ada di PostgreSQL **dan** tidak pernah dipasang migrasi mana pun — jadi janji FR-AUDIT-03 tidak dapat dijalankan apa adanya. Diganti pola PL/pgSQL yang sudah diuji pada PostgreSQL 16.10 dan diikat ke migrasi `007`, dengan test baru di `70-TESTING.md` §4.3. Satu temuan baru dicatat (**C-028**).

### Changed

| File | Alasan | Prompt |
|---|---|---|
| `docs/design/44-SECURITY.md` | §6: fungsi `prevent_audit_modification()` + dua trigger (row-level `UPDATE`/`DELETE`, statement-level `TRUNCATE`) yang menolak `23001`; jalur pemeliharaan `bwdcs.audit_maintenance`; alasan `REVOKE` tidak dipakai; alasan `document_versions` tidak diberi trigger; down migration. §8: checklist test append-only | P-019 |
| `docs/design/41-DATABASE.md` | §2.5: penunjuk penegakan append-only ke `44-SECURITY.md` §6; §4: isi migrasi `007` kini memuat kedua trigger (sebelumnya tidak ada migrasi yang memasangnya) | P-019 |
| `docs/design/70-TESTING.md` | §4.3 baru: enam test append-only (UPDATE/DELETE/TRUNCATE ditolak, INSERT lolos, dua trigger terpasang, GUC harus opt-in); §8: catatan `teardownTestDB` memakai GUC pemeliharaan | P-019 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-020 → `FIXED`; C-028 (baru) → `OPEN`; ringkasan menjadi 28 temuan / 21 FIXED / 7 OPEN | P-019 |
| `docs/progress/audits/README.md` | Baris AUDIT-001: 28 (14 S1, 10 S2, 4 S3), 21 FIXED, 7 OPEN; catatan temuan yang ditambahkan menyusul diperluas ke C-024..C-028 | P-019 |
| `docs/progress/TASKS.md` | `T-032` ditambahkan di tabel DONE; `T-017` diperbarui (C-020 selesai, sisa OPEN kini C-028) | P-019 |
| `docs/progress/TRACEABILITY.md` | Baris `FR-AUDIT-03`: kolom Desain (trigger + migrasi `007`) dan Test (`70-TESTING.md` §4.3) diisi; catatan P-019 | P-019 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-010: hasil P-019 + sisa OPEN diperbarui; **Q-012** baru (retensi audit log, opsi A/B) | P-019 |
| `docs/progress/STATE.md` | Audit (28/21/7), konvensi append-only pada baris konvensi, task aktif, next action, tabel file penting | P-019 |
| `docs/progress/SESSION-LOG.md` | Entri P-019 | P-019 |
| `CONTINUE.md` | Header (prompt terakhir P-019, berikutnya P-020) + blok §0 diselaraskan | P-019 |
| `AGENTS.md` | Status audit 28/21/7 + larangan membuat trigger append-only kedua sendiri (trigger resmi dipasang migrasi `007`) | P-019 |

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-019-2026-09-18-trigger-append-only-audit-log.md` | Log prompt sesi perbaikan C-020 | P-019 |

---

## 2026-09-18 (sesi P-018)

Menjalankan urutan **Phase 0** setelah user memberi izin (Q-004 `git init`, Q-009 toolchain): PATH toolchain (`T-011`), role + database `bwdcs` di PostgreSQL 16.10 yang sudah berjalan (`T-013`), `goose` (`T-012`), repo git + struktur folder (`T-002`/`T-002a`), dan backend skeleton yang benar-benar dibangun, diuji, serta menjawab `GET /health` 200 (`T-003`). Satu temuan audit baru yang muncul saat menulis kode — **package** `filestorage` tidak bisa dipakai sebagaimana didokumentasikan — dicatat sebagai C-026/C-027 dan diperbaiki di sesi yang sama.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `.gitignore` | Melindungi `.env` (T-002a), `storage/`, `bin/`, `node_modules/`, `.freebuff/` | P-018 |
| `.editorconfig` | Konvensi editor: tab untuk Go, 2 spasi untuk TS/JS/JSON, LF, final newline | P-018 |
| `backend/go.mod`, `backend/go.sum` | Module `bwdcs/backend`, `go 1.22.5`; gin v1.10.0, pgx v5.7.4, viper v1.19.0 | P-018 |
| `backend/cmd/server/main.go` | Entry point sesuai urutan `60-DEPLOYMENT.md` §4.2: config, storage, pool pgx (lazy), HTTP server, shutdown rapi, TODO `T-004`/`T-005` | P-018 |
| `backend/internal/config/config.go` | Loader viper: `.env` opsional, environment menang, validasi env wajib yang melaporkan semua masalah sekaligus | P-018 |
| `backend/internal/config/config_test.go` | Test env wajib hilang/pendek, durasi & storage type tidak valid, default non-rahasia | P-018 |
| `backend/internal/config/envfile_test.go` | Test pembacaan `.env`, kutipan nilai berspasi, prioritas environment | P-018 |
| `backend/internal/pkg/filestorage/filestorage.go` | Kontrak `FileStorage` (ADR-0013/ADR-0005) + `Prober` untuk `GET /health` | P-018 |
| `backend/internal/pkg/filestorage/local.go` | Implementasi lokal: skema key `orgs/{org}/projects/{proj}/docs/{doc}/{version}/{nama}`, sanitasi nama, tolak path traversal, tolak penimpaan versi | P-018 |
| `backend/internal/pkg/filestorage/local_test.go` | 9 test: bentuk key, anti-traversal, imutabilitas versi, `Delete` idempotent, `Ping` | P-018 |
| `backend/internal/handler/health_handler.go` | `GET /health` dengan pemeriksaan database + storage | P-018 |
| `backend/Makefile` | Target `60-DEPLOYMENT.md` §3.1 + `vet`/`fmt`/`migrate-status`; memuat `../.env` bila ada | P-018 |
| `frontend/README.md` | Penanda blokir Phase 4 (`DESIGN.md`), bukan skeleton aplikasi | P-018 |
| `docs/progress/prompts/P-018-2026-09-18-phase-0-toolchain-dan-skeleton-backend.md` | Log prompt sesi ini | P-018 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `.env` (lokal, tidak di-commit) | Nilai berspasi dikutip; kredensial DB `bwdcs`, `JWT_SECRET` acak, `APP_PORT=8081`, nilai `ADMIN_*` | Tanpa kutip, `set -a; . .env` memotong nilai pada spasi dan menjalankan sisa baris sebagai perintah | P-018 |
| `.env.example` | Kutipan `ADMIN_ORG_NAME` + catatan aturan kutipan; penanda `.gitignore` sudah ada (bukan "belum dibuat") | Cermin yang aman disalin dan di-source | P-018 |
| `docs/design/40-TSD.md` | §2.0 nama module `bwdcs/backend` + catatan toolchain (pgx v5.7.4, batas Go 1.22.5); §2.1 field `Bootstrap` + perilaku loader; §2.4 `Save(..., originalName string, ...)` + skema key + imutabilitas | Tiga kontrak yang dibutuhkan kode belum ada/kurang (C-026) | P-018 |
| `docs/design/60-DEPLOYMENT.md` | §5: cuplikan health check memakai `storage.Ping` dan dapat dikompilasi | Cuplikan lama tidak dapat dibangun dan membaca kesehatan storage secara terbalik (C-027) | P-018 |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | Temuan C-026/C-027 ditambahkan dan ditutup; hitungan 27 temuan / 20 FIXED / 7 OPEN | Disiplin jejak keputusan | P-018 |
| `docs/progress/TASKS.md` | `T-002`, `T-002a`, `T-003`, `T-011`, `T-012`, `T-013`, `T-031` → DONE; catatan blocker diperbarui; **`T-024` dipindahkan dari DONE ke TODO** (15/51 endpoint beranotasi `Izin:`) | Papan status harus mencerminkan kenyataan; baris itu sebelumnya terbaca selesai padahal 36 endpoint belum | P-018 |
| `docs/progress/TRACEABILITY.md` | NFR-MAIN-01/02/03 → PARTIAL dengan bukti; NFR-PORT-01 diperbarui (jalur tanpa Docker terbukti, Dockerfile belum ada) | Requirement yang mulai dikerjakan wajib punya baris | P-018 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-004 dan Q-009 dijawab user dengan izin → `RESOLVED` | Izin harus tercatat sebelum bekerja | P-018 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Posisi Phase 0, tabel toolchain, status audit, dan next action `T-004` | Snapshot harus selaras | P-018 |
| `~/.zshrc` (di luar repo) | Blok `BWDCS toolchain` idempoten | `go`, `goose`, dan `psql` 16 harus tersedia di shell baru | P-018 |

---

## 2026-09-18 (sesi P-017)

Menyelesaikan task **`T-028`**: kontrak endpoint re-submit setelah revisi ditetapkan di `42-API.md` §5 sebagai `POST /workflows/instances/:id/resubmit` — melanjutkan **instance yang sama** dengan izin `workflow_instance:submit` (matriks ADR-0014), tanpa instance baru dan tanpa perubahan skema. Sesi ini juga menutup temuan baru **C-025**: aturan "satu aksi per step" di `43-WORKFLOW.md` §4.2 membuat alur revisi ADR-0016 mustahil dijalankan.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-017-2026-09-18-kontrak-endpoint-resubmit.md` | Log prompt sesi ini | P-017 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/42-API.md` | §5: endpoint `POST /workflows/instances/:id/resubmit` (5 prasyarat, dua guard, efek, izin, daftar "yang tidak terjadi"); jeda revisi pada `/actions`; `document_status` pada endpoint daftar; rujukan `T-028` yang menggantung dihapus | Tanpa kontrak, agen akan menebak cara melanjutkan review setelah revisi | P-017 |
| `docs/design/43-WORKFLOW.md` | §4.6 baru (perilaku engine re-submit: instance sama, deadline dihitung ulang, tanpa `workflow_actions`, jeda, siklus); §4.2 langkah 4 jadi "per siklus"; §4.5 butir 3 (rujukan `T-027` → kontrak `42-API.md` §5) | Aturan lama memblokir reviewer pada step hasil rollback (C-025) | P-017 |
| `docs/design/40-TSD.md` | §2.4 `WorkflowService.Resubmit`; §6 route resubmit dengan `RequirePermission("workflow_instance", "submit")` | Antarmuka dan wiring harus ikut kontrak | P-017 |
| `docs/design/50-FSD.md` | §4.3 aksi "Resubmit for Review"; §5.2 tombol berubah setelah revisi + Approval Panel nonaktif saat jeda; §5.4 tab Pending menyaring `document_status=revision_required` | UI tidak boleh menawarkan aksi yang ditolak API | P-017 |
| `docs/design/70-TESTING.md` | §3.6 tujuh test re-submit (instance sama, deadline, wajib versi baru, status salah, stale version, jeda, siklus) | Janji perilaku harus punya test | P-017 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-025 ditambahkan (S1) dan `FIXED`; S1 11 → 12; ringkasan 25 temuan / 18 FIXED / 7 OPEN | Mengikuti aturan audit §2.2/§2.4 | P-017 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 25 temuan (12 S1, 9 S2, 4 S3), 18 FIXED, 7 OPEN | Audit harus mencerminkan cakupan sebenarnya | P-017 |
| `docs/progress/TASKS.md` | `T-028` dipindah dari TODO ke DONE; `T-017` diperbarui | Protokol progress | P-017 |
| `docs/progress/TRACEABILITY.md` | `FR-WF-09` kolom Desain + §4.6; catatan P-017 | Requirement harus menunjuk desainnya | P-017 |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-011 baru** (notifikasi overdue saat jeda; unggahan saat `in_review`); Q-010 hasil P-017 | Ambiguitas dicatat, tidak ditebak | P-017 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md`, `SESSION-LOG.md` | Snapshot, status audit 25/18/7, 51 endpoint, file sesi terakhir | Protokol progress wajib | P-017 |

---

## 2026-09-18 (sesi P-016)

Perbaikan temuan **C-016**: format dan pemberian nomor dokumen ditetapkan lewat **ADR-0017** — `{PROJECT_CODE}-{NNN}` selalu dibangkitkan server (atomik di `document_sequences`), immutable, tanpa penomoran manual, dan `projects.code` menjadi permanen karena dipakai sebagai prefiks. Tujuannya agar migrasi `004` dan validasinya di `T-004` tidak lagi dikarang per agen. Sekaligus ditutup **C-024** (requirement ID hantu `FR-DOC-08`) yang ditemukan di modul yang sama.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0017-format-nomor-dokumen.md` | Keputusan arsitektur penomoran dokumen: format, pembangkit atomik, imutabilitas, 6 alternatif ditolak. Diwajibkan karena temuan ini mengubah perilaku yang terlihat user dan menentukan validasi migrasi | P-016 |
| `docs/progress/prompts/P-016-2026-09-18-format-nomor-dokumen.md` | Log prompt sesi ini | P-016 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/adr/README.md` | Index: baris ADR-0017 | Setiap ADR wajib muncul di index | P-016 |
| `docs/design/41-DATABASE.md` | §2.2 komentar `projects.code` (UPPERCASE, pola, permanen); §2.3 tabel `document_sequences` + komentar `document_number`; §3 dua baris index; §4 prosedur isi migrasi `004` + kueri pembangkit | Skema dan migrasi tidak boleh ditafsirkan berbeda antar agen | P-016 |
| `docs/design/42-API.md` | §3 `code` project (normalisasi, pola, unik, tidak dapat diubah; `PATCH` menolak `code`); §4 `POST /documents` tidak lagi menerima `document_number` + aturan nomor; §9 contoh `WEB-001`; §12 paragraf `409` digabung dan contohnya diganti | Kontrak klien harus eksplisit; konflik nomor dokumen tidak lagi dapat dipicu klien | P-016 |
| `docs/design/50-FSD.md` | §3.2 validasi `Code` (UPPERCASE, pola, permanen); §4.2 Document Number menjadi read-only hasil server | Form tidak boleh menawarkan input yang akan ditolak API | P-016 |
| `docs/design/51-UX.md` | Contoh nomor `DOC-2026-001`/`DOC-2026-005` → `WEB-001`/`API-001`; kode project `PROJ-001`/`PROJ-002` → `WEB`/`API` | Mockup tidak boleh mengajarkan format yang tidak sah | P-016 |
| `docs/design/20-SRS.md` | FR-DOC-02 menunjuk ADR-0017 | Requirement High kini punya definisi mengikat | P-016 |
| `docs/design/40-TSD.md` | §2.1 `Project.Code` + `Document.DocumentNumber` diberi komentar ADR-0017; §5.4 `CreateDocumentInput` tanpa `document_number` + aturan pembangkitan di transaksi | DTO dan model tidak boleh membawa field yang bukan input | P-016 |
| `docs/design/44-SECURITY.md` | §4.1 `CreateDocumentInput` tanpa `document_number`; catatan bahwa `alphanum` salah dan pola project code diperiksa regex di handler | Validasi yang tertulis harus dapat dijalankan | P-016 |
| `docs/design/90-AGENT-GUIDE.md` | §3.1 contoh `DocumentService.Create` membangkitkan nomor di dalam transaksi lewat `repo.NextDocumentNumber` | Contoh yang dijiplak agen tidak boleh menyuntikkan nomor dari input | P-016 |
| `docs/design/70-TESTING.md` | §3.1 assert `TEST-001`; §3.5 test penomoran (berurutan, rollback, konkurensi, tolak nomor klien); §5.1 E2E tidak mengisi nomor + assert hasil | Janji perilaku harus punya test, termasuk jalur konkurensi | P-016 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Contoh commit: `[FR-DOC-01, FR-DOC-08]` → `[FR-VER-01, FR-VER-04]` | `FR-DOC-08` tidak ada di SRS (temuan C-024) | P-016 |
| `IDEA.md` | Contoh `Entity ID: DOC-001` → `WEB-001` | Menyelaraskan contoh dengan format yang ditetapkan | P-016 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-016 → `FIXED` (detail + tabel status); C-024 ditambahkan; S3 3 → 4; ringkasan 24 temuan / 17 FIXED / 7 OPEN; §6 menyebut ADR-0017 | Mengikuti aturan audit §2.2/§2.4 | P-016 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 24 temuan (11 S1, 9 S2, 4 S3), 17 FIXED, 7 OPEN | Audit harus mencerminkan cakupan sebenarnya | P-016 |
| `docs/progress/STATE.md`, `TASKS.md` (T-030 DONE, T-004 diperjelas, T-017 diperbarui), `TRACEABILITY.md` (FR-DOC-02), `OPEN-QUESTIONS.md` (Q-010), `CONTINUE.md`, `AGENTS.md` | Ledger sesi P-016 | Protokol progress wajib | P-016 |

---

## 2026-09-18 (sesi P-015)

Perbaikan temuan **C-018**: tiga halaman yang ada di navigasi `51-UX.md` §2.1 tetapi belum punya spec kini dispesifikasikan di `50-FSD.md` — Approvals (§5.4, sebagai **view** workflow instance sesuai usul resolusi audit), Reports (§10.6), Administration > Workflows (§10.7). Endpoint daftar `GET /workflows/instances` ditambahkan ke `42-API.md` §5 sebagai prasyarat halaman antrean. Tanpa ADR baru (tidak ada keputusan arsitektur; halaman mengikuti keputusan yang sudah terkunci).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-015-2026-09-18-spec-fsd-halaman-tanpa-spec.md` | Log prompt sesi ini | P-015 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/50-FSD.md` | §5.4 Halaman Approvals, §10.6 Reports, §10.7 Workflow Definition Management (baru); §5.1 diberi penunjuk ke §10.7 | Menutup C-018: halaman nav tanpa spec tidak dapat diimplementasikan frontend | P-015 |
| `docs/design/42-API.md` | §5: endpoint daftar `GET /workflows/instances` (`?status=&scope=assigned_to_me`); 2 rujukan `T-027` → `T-028`; §11 catatan menunjuk `50-FSD.md` §10.7 | Halaman antrean butuh endpoint daftar; nomor task usang dari renumbering P-014 | P-015 |
| `docs/design/51-UX.md` | §2.1 catatan penunjuk ke tiga spec FSD baru | Nav tidak boleh jadi satu-satunya spec halaman | P-015 |
| `AGENTS.md` | Baris routing "Halaman Approvals"; status audit 15 FIXED / 8 OPEN | Usul resolusi C-018 menyuruh menambah baris routing | P-015 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-018 → `FIXED`; ringkasan 23 temuan / 15 FIXED / 8 OPEN | Mengikuti aturan audit §2.2/§2.4 | P-015 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 15 FIXED, 8 OPEN | Audit harus mencerminkan cakupan sebenarnya | P-015 |
| `docs/progress/STATE.md`, `TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `CONTINUE.md` | Ledger sesi P-015 (T-029 DONE, T-017 diperbarui, Q-010, snapshot §0) | Protokol progress wajib | P-015 |

---

## 2026-09-18 (sesi P-014)

Perbaikan temuan **C-022**: arah rollback aksi `request_revision` diputuskan lewat **ADR-0016** mengikuti FR-WF-09 — kembali ke **step sebelumnya** (`current_step - 1`, batas bawah step 1), bukan reset ke step 1. Satu keputusan menutup kontradiksi antara `20-SRS.md` FR-WF-09 dan `43-WORKFLOW.md` §4.5; kasus tepi step 1 dan jalur re-submit kini eksplisit. Instance tetap `running`, guard ADR-0015 tidak berubah, tanpa perubahan skema.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0016-request-revision-rollback-step-sebelumnya.md` | Keputusan arsitektur: rollback satu langkah sesuai FR-WF-09, kasus tepi step 1, re-submit lanjut instance yang sama, 4 alternatif ditolak. Diwajibkan karena temuan ber-ADR dan perilakunya terlihat user | P-014 |
| `docs/progress/prompts/P-014-2026-09-18-request-revision-rollback-step-sebelumnya.md` | Log prompt sesi ini | P-014 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/43-WORKFLOW.md` | §4.5 ditulis ulang (satu perilaku rollback + batas bawah step 1 + instance tetap `running` + re-submit lanjut instance yang sama); catatan §7 yang salah rujuk C-021 diperbaiki | Bentuk lama menetapkan default "reset ke step 1" + opsi konfigurable yang menentang FR-WF-09 | P-014 |
| `docs/design/20-SRS.md` | FR-WF-09 diberi penunjuk ADR-0016 | Requirement High kini punya definisi mengikat | P-014 |
| `docs/design/42-API.md` | §5: `POST /workflows/submit` hanya untuk dokumen `draft` yang belum punya instance; perilaku aksi `request_revision`; re-submit setelah revisi melanjutkan instance yang sama | Kontrak klien tidak boleh ditebak; dua jalur re-entry tidak boleh hidup bersamaan | P-014 |
| `docs/design/41-DATABASE.md` | §2.4 catatan transisi rollback (guard ADR-0015 tetap berlaku, tanpa perubahan skema) | Skema dan perilaku harus konsisten | P-014 |
| `docs/design/50-FSD.md` | §8.1 menyebut perilaku rollback satu langkah | Dokumen fitur tidak boleh membawa perilaku lama | P-014 |
| `docs/design/70-TESTING.md` | §3.4 test kasus tepi step 1 + rollback satu langkah | Janji perilaku harus punya test, termasuk tepi step 1 | P-014 |
| `docs/adr/README.md` | Index: baris ADR-0016 | Setiap ADR wajib muncul di index | P-014 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-022 → `FIXED`; ringkasan 23 temuan / 14 FIXED / 9 OPEN; catatan P-014 | Mengikuti aturan audit §2.2/§2.4 | P-014 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 14 FIXED, 9 OPEN | Audit harus mencerminkan cakupan sebenarnya | P-014 |
| `docs/progress/TASKS.md` | `T-027` DONE; `T-028` TODO baru (kontrak endpoint re-submit); `T-017` dipersempit ke 9 temuan sisa | Perbaikan harus punya task; utang yang ditemukan tidak boleh hilang | P-014 |
| `docs/progress/TRACEABILITY.md` | Baris `FR-WF-09` diperbarui (Desain = ADR-0016) | Requirement yang keputusannya baru ditetapkan harus terlacak | P-014 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-010: hasil P-014; C-022 keluar dari daftar OPEN; rekomendasi berikutnya C-018/C-016 | Keputusan tercatat di tempatnya | P-014 |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CONTINUE.md`, `AGENTS.md` | Ledger dan snapshot §0 sesi P-014 | Protokol progress | P-014 |

---

## 2026-09-18 (sesi P-013)

Perbaikan temuan **C-005**: optimistic locking transisi workflow instance ditetapkan lewat ADR-0015, bukan diserahkan ke masing-masing agen. Dua kolom yang dipakai dokumen tetapi tidak ada di skema ditambahkan sekaligus (`version`, `current_step_deadline`), dan pola guard-nya ditulis di satu tempat yang ditunjuk semua dokumen. Dua temuan baru yang muncul saat mengerjakan (`C-021`, `C-023`) ikut ditutup; satu temuan butuh keputusan user (`C-022`).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0015-optimistic-locking-workflow-instance.md` | Keputusan arsitektur: guard `version` di database, tanpa retry otomatis, plus `current_step_deadline`. Diwajibkan karena temuan ber-ADR dan polanya menyentuh seluruh modul workflow | P-013 |
| `docs/progress/prompts/P-013-2026-09-18-optimistic-locking-workflow-instance.md` | Log prompt sesi ini | P-013 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/41-DATABASE.md` | §2.4: `workflow_instances.version` + `current_step_deadline` + indeks parsial `(current_step_deadline) WHERE status = 'running'` + blok "Aturan transisi (ADR-0015)"; §3 baris indeks; §4 catatan isi migrasi `005` | Dua kolom dipakai dokumen tetapi tidak ada di skema; guard harus terlihat dari skema, bukan hanya dari dokumen workflow | P-013 |
| `docs/design/43-WORKFLOW.md` | §4.1: `current_step_deadline` + `version = 0` saat submit; §4.2: urutan transaksi eksplisit (guard gagal → rollback); catatan setelah §4.5 bahwa `handle*` hanya menghitung state; §6 ditulis ulang (SQL 4 kondisi + 6 aturan); §7 sumber deadline | Solusi lama memakai `WHERE id AND version` tanpa status/step dan tanpa aturan rollback — tidak dapat dijalankan apa adanya | P-013 |
| `docs/design/42-API.md` | §5: `version` + `current_step_deadline` pada response, `version` opsional pada request aksi, izin `workflow_instance:submit`/`:read`, kontrak `409 WORKFLOW_CONFLICT` dengan `details`; §12: penjelasan beda `CONFLICT` vs `WORKFLOW_CONFLICT` | Perilaku klien saat konflik tidak boleh ditebak; matriks memisahkan approve/reject/request_revision | P-013 |
| `docs/design/40-TSD.md` | §2.3: `WorkflowInstance.Version` + `CurrentStepDeadline`; §2.4: kontrak `ExecuteAction` (satu transaksi, error khusus saat rowsAffected = 0); §6: empat aturan pemetaan izin + route aksi memakai `workflow_instance:read` | Model harus memuat kolom; route tidak boleh memasang izin aksi tunggal untuk endpoint yang aksinya ada di body (C-023) | P-013 |
| `docs/design/70-TESTING.md` | §3.3 test konkurensi baru (dua goroutine, tepat satu menang, rollback terbukti); §5.1 test E2E konflik dua konteks; §8.1 tabel env test (`E2E_*`, `TEST_DATABASE_URL`) | Janji "approve ganda dicegah" harus punya test yang membuktikannya, termasuk bukti transaksi batal | P-013 |
| `docs/adr/README.md` | Index: baris ADR-0015 | Setiap ADR wajib muncul di index | P-013 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-005 → `FIXED`; temuan baru C-021 (FIXED), C-022 (OPEN), C-023 (FIXED); ringkasan 23 temuan / 13 FIXED; catatan temuan tambahan P-013; §6 daftar temuan ber-ADR diperbarui | Temuan di berkas dan transisi yang sama tidak boleh ditinggalkan separuh; nomor temuan naik, tidak dipakai ulang | P-013 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 23 temuan (11 S1, 9 S2, 3 S3), 13 FIXED, 10 OPEN + catatan asal C-021..C-023 | Audit harus mencerminkan cakupan sebenarnya | P-013 |
| `docs/progress/TASKS.md` | `T-026` DONE; `T-017` dipersempit ke 9 temuan yang masih OPEN (C-005 keluar dari daftar) | Perbaikan harus punya task dan bukti | P-013 |
| `docs/progress/TRACEABILITY.md` | Baris `FR-WF-06`, `FR-WF-07`, `FR-WF-08` | Requirement yang kontraknya baru ditetapkan harus terlacak | P-013 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-010: hasil P-013; **C-022 (perilaku request revision)** ditambahkan sebagai pilihan berikutnya | Keputusan lanjutan | P-013 |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CONTINUE.md` | Ledger dan snapshot §0 sesi P-013 | Protokol progress | P-013 |

---

## 2026-09-18 (sesi P-012)

Perbaikan temuan **C-012**: enam requirement yang belum punya kontrak endpoint dilengkapi di `42-API.md` (satu High FR-ROLE-04, lima Medium), masing-masing dengan izin dari matriks ADR-0014. Satu endpoint dipersempit dan satu bab baru ditambahkan (Reports), sehingga nomor bab 42-API bergeser dan rujukannya diperbarui.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-012-2026-09-18-kontrak-endpoint-requirement-tanpa-endpoint.md` | Log prompt sesi ini | P-012 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/42-API.md` | §2: `POST /auth/change-password` (FR-AUTH-09, + kode `INVALID_CURRENT_PASSWORD`, pencabutan token lain); §9: filter `GET /audit` dilengkapi + izin `audit:read` (FR-AUDIT-04); **§10 baru: Reports** — `GET /reports/export` (FR-REP-01, izin `report:export`); §11: `POST /admin/users/:id/reset-password` (FR-AUTH-08), `PUT /admin/users/:id/roles` (FR-ROLE-04, izin `user_role:manage`), `POST` + `PATCH /admin/organizations` (FR-ORG-03), dan `PATCH /admin/users/:id` dipersempit ke status/profil; Error Responses -> §12, Swagger -> §13 | Enam requirement tidak punya kontrak meski halamannya sudah ada di FSD; `PATCH /admin/users/:id` mencampur perubahan permission dengan data profil sehingga tidak dapat diaudit sebagai aksi tersendiri | P-012 |
| `docs/design/50-FSD.md` | §2.2, §10.1, §10.3: setiap aksi sekarang menyebut endpoint-nya (`42-API.md`) | FSD dan kontrak API harus saling menunjuk; sebelumnya tidak ada halaman pun yang tertaut ke endpoint | P-012 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | DoD: rujukan format error `42-API.md` §11 -> §12 | Nomor bab bergeser karena penyisipan §10 Reports | P-012 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-012 -> `FIXED`; baris C-018 diberi catatan perluasan cakupan (halaman Reports dan Administration > Workflows juga belum ada di FSD); ringkasan 10 FIXED / 10 OPEN | Audit harus mencerminkan tindak lanjut; gap baru tidak diberi ID karangan | P-012 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 10 dari 20 FIXED | Sama | P-012 |
| `docs/progress/TASKS.md` | `T-025` (C-012) DONE; T-024 dirapikan (`§2-§11`); T-017 dipersempit ke 10 temuan | Perbaikan harus punya task dan bukti | P-012 |
| `docs/progress/TRACEABILITY.md` | Baris `FR-AUTH-08`, `FR-AUTH-09`, `FR-ORG-03`, `FR-REP-01`, `FR-AUDIT-04`; kolom Desain `FR-ROLE-04` dilengkapi | Requirement tanpa baris traceability tidak dapat diaudit | P-012 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-010: sisa 10 temuan; C-005 disarankan berikutnya | Keputusan lanjutan | P-012 |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CONTINUE.md` | Ledger dan snapshot §0 sesi P-012 | Protokol progress | P-012 |

---

## 2026-09-18 (sesi P-011)

Perbaikan temuan **C-011** (dua endpoint untuk definisi workflow) dan **C-013** (endpoint yang diregistrasi tetapi tidak ada di spesifikasi API). Akarnya dibereskan: **`42-API.md` ditetapkan sebagai sumber tunggal daftar endpoint**, dan `40-TSD.md` §6 dipersempit menjadi contoh pemasangan route. Dua belas fence markdown liar di empat dokumen desain juga dibersihkan karena membuat bab endpoint dirender sebagai blok kode.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-011-2026-09-18-rekonsiliasi-daftar-endpoint.md` | Log prompt sesi ini | P-011 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/42-API.md` | Header: blok "Sumber tunggal endpoint" + aturan "bila berbeda, dokumen ini yang berlaku"; §5: `GET /workflows/definitions/:id` dan `POST /workflows/definitions/:id/steps` ditambahkan lengkap dengan contoh request dan izinnya; §10: `POST /admin/workflow-definitions` dihapus + penjelasan jalur tunggal; 9 fence markdown liar dihapus | **Sumber tunggal endpoint**; C-011 dan C-013 | P-011 |
| `docs/design/40-TSD.md` | §6: daftar ~45 route diganti contoh wiring (auth, projects, documents, workflows, admin) + 3 aturan (daftar lengkap di `42-API.md`, setiap route wajib `RequirePermission`, route tanpa middleware izin hanya butuh autentikasi) + catatan bahwa definisi workflow hanya di `/workflows/definitions` | Dua daftar hampir lengkap terus berbeda (C-013); yang kedua bahkan tidak memuat `PATCH /admin/settings/:key` | P-011 |
| `docs/design/44-SECURITY.md` | §3.3: penunjuk "pemetaan route ada di `40-TSD` §5.3/§6" diperbaiki menjadi "endpoint di `42-API.md`, pemasangan izin di `40-TSD` §5.3/§6" | Penunjuk lama menunjuk daftar yang kini sengaja tidak lengkap | P-011 |
| `docs/design/43-WORKFLOW.md`, `docs/design/60-DEPLOYMENT.md`, `docs/design/70-TESTING.md` | Masing-masing satu fence markdown liar di akhir berkas dihapus | Fence tidak seimbang menelan bab berikutnya sebagai blok kode | P-011 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-011 dan C-013 → `FIXED`; baris C-009..C-013 dipecah; ringkasan 9 FIXED / 11 OPEN | Audit mencerminkan tindak lanjut nyata | P-011 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 9 dari 20 FIXED | Sama | P-011 |
| `docs/progress/TASKS.md` | `T-023` (C-011 + C-013) DONE; `T-024` TODO baru (anotasi `Izin:` per endpoint di `42-API.md`); `T-017` dipersempit ke 11 temuan | Perbaikan harus punya task; pekerjaan yang ditunda harus tercatat | P-011 |
| `docs/progress/TRACEABILITY.md` | Baris `FR-WF-01` dan `FR-WF-02` | Dua endpoint yang ditambahkan menyentuh requirement ini | P-011 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-010: sisa 11 temuan; C-012 disarankan berikutnya | Keputusan lanjutan | P-011 |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CONTINUE.md` | Ledger dan snapshot §0 sesi P-011 | Protokol progress | P-011 |

---

## 2026-09-18 (sesi P-010)

Perbaikan temuan **C-014**: ringkasan environment variable yang usang di `90-AGENT-GUIDE.md` §7 dihapus dan diganti penunjuk ke sumber tunggalnya. Dua duplikasi konfigurasi runtime sekelas ikut dibersihkan karena keduanya salinan dari kontrak yang sama (blok YAML compose di `30-ARCHITECTURE.md` §5.1 dengan port host salah, dan kredensial `admin123` pada test E2E). Tidak ada ADR baru — ini konsolidasi dokumen, bukan keputusan arsitektur.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/prompts/P-010-2026-09-18-sumber-tunggal-konfigurasi-runtime.md` | Log prompt sesi ini | P-010 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/90-AGENT-GUIDE.md` | §7: blok 6 variabel (termasuk `ADMIN_PASSWORD=admin123`) dihapus, diganti tabel penunjuk ke `60-DEPLOYMENT.md` §2.1 + `.env.example`, perintah `cp .env.example .env`, dan 2 aturan mengikat (jangan menyalin daftar; nilai contoh terlarang) | Ringkasan itu tidak lengkap (6 dari 20 variabel) dan memuat nilai yang justru membuat aplikasi gagal start menurut ADR-0010 | P-010 |
| `docs/design/30-ARCHITECTURE.md` | §5.1: blok YAML compose (±30 baris, memuat `ports: ["8080:8080"]`) diganti penunjuk ke `docker-compose.yml` + `60-DEPLOYMENT.md` §2, dengan perintah validasi | Port host salah (`8081` di mesin development ini) dan menjadi sumber drift ketiga untuk konfigurasi yang sama | P-010 |
| `docs/design/70-TESTING.md` | §5.1: kredensial login E2E dibaca dari environment (`E2E_ADMIN_USERNAME`/`E2E_ADMIN_PASSWORD`), bukan `admin`/`admin123` | Test yang memakai nilai terlarang ADR-0010 tidak akan pernah lolos terhadap deployment nyata | P-010 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §7: butir "90-AGENT-GUIDE §7 hanyalah ringkasan" diganti menunjuk ke keadaan baru (keduanya hanya menunjuk) + larangan salinan daftar/YAML | Aturan lama mengandaikan salinan masih ada | P-010 |
| `docs/progress/OPEN-QUESTIONS.md` | §2 item 4 (daftar env tersebar) → RESOLVED | Sumber tunggal kini benar-benar tunggal | P-010 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Baris C-009..C-016 dipecah; C-014 → FIXED; ringkasan 7 FIXED / 13 OPEN | Audit mencerminkan tindak lanjut nyata | P-010 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 7 dari 20 FIXED | Sama | P-010 |
| `docs/progress/TASKS.md` | `T-022` (perbaikan C-014) DONE; `T-017` dipersempit ke 13 temuan | Perbaikan harus punya task dan bukti | P-010 |
| `docs/progress/TRACEABILITY.md` | Baris `NFR-PORT-01` ditambahkan dengan artefak `docker-compose.yml`/`.env.example` | Konfigurasi runtime kini punya bukti yang dapat dijalankan | P-010 |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CONTINUE.md` | Ledger dan snapshot §0 sesi P-010 | Protokol progress | P-010 |

---

## 2026-09-18 (sesi P-009)

Perbaikan temuan audit **C-017** (matriks permission tidak ada sehingga isi migrasi `008_seed_default_roles.sql` harus dikarang) sekaligus **C-008** (hak akses audit log) yang tidak dapat dibiarkan ambigu saat matriks ditulis. Keputusan dicatat sebagai ADR-0014.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0014-matriks-permission-rbac.md` | Matriks permission = sumber tunggal migrasi `008`; kosakata tertutup 17 resource × 15 action; jumlah baris 44/30/18/12 | P-009 |
| `docs/progress/prompts/P-009-2026-09-18-matriks-permission-rbac.md` | Log prompt sesi ini | P-009 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/44-SECURITY.md` | §3.1 menjadi sumber tunggal: model + §3.1.1 kosakata tertutup + §3.1.2 matriks 44 baris × 4 role + §3.1.3 aturan scoping; §3.3 hierarki dipisah menjadi role sistem dan role project + catatan C-007 | Matriks lama hanya ada di TSD dengan nama aksi yang tidak dapat dipetakan ke `role_permissions` | P-009 |
| `docs/design/40-TSD.md` | §5.3 diganti penunjuk + contoh `RequirePermission(resource, action)`; §2.2/§6 memakai `RequirePermission`, izin admin dicek per route | Menghapus matriks kedua dan bentuk `RBACMiddleware("admin")` yang tidak dapat dipetakan ke tabel | P-009 |
| `docs/design/41-DATABASE.md` | §2.1 komentar `resource`/`action` dan `roles.name` diarahkan ke `44-SECURITY` §3.1; §4 memuat prosedur isi migrasi `008` (4 role, 104 baris `role_permissions`, idempotent, kerangka SQL, kueri verifikasi); §4.1 langkah 3 bootstrap menetapkan role `administrator` | Isi migrasi `008` sebelumnya tidak ada di dokumen; admin pertama sebelumnya tanpa role | P-009 |
| `docs/design/70-TESTING.md` | §4.1: test matriks memakai pasangan (resource, action) dari basis data, tambah test jumlah baris seed dan test scoping; typo `Test RBAC_Permisisons` diperbaiki | Test lama memakai `"create_project"` yang tidak ada di kosakata, dan tidak menguji isi seed | P-009 |
| `docs/design/51-UX.md` | §2.1: baris Reports dipisah, Audit = Admin; catatan bahwa "Reviewer+" menunggu C-006 | Menutup C-008 tanpa meninggalkan label menu yang bertentangan dengan matriks | P-009 |
| `docs/design/50-FSD.md` | §10.2 "View permission matrix" diarahkan ke sumber datanya | Role Management membaca `role_permissions`, bukan menyalin matriks | P-009 |
| `docs/design/20-SRS.md` | FR-ROLE-03 menunjuk sumber matriks dan migrasi `008` | Requirement harus dapat diverifikasi | P-009 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §3 langkah 4: seed `008` dari `44-SECURITY` §3.1, kriteria selesai memuat 104 baris `role_permissions` | Bootstrap Phase 0 punya target terukur | P-009 |
| `AGENTS.md` | Baris routing "Role & permission (RBAC)" | Menemukan matriks tanpa menebak dokumen | P-009 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | C-017 dan C-008 → `FIXED` | Audit mencerminkan tindak lanjut nyata | P-009 |
| `docs/progress/audits/README.md` | Status AUDIT-001: 6 dari 20 FIXED | Sama | P-009 |
| `docs/adr/README.md` | Index ADR-0014 | Setiap ADR wajib terdaftar | P-009 |
| `docs/progress/TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `STATE.md`, `SESSION-LOG.md`, `CONTINUE.md` | Ledger sesi P-009: `T-021` DONE, baris FR-ROLE-01..04, Q-010 diperbarui, snapshot §0 | Protokol progress | P-009 |

---

## 2026-09-18 (sesi P-008)

Perbaikan tiga temuan S1 dari `AUDIT-001`: C-001 (lapisan audit log), C-002 (struktur folder backend), C-003 (status kanonik vs label & overdue turunan). Setiap keputusan dicatat sebagai ADR `ACCEPTED`, bukan diselipkan diam-diam ke dokumen. C-019 ikut tertutup sebagai efek samping C-003.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0011-audit-log-layer.md` | Mengunci C-001: audit ditulis di service, di dalam transaksi yang sama; handler tidak pernah memanggil audit | P-008 |
| `docs/adr/0012-status-kanonik-dan-overdue-turunan.md` | Mengunci C-003: nilai kanonik vs label, dan "overdue" sebagai turunan yang tidak pernah disimpan | P-008 |
| `docs/adr/0013-struktur-paket-backend.md` | Mengunci C-002: satu folder = satu package, tanpa subfolder per modul; `40-TSD.md` §2.0 sebagai sumber tunggal | P-008 |
| `docs/progress/prompts/P-008-2026-09-18-perbaikan-audit-c001-c003.md` | Log prompt sesi ini | P-008 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/40-TSD.md` | §2.0 baru: pohon struktur folder kanonik + 5 aturan; §2.4: 5 aturan transaksi & audit; §2.6: contoh handler tanpa `AuditService`/`audit.Log` | Menjadi sumber tunggal struktur (C-002) dan lapisan audit (C-001) | P-008 |
| `docs/design/30-ARCHITECTURE.md` | §3.2: pohon struktur (versi subfolder per modul) diganti ringkasan + tautan; §3.3: penunjuk ADR-0011 | Menghapus versi struktur kedua | P-008 |
| `docs/design/01-AGENT-WORKFRAME.md` | §6: pohon backend diselaraskan (ada `bootstrap` dan `pkg`) + tautan ke `40-TSD.md` §2.0 | Menghapus versi struktur ketiga | P-008 |
| `docs/design/90-AGENT-GUIDE.md` | §2.1: pohon backend diganti ringkasan + tautan; §3.1: contoh service memakai `package service`, `withTx`, dan `audit.Log(ctx, tx, ...)` | Menghapus versi struktur keempat dan contoh audit yang tidak bertransaksi | P-008 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §3 langkah 1 diarahkan ke `40-TSD.md` §2.0; §5 butir DoD audit diperjelas (di service, satu transaksi) | DoD adalah gerbang `DONE` | P-008 |
| `docs/design/70-TESTING.md` | §2.1/§3.1: path berkas test disesuaikan struktur flat; 3 test baru (audit satu transaksi, `is_overdue` turunan, `'overdue'` ditolak constraint) | Mitigasi yang dijanjikan ADR-0011 & ADR-0012 | P-008 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | §9: nama berkas contoh `auth.go` → `auth_handler.go`/`auth_service.go` | Konsistensi dengan struktur flat | P-008 |
| `docs/design/50-FSD.md` | §11 baru: tabel status kanonik vs label (documents, tasks, project/instance) + tabel nilai turunan; §6.1: "Overdue" dinyatakan turunan | Satu sumber pemetaan status (C-003) | P-008 |
| `docs/design/51-UX.md` | §2.1: sub-menu "Pending"/"Revision" → "Pending Review"/"Revision Required" + catatan ADR-0012 | Menu tidak lagi berbeda dari label status (menutup C-019) | P-008 |
| `docs/design/43-WORKFLOW.md` | §7: catatan bahwa keterlambatan step adalah turunan; cron hanya mengirim notifikasi | Mencegah `'overdue'` masuk kolom status | P-008 |
| `docs/design/20-SRS.md` | FR-TASK-06: klausa "penanda turunan, bukan nilai status baru" | Requirement tidak lagi bisa ditafsirkan menambah nilai status | P-008 |
| `docs/design/41-DATABASE.md` | §2.3/§2.5: komentar kanonik pada `documents.status` dan `tasks.status` + rujukan `50-FSD.md` §11 | DDL adalah sumber nilai; komentar mencegah nilai turunan ditambahkan | P-008 |
| `docs/design/80-ROADMAP.md` | Phase 3: "Overdue detection" dinyatakan turunan, cron hanya notifikasi; exit criteria disesuaikan | Menghindari job yang mengubah status | P-008 |
| `IDEA.md` | Bagian 2.G: catatan ADR-0012 bahwa "Task = Overdue" adalah penanda turunan | Dokumen sumber tetap akurat tanpa mengubah isi kebutuhan | P-008 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Tabel status tindak lanjut diisi: C-001/C-002/C-003/C-019 `FIXED`, sisanya `OPEN` | Audit harus mencerminkan tindak lanjut nyata | P-008 |
| `docs/progress/audits/README.md` | Status AUDIT-001: `3+1 dari 20 FIXED`, sisanya OPEN | Sama | P-008 |
| `docs/progress/TASKS.md` | `T-018`/`T-019`/`T-020` (C-001/C-002/C-003) DONE; `T-017` dipersempit ke 17 temuan sisanya | Perbaikan harus punya task dan bukti | P-008 |
| `docs/progress/TRACEABILITY.md` | Baris FR-AUDIT-01/03, FR-TASK-03/06 ditambahkan dengan dokumen desain baru | Tiga temuan menyentuh requirement | P-008 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-010 RESOLVED: user memilih C-001..C-003 lebih dulu | Keputusan sudah diambil | P-008 |
| `docs/progress/STATE.md` | Audit terbuka: 17 temuan; posisi, next action, daftar file terakhir | Kondisi setelah perubahan | P-008 |
| `docs/progress/SESSION-LOG.md` | Entri P-008 | Riwayat sesi | P-008 |
| `CONTINUE.md` | §0 snapshot: prompt terakhir P-008, audit 17 temuan OPEN; §2.3a diperbarui | Titik masuk resume harus benar | P-008 |
| `docs/adr/README.md` | Index ADR 0011-0013 | Setiap ADR wajib terdaftar | P-008 |

---

## 2026-09-17 (sesi P-007)

Audit kontradiksi dokumen. **Tidak ada dokumen desain yang diubah pada sesi ini**, sesuai instruksi agar laporan datang lebih dulu.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | 20 temuan kontradiksi (10 S1, 7 S2, 3 S3) dengan bukti `file:line`, dampak, dan usul resolusi | P-007 |
| `docs/progress/audits/README.md` | Aturan direktori audit: ID temuan tetap, status tindak lanjut, larangan mengubah dokumen lain dari dalam audit | P-007 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/progress/TASKS.md` | `T-016` (audit) DONE; `T-017` (perbaikan temuan) baru | Perbaikan harus dipilih dulu | P-007 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-010 (temuan mana yang diperbaiki) ditambahkan; Q-004 ditandai BLOCKING dengan cakupan `.gitignore` | Keputusan user diperlukan sebelum memperbaiki | P-007 |

---

## 2026-09-17 (sesi P-006)

Berkas konfigurasi runtime yang dapat dieksekusi dan tervalidasi, plus penghapusan duplikasi YAML di dokumen deployment.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `.env.example` | Cermin daftar environment variable `60-DEPLOYMENT.md` §2.1; mencatat tiga mode `DB_HOST` | P-006 |
| `docker-compose.yml` | Compose referensi (app + postgres 16-alpine, volume, health check); port host 8081/5433 agar tidak menabrak `wms-backend` dan PostgreSQL host | P-006 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/design/60-DEPLOYMENT.md` | §2: blok YAML 60 baris diganti tabel kontrak + cara validasi; §2.1: diganti tabel variabel lengkap (17 variabel); §4.1: "Tidak Ada `init.sql`" beserta alasan; §4.2: daftar migrasi dihapus, diarahkan ke `41-DATABASE.md` §4 | Duplikasi YAML menjadi sumber drift, dan ditemukan **daftar migrasi kedua** yang berbeda dari `41-DATABASE.md` §4 | P-006 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Endpoint health diseragamkan menjadi `GET /health`; checklist §3.1 item 2 & 6 diperbarui | `60-DEPLOYMENT.md` §5 memakai `/health`, sedangkan dokumen lain menulis `/healthz` | P-006 |
| `docs/design/01-AGENT-WORKFRAME.md` | Pohon struktur §6 memuat `.env.example` dan `docker-compose.yml`; gap item 5 selesai | Struktur proyek harus mencerminkan berkas nyata | P-006 |
| `docs/progress/TASKS.md` | `T-015` DONE, `T-002a` baru (`.gitignore` wajib memuat `.env`), `T-003` memakai `/health` | `.env` belum terlindungi karena `.gitignore` belum ada | P-006 |

---

## 2026-09-17 (sesi P-005)

Pemeriksaan PostgreSQL 16 dan rencana toolchain backend. **Koreksi penting:** yang berjalan di port 5432 adalah PostgreSQL 16.10 (Postgres.app), bukan Homebrew 14.6; tidak perlu memasang PostgreSQL.

### Fixed

| File | Perbaikan | Alasan | Prompt |
|---|---|---|---|
| `docs/progress/STATE.md` | Tabel environment & port: server 16.10 terpisah dari client 14.6; port 5432/5433; Docker tidak diperlukan | `psql --version` menampilkan versi *client*, bukan *server*, sehingga kesimpulan P-004 keliru | P-005 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §2: PATH tiga direktori termasuk biner Postgres.app 16; §7.1: PostgreSQL tetap di 5432; §3.1 item 2 & 6a | Menghindari remapping port yang tidak perlu | P-005 |
| `docs/progress/TASKS.md` | `T-011` mencakup biner Postgres.app; `T-013` diubah menjadi "buat role + database `bwdcs`" (tanpa instalasi); `T-014` jadi opsional | Rencana harus mencerminkan kondisi nyata | P-005 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-009 ditulis ulang: sisa kebutuhan hanya PATH, `goose`, dan pembuatan database | Sebelumnya meminta instalasi PostgreSQL yang tidak dibutuhkan | P-005 |
| `docs/progress/prompts/P-004-...md`, `docs/progress/SESSION-LOG.md` | Catatan koreksi ditambahkan (riwayat lama tidak dihapus) | Aturan ledger append-only | P-005 |

---

## 2026-09-17 (sesi P-004)

Keputusan auth (Q-006, Q-007) dengan opsi paling aman, penyelarasan dokumen desain, dan verifikasi tooling (T-001).

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0010-first-run-bootstrap.md` | Bootstrap organisasi & admin pertama: hanya bila tabel `users` kosong, satu transaksi, `ADMIN_PASSWORD` minimal 12 karakter dan bukan nilai contoh | P-004 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/adr/0009-logout-token-invalidation.md` | `PROPOSED` -> `ACCEPTED`; keputusan: daftar revokasi `jti` di PostgreSQL, cache 30 detik, revokasi menyeluruh saat password berubah/reset & akun dinonaktifkan | Pilihan paling aman tanpa menambah dependensi runtime (Redis ditolak) | P-004 |
| `docs/adr/README.md` | Index: 0009 `ACCEPTED`, 0010 ditambahkan | Index harus lengkap | P-004 |
| `docs/design/41-DATABASE.md` | DDL `token_revocations` + indeks, migrasi `009_create_token_revocations.sql`, §4.1 urutan startup (migrasi -> bootstrap -> server) | Kontrak tabel & urutan startup sebelum kode | P-004 |
| `docs/design/44-SECURITY.md` | §2.2: klaim `jti`, tabel peristiwa revokasi, catatan jendela cache 30 detik; §2.3 session management | Perilaku logout harus eksplisit, bukan tafsiran | P-004 |
| `docs/design/42-API.md` | §2: semantik `POST /auth/logout` (idempotent, `logout_all`, 401 untuk token dicabut) dan catatan refresh | Handler harus punya kontrak tetap | P-004 |
| `docs/design/40-TSD.md` | `JWTClaims` memuat `JTI`; `AuthService.Logout(token, logoutAll)`, `Refresh`, `RevokeAllForUser`; §5.2.1 `RevocationStore`; §2.7 `internal/bootstrap` | Bentuk kode sebelum menulis kode | P-004 |
| `docs/design/60-DEPLOYMENT.md` | Env `ADMIN_ORG_NAME`/`ADMIN_ORG_CODE` + catatan hanya dipakai saat tabel kosong; §4.2 urutan startup | Env yang dibutuhkan bootstrap belum terdaftar | P-004 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §2 catatan PATH (Go ada di `/usr/local/go/bin`); §3 bootstrap termasuk migrasi 009; §3.1 checklist item 4/5/9 selesai; §7.1 port nyata 8081 & 5433 | Temuan mesin harus tercatat supaya agen berikutnya tidak salah simpulkan | P-004 |
| `docs/design/01-AGENT-WORKFRAME.md` | Index ADR-0008/0009/0010; gap item 6 & temuan baru | Konsistensi | P-004 |
| `docs/progress/STATE.md` | Tabel environment hasil T-001 + tabel port yang terpakai | Ledger mencerminkan kondisi nyata | P-004 |
| `docs/progress/TASKS.md` | `T-001` DONE dengan bukti; `T-011`-`T-014` baru; `T-004`/`T-005` keluar dari BLOCKED | Papan kerja akurat | P-004 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-006 & Q-007 RESOLVED dengan bukti; Q-009 baru (izin tooling) | Keputusan tidak boleh hilang | P-004 |

---

## 2026-09-17 (sesi P-003)

Pre-flight sebelum Phase 0: pemeriksaan kesiapan dokumen, penguncian pilihan library, dan pencatatan keputusan yang belum ada.

### Added

| File | Alasan | Prompt |
|---|---|---|
| `docs/adr/0008-backend-library-lockin.md` | `01-AGENT-WORKFRAME` §2.3 dan `30-ARCHITECTURE` §6 menulis "Chi/Gin" dan "SQLX/pgx" (pilihan ganda = belum diputuskan bagi agen baru), sedangkan `40-TSD` §1 sudah spesifik | P-003 |
| `docs/adr/0009-logout-token-invalidation.md` | `44-SECURITY` §2 mewajibkan invalidasi token saat logout, mekanismenya belum ada, Redis opsional (status PROPOSED) | P-003 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `docs/adr/README.md` | Index ditambah ADR-0008 dan ADR-0009 | Index harus lengkap | P-003 |
| `docs/design/01-AGENT-WORKFRAME.md` | §2.3 backend dikunci ke `gin` + `pgx/v5` (lihat ADR-0008) | Menghapus ambiguitas yang menghalangi penulisan `go.mod` | P-003 |
| `docs/design/30-ARCHITECTURE.md` | §6 baris Web Framework & Backend framework dikunci ke Gin | Konsisten dengan ADR-0008 | P-003 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §3.1 pre-flight checklist (9 item) + §7.1 port dev (8080/5432/5173) dan proxy Vite | Prasyarat kerja dan port harus ditentukan sebelum server dijalankan | P-003 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-006 (invalidasi token), Q-007 (bootstrap org & admin), Q-008 (whitelist dokumen kantor) + temuan inkonsistensi #6-#9 | Keputusan yang belum ada tidak boleh ditebak | P-003 |
| `docs/progress/TASKS.md` | `T-004` dan `T-005` ditandai BLOCKED dengan blocker spesifik | Papan harus mencerminkan kenyataan | P-003 |

---

## 2026-09-17 (sesi P-002)

### Added

| File | Alasan | Prompt |
|---|---|---|
| `CONTINUE.md` | Titik masuk resume lintas agen/model: urutan baca dokumen, rekonstruksi posisi, urutan pekerjaan, checklist penutup, blok snapshot §0 | P-002 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `AGENTS.md` | Peringatan + baris tabel `CONTINUE.md`, langkah resume jadi langkah 1, kewajiban update snapshot, routing | Titik masuk harus terlihat sebelum agen memilih modul | P-002 |
| `docs/design/01-AGENT-WORKFRAME.md` | `CONTINUE.md` di tabel sumber §2.1, langkah §4.1, pohon struktur §6, langkah HANDOFF | Konsistensi alur kerja | P-002 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | `CONTINUE.md` sebagai artefak resmi + langkah RESUME/CATAT + checklist §8 | Snapshot menjadi kewajiban penutup sesi | P-002 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §9 Resume & Handoff mengarah ke `CONTINUE.md` | Satu titik masuk untuk semua jalur resume | P-002 |
| `docs/design/90-AGENT-GUIDE.md` | Urutan baca resume dimulai dari `CONTINUE.md` | Agen baru tidak mulai dari menebak | P-002 |
| `docs/design/00-README.md` | Indeks dokumen luar `docs/design` ditambah `CONTINUE.md` + jalur baca | Navigasi | P-002 |
| `docs/progress/README.md` | Inventaris + bagian resume | Ledger dan titik masuk saling merujuk | P-002 |
| `docs/progress/TASKS.md` | `T-008` (handoff lintas agen + jaga snapshot) | Snapshot bisa mati tanpa task pengikat | P-002 |

---

## 2026-09-17 (sesi P-001)

### Added

| File | Alasan | Prompt |
|---|---|---|
| `DESIGN.md` | Placeholder jujur arah desain. Belum diisi, sehingga semua UI berstatus "draft without direction" (antislop R-37) | P-001 |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Protokol wajib pencatatan progress untuk setiap prompt & perubahan file | P-001 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Bootstrap repo kosong, konvensi git, definition of done, traceability requirement | P-001 |
| `docs/progress/README.md` | Aturan & inventaris ledger progress | P-001 |
| `docs/progress/STATE.md` | Snapshot kondisi proyek untuk resume agen | P-001 |
| `docs/progress/SESSION-LOG.md` | Riwayat sesi/prompt (append-only) | P-001 |
| `docs/progress/CHANGELOG.md` | File ini | P-001 |
| `docs/progress/TASKS.md` | Backlog & papan task dengan ID `T-###` | P-001 |
| `docs/progress/TRACEABILITY.md` | Matriks requirement → dokumen → file → test | P-001 |
| `docs/progress/OPEN-QUESTIONS.md` | Pertanyaan pending yang butuh keputusan user | P-001 |
| `docs/progress/prompts/TEMPLATE.md` | Template log per prompt | P-001 |
| `docs/progress/prompts/P-001-2026-09-17-agent-documentation-foundation.md` | Log prompt pertama | P-001 |
| `docs/adr/README.md` | Index & aturan ADR, sumber tunggal keputusan arsitektur | P-001 |
| `docs/adr/0001-backend-go-sqlx.md` .. `docs/adr/0005-local-file-storage-first.md` | Migrasi keputusan yang sebelumnya tersebar di dua decision log | P-001 |
| `docs/adr/0006-antislop-usage-mode.md`, `docs/adr/0007-design-direction-source.md` | Keputusan pending yang butuh input user (status `PROPOSED`) | P-001 |
| `scripts/check-doc-links.sh` | Pemeriksa referensi file di seluruh `*.md`. `BROKEN` = kesalahan (exit 1), `PLANNED` = file kode/aset belum dibuat, referensi placeholder dilewati | P-001 |

### Changed

| File | Perubahan | Alasan | Prompt |
|---|---|---|---|
| `AGENTS.md` | Ditambah section "Protokol Progress (WAJIB)" dan routing dokumen baru (progress, ADR, development workflow) | Kewajiban update dokumen harus terlihat di entry file yang dibaca agen | P-001 |
| `docs/design/00-README.md` | Index dokumen ditambah 02, 12, ADR, `DESIGN.md`, `docs/progress/` | Navigasi tidak lagi menyesatkan | P-001 |
| `docs/design/01-AGENT-WORKFRAME.md` | §4 ditambah langkah 7 (update progress), §6 struktur proyek diperbarui, §7 decision log mengarah ke ADR, §8 diganti menjadi tabel gap & keputusan pending yang aktual | Menyatukan aturan dan menghapus daftar yang sudah usang | P-001 |
| `docs/design/90-AGENT-GUIDE.md` | §1 ditambah jalur resume sesi, §5 decision log mengarah ke ADR, §7 quick reference ditambah perintah verifikasi progress | Agen yang melanjutkan butuh titik masuk yang benar | P-001 |
| `docs/design/20-SRS.md` | §2.1 typo teks campuran diperbaiki; §2.4/§2.5 constraint diselaraskan dengan ADR-0004 (deployment fleksibel, Docker opsional) | SRS bertentangan dengan 01-AGENT-WORKFRAME dan 00-README | P-001 |
| `docs/design/80-ROADMAP.md` | §3 progress bar 100% diganti tabel status nyata + pointer ke `docs/progress/STATE.md` | Progress bar lama mengklaim MVP selesai padahal kode belum ada | P-001 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | §8 perintah link check diganti pemanggilan `scripts/check-doc-links.sh` | Perintah grep lama me-resolve path dari root sehingga menghasilkan false positive | P-001 |
| `docs/design/90-AGENT-GUIDE.md` | §7 quick reference memakai `scripts/check-doc-links.sh` + keterangan arti BROKEN/PLANNED | Konsisten dengan alur verifikasi resmi | P-001 |
| `docs/progress/OPEN-QUESTIONS.md` | Q-003 menulis `skills/<nama>/SKILL.md` (sebelumnya nama berkas tanpa path sehingga terdeteksi sebagai referensi rusak) | Agar pemeriksa referensi tidak bising oleh penulisan placeholder | P-001 |
