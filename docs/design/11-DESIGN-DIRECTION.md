# 11-DESIGN-DIRECTION — Design Direction Brief

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** **SUPERSEDED oleh `DESIGN.md`** (ADR-0007 jalur 2, sesi P-037) — dokumen ini adalah kuesioner aslinya, disimpan sebagai riwayat. **Jangan mengisinya dan jangan membacanya sebagai arah desain yang berlaku.** Arah desain BWDCS ada di root repo: `DESIGN.md`.

Alasan: user memilih **jalur 2** ADR-0007 dan memberi izin eksplisit kepada agen untuk menyusun arah desain (P-037). Nilai `[TUNGGU INPUT]` di bawah karena itu **tidak lagi berarti "belum ada keputusan"**, melainkan "kuesioner ini tidak dipakai": pertanyaannya dijawab di `DESIGN.md` §1-§8, dan nilainya dikunci di `frontend/src/styles/tokens.css` serta diperiksa test kontras.

---

## Pertanyaan Arah Desain

Sebelum membangun UI, perlu ditentukan arah desain. Berikut pertanyaan kunci (jawaban berlaku: `DESIGN.md`):

### 1. Identity & Personality

BWDCS adalah **internal business tool** untuk pengelolaan dokumen dan workflow perusahaan. 

Tugas untuk user:
- Product owner / stakeholder: isi identitas (misal: formal, modern, trustworthy, atau pragmatic)
- Atau serahkan pada agen dengan warning bahwa hasilnya akan berupa draft tanpa arah spesifik

### 2. Palette (maksimal 2-3 warna inti + 1 accent)

| Peran | Warna Hex | Purpose |
|---|---|---|
| Primary | `[TUNGGU INPUT]` | Action buttons, active states, links |
| Secondary | `[TUNGGU INPUT]` | Secondary actions, borders |
| Neutral | `[TUNGGU INPUT]` | Background, text |
| Accent | `[TUNGGU INPUT]` | Highlight, focus, important status |

### 3. Typography

| Role | Font | Reason |
|---|---|---|
| Primary | `[TUNGGU INPUT]` | [ALASAN] |
| Mono | `[TUNGGU INPUT]` | [ALASAN] |

### 4. Mood & Feel

- [ ] Professional & trustworthy
- [ ] Modern & clean
- [ ] Dense information, tool-like
- [ ] Other: [isi]

---

> **Catatan historis:** dokumen ini dahulu adalah template yang diisi user. Sejak P-037 statusnya `SUPERSEDED`: `DESIGN.md` sudah terisi, sehingga **"draft without direction" tidak berlaku lagi** dan dial sementara 1/1/1 tidak dipakai. Dial resmi (ENERGY 1 / RHYTHM 2 / MOTION 1) beserta alasannya ada di `DESIGN.md` §5, dan Design Read yang berlaku ada di `DESIGN.md` §9.

---

## Design Read (historis — yang berlaku ada di `DESIGN.md` §9)

Bentuk yang dipakai:

> "Reading this as: <page kind> for <audience>, in a <visual language> style, dial <ENERGY/RHYTHM/MOTION>."

Yang berlaku untuk BWDCS: *"Reading this as: internal document-control console for administrators, managers, and contributors who process records daily, in a ruled-ledger visual language (paper, ink, mono index numbers), dial ENERGY 1 / RHYTHM 2 / MOTION 1."* Setiap halaman baru wajib menuliskan Design Read-nya di log prompt sesi itu.
