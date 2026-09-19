# P-XXX — YYYY-MM-DD — <judul singkat sesi>

> Salin file ini menjadi `docs/progress/prompts/P-<nomor>-<YYYY-MM-DD>-<slug>.md`.
> Nomor urut meneruskan prompt terakhir di direktori ini (naik, tidak pernah dipakai ulang).

| Field | Isi |
|---|---|
| ID | P-XXX |
| Waktu mulai | YYYY-MM-DD HH:MM (zona waktu lokal) |
| Aktor | agen / manusia (sebutkan tool atau nama) |
| Model / agen | mis. Buffy |
| Fase roadmap | 0 / 1 / 2 / 3 / 4 / 5 / pra-fase |
| Task terkait | `T-###` dari `TASKS.md` |
| Status akhir | DONE / PARTIAL / BLOCKED / FAILED |

---

## 1. Prompt User

> Tulis ringkasan prompt apa adanya (maksimal 5 baris). Jangan menafsirkan sampai bagian "Interpretasi".

## 2. Interpretasi & Scope

- Yang diminta:
- Yang TIDAK termasuk (out of scope):
- Asumsi yang diambil:
- Pertanyaan yang muncul (link ke `OPEN-QUESTIONS.md`):

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | | |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | | | |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| | | | |

> Jangan lupa menyalin daftar ini ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | | | PASS / FAIL |

- [ ] Typecheck / build dijalankan
- [ ] Test relevan dijalankan
- [ ] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3)

> Tanpa bagian ini, status pekerjaan hanya boleh `PARTIAL`, bukan `DONE`.

## 7. Hasil & Dampak

- Selesai:
- Belum selesai / sisa:
- Risiko / utang teknis:
- Dampak ke dokumen desain (apakah ada dokumen yang harus ikut diubah?):

## 8. Update Ledger (Checklist Wajib)

- [ ] `STATE.md` diperbarui
- [ ] `SESSION-LOG.md` ditambah entri
- [ ] `CHANGELOG.md` ditambah entri
- [ ] `TASKS.md` diperbarui (status task baru/berubah)
- [ ] `TRACEABILITY.md` diperbarui (bila menyentuh requirement)
- [ ] `OPEN-QUESTIONS.md` diperbarui (bila ada pertanyaan baru)
- [ ] ADR dibuat/diperbarui (bila ada keputusan arsitektur)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| | | |
