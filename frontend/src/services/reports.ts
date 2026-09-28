import { filenameFromDisposition } from "@/utils/download";
import { http } from "./http";

/**
 * Tipe laporan `GET /reports/export` (`42-API.md` §10) — sama dengan tiga
 * sub-menu `50-FSD.md` §10.6. `format` tidak dijadikan parameter: MVP hanya
 * CSV, dan nilai lain ditolak server `422` (FR-REP-01).
 */
export type ReportType = "projects" | "documents" | "tasks";

/**
 * Penyaring export. Bentuknya = irisan parameter daftar (§3, §4, §6) yang
 * benar-benar diterima endpoint export: `type`, `format`, `project_id`,
 * `status`, `search` (`42-API.md` §10, `parseReportExportQuery`). Penyaring
 * lain (kategori, pemilik, rentang tanggal) **tidak** dikirim — halaman yang
 * memanggil menyatakan itu terbuka, bukan menyamarkannya sebagai tersaring.
 */
export interface ReportExportQuery {
  type: ReportType;
  search?: string;
  status?: string;
  project_id?: string;
}

export interface ExportedReport {
  blob: Blob;
  filename: string;
}

/**
 * Mengunduh export CSV. Berkasnya diambil sebagai blob karena endpointnya
 * menuntut header `Authorization` (pola yang sama dengan
 * `downloadDocumentVersion` di `services/documents.ts`). Nama berkas dari
 * `Content-Disposition` server (`bwdcs-<type>-<YYYYMMDD>.csv`); cadangannya
 * tanpa tanggal karena tanggal milik server, bukan karangan klien.
 */
async function exportReport(query: ReportExportQuery): Promise<ExportedReport> {
  const params: Record<string, string> = {
    type: query.type,
    format: "csv",
  };
  const search = query.search?.trim();
  if (search) params.search = search;
  if (query.status) params.status = query.status;
  if (query.project_id) params.project_id = query.project_id;

  const response = await http.get<Blob>("/reports/export", {
    params,
    responseType: "blob",
  });
  const headers = response.headers as Record<string, string | undefined>;
  return {
    blob: response.data,
    filename: filenameFromDisposition(
      headers["content-disposition"],
      `bwdcs-${query.type}.csv`,
    ),
  };
}

export { exportReport };
