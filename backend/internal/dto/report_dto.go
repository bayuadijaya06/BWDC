package dto

// ReportExportQuery adalah query GET /reports/export?type=&format=&project_id=&status=&...
//
// MVP hanya CSV. `type` menentukan sumber data, `format` hanya `csv`.
// Kolom CSV = kolom tabel pada halaman terkait di 50-FSD (aturan 42-API §10).
type ReportExportQuery struct {
	Type      string  `form:"type"`
	Format    string  `form:"format"`
	ProjectID *string `form:"project_id"`
	Status    *string `form:"status"`
	Search    *string `form:"search"`
}

// ReportExportResponse tidak dipakai sebagai JSON: endpoint mengembalikan
// `text/csv` stream dengan `Content-Disposition: attachment`.
// Tipe ini hanya untuk dokumentasi kontrak.
type ReportExportResponse struct {
	Filename string `json:"filename"`
	Size     int    `json:"size"`
}
