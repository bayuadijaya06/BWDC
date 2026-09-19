# 80-ROADMAP — Development Roadmap

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Development Phases

### Phase 0: Foundation (Week 1-2)

**Tujuan:** Setup infrastruktur dan core domain

| Task | Deliverable |
|---|---|
| Projekt structure setup | Go module, package layout |
| Database schema creation | Migration files 001-003 |
| Auth module implementation | Login, JWT, RBAC middleware |
| Config & logging setup | Viper config, slog integration |
| Basic Docker setup | docker-compose.yml |
| Admin seed script | Default users & roles |

**Exit Criteria:**
- [ ] User dapat login
- [ ] Admin dapat membuat user
- [ ] RBAC middleware berfungsi
- [ ] Docker compose up berjalan tanpa error

---

### Phase 1: Core Domain (Week 3-5)

**Tujuan:** Implementasi project, document, version

| Task | Deliverable |
|---|---|
| Project CRUD | Full project management |
| Document CRUD | Upload, metadata, list |
| Version control | Immutable versions, history |
| File storage abstraction | LocalFileSystem implementation |
| Document download | Stream handler |
| Basic search & filter | Title, status, category filter |

**Exit Criteria:**
- [ ] User dapat membuat project
- [ ] User dapat upload dokumen
- [ ] Versioning berfungsi (1.0 → 1.1 → 2.0)
- [ ] File tersimpan di storage
- [ ] Download bekerja

---

### Phase 2: Workflow Engine (Week 6-8)

**Tujuan:** Implementasi workflow approval

| Task | Deliverable |
|---|---|
| Workflow definition CRUD | Admin dapat buat workflow |
| Workflow instance creation | Submit triggers instance |
| Action handlers | Approve, Reject, Request Revision |
| Step progression logic | Auto-advance on approve |
| Revision rollback logic | Back to previous step |
| Notification on action | In-app notification |

**Exit Criteria:**
- [ ] Workflow 3-step berjalan lengkap
- [ ] Approve lanjut ke step berikutnya
- [ ] Reject → document rejected
- [ ] Request Revision → document revision_required
- [ ] Audit log tercatat

---

### Phase 3: Task & Comment (Week 9-10)

**Tujuan:** Task management dan commenting

| Task | Deliverable |
|---|---|
| Task CRUD | Create, update, complete |
| Task assignment | Assign to user |
| Overdue detection | Turunan `due_date` + status (ADR-0012); cron hanya mengirim notifikasi |
| Comment system | Threaded comments |
| Task-document relation | Link task to document |

**Exit Criteria:**
- [ ] Manager dapat create & assign task
- [ ] Task status transition works
- [ ] Comment pada document/task/project
- [ ] Overdue tasks terdeteksi sebagai turunan (tanpa menambah nilai status)

---

### Phase 4: UI Frontend (Week 11-14)

**Tujuan:** React SPA implementation

| Task | Deliverable |
|---|---|
| Layout & navigation | Sidebar, header, routing |
| Dashboard page | Stats, widgets, recent activity |
| Project pages | List, create, detail |
| Document pages | List, upload, detail, versions |
| Workflow pages | Definition management, approval panel |
| Task pages | List, create, detail |
| Admin pages | User, role, settings management |
| Notification center | Dropdown + page |

**Exit Criteria:**
- [ ] Semua page dapat diakses
- [ ] Form validasi berfungsi
- [ ] Real-time data (refresh after action)
- [ ] Responsive on desktop

---

### Phase 5: Polish & Deployment (Week 15-16)

**Tujuan:** Finalization

| Task | Deliverable |
|---|---|
| Audit log UI | Admin dapat view audit |
| Export CSV | Projects, documents, tasks |
| Search enhancement | Full-text search on document title |
| Error handling | Global error boundary |
| Performance optimization | Query optimization |
| Documentation | README, API docs |
| Security audit | Penetration testing |
| Docker production build | Multi-stage build |

**Exit Criteria:**
- [ ] MVP fitur lengkap
- [ ] Test coverage ≥ 70%
- [ ] Docker deployment verified
- [ ] Security scan clean

---

## 2. Future Enhancements (Post-MVP)

### Version 1.1

| Feature | Description |
|---|---|
| Email notifications | SMTP integration |
| PDF preview | Document preview tanpa download |
| Advanced search | Elasticsearch integration |
| Bulk operations | Bulk delete, bulk assign |
| Custom roles | Role customization |

### Version 1.2

| Feature | Description |
|---|---|
| S3 storage | MinIO/AWS S3 backend |
| WebSocket updates | Real-time notifications |
| API webhooks | External integration |
| Mobile responsive | Mobile-optimized UI |
| i18n | Multi-language support |

### Version 2.0 (Future Architecture)

| Feature | Description |
|---|---|
| Microservices split | Separate services per domain |
| Redis cache | Performance optimization |
| GraphQL API | Alternative to REST |
| AI assistance | Document summarization |
| E-signature | Digital signature integration |

---

## 3. Status Implementasi

Roadmap ini adalah rencana, bukan laporan progres. Angka progress lama (semua fase 100%) dihapus pada 2026-09-17 karena menyesatkan: saat itu belum ada kode aplikasi sama sekali.

**Sumber status nyata: `docs/progress/STATE.md`.** Perbarui tabel di bawah hanya bila sebuah fase benar-benar berubah status, dan catat buktinya di `docs/progress/CHANGELOG.md`.

| Fase | Cakupan | Status | Bukti / Catatan |
|---|---|---|---|
| Pra-fase | Dokumentasi: aturan kerja agen, protokol progress, ADR, alur pengembangan | Selesai | Log prompt `P-001`, `docs/progress/CHANGELOG.md` |
| Phase 0 | Foundation: struktur, config, migrasi, auth | **Selesai** | Skeleton + migrasi `001`-`009` + bootstrap admin (`T-003`/`T-004`, P-018/P-020) + auth/JWT/RBAC (`T-005`, P-021): 4 endpoint auth hidup, 22 tabel, 104 baris `role_permissions` |
| Phase 1 | Core Domain: project, document, version, storage | **Sebagian besar selesai** | Project (`T-035`, P-022) dan document + versi + storage + unduh + pencarian (`T-037`, P-023) hidup: 15 endpoint. Sisa exit criteria yang belum diverifikasi: filter kategori/owner/rentang tanggal pada dokumen (`OPEN-QUESTIONS` Q-016 butir 8) |
| Phase 2 | Workflow engine | Belum dimulai | `43-WORKFLOW.md` sudah terkunci lewat ADR-0015/ADR-0016, belum ada kode |
| Phase 3 | Task & comment | **Sebagian** — Task selesai lebih awal | Task (`T-038`, P-025/P-026): 5 endpoint dengan cakupan baris kedua dan penyaring server-side. **Comment belum.** Catatan urutan: Task (fase ini) dikerjakan sebelum Phase 2 karena mandiri, murah, dan sudah menutup pola cakupan §3.1.3; urutan sisanya diputuskan di `OPEN-QUESTIONS.md` **Q-018** |
| Phase 4 | UI frontend | Diblokir | Menunggu isi `DESIGN.md` dan pilihan mode antislop (Q-001, Q-002) |
| Phase 5 | Polish & deployment | Belum dimulai | |

**Total Estimated Timeline:** 16 weeks (4 months)

**Aturan pelaporan:** jangan pernah menaikkan status fase tanpa bukti verifikasi (build/test/click-through) yang tercatat di ledger progress.

**Catatan koreksi (P-026, temuan C-047):** tabel ini sempat tertinggal jauh — Phase 0 dan Phase 1 masih tertulis "Belum dimulai" padahal keduanya sudah berjalan, dan sesi-sesi sebelumnya melabeli modul Task sebagai "Phase 1" sementara tabel fase di atas menempatkannya di **Phase 3** (diikuti `TASKS.md` yang menaruh "Task, overdue detection, comment" pada backlog fase 3). Sumber kebenaran urutan fase adalah **dokumen ini**; ledger mencatat *fase tempat pekerjaan itu berada*, dan bila sebuah fase dikerjakan lebih awal hal itu dinyatakan terus terang seperti pada baris Phase 3.
