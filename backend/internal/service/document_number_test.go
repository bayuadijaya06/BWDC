package service_test

import (
	"context"
	"sync"
	"testing"

	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// TestCreateDocument_AssignsSequentialNumber menutup ADR-0017: nomor dibaca
// dari `document_sequences` (bukan COUNT(*)/MAX()), dan berurutan per project.
func TestCreateDocument_AssignsSequentialNumber(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "NUMSER")

	first := fixture.createDocument(owner, projectID, "Dokumen pertama")
	second := fixture.createDocument(owner, projectID, "Dokumen kedua")

	if first.Document.DocumentNumber != "NUMSER-001" {
		t.Errorf("nomor dokumen pertama %q, diharapkan NUMSER-001", first.Document.DocumentNumber)
	}
	if second.Document.DocumentNumber != "NUMSER-002" {
		t.Errorf("nomor dokumen kedua %q, diharapkan NUMSER-002", second.Document.DocumentNumber)
	}
}

// TestCreateDocument_NumberIsPerProject memastikan penghitungnya memang milik
// project, bukan organisasi: project lain memulai lagi dari 001.
func TestCreateDocument_NumberIsPerProject(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	firstProject := fixture.createProject(owner, "ALPHA")
	secondProject := fixture.createProject(owner, "BETA")

	fixture.createDocument(owner, firstProject, "Milik ALPHA")
	other := fixture.createDocument(owner, secondProject, "Milik BETA")

	if other.Document.DocumentNumber != "BETA-001" {
		t.Errorf("nomor dokumen %q, diharapkan BETA-001 (penghitung per project)", other.Document.DocumentNumber)
	}
}

// TestCreateDocument_RollbackDoesNotConsumeNumber menutup ADR-0017 butir
// "penghitung yang rollback tidak menghabiskan nomor".
//
// Yang diuji langsung adalah transaksi yang benar-benar dibatalkan: kenaikan
// penghitung dilakukan di dalam transaksi yang di-rollback, lalu dokumen
// berikutnya harus tetap mendapat nomor itu.
func TestCreateDocument_RollbackDoesNotConsumeNumber(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "ROLLBK")
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("mulai transaksi: %v", err)
	}
	number, err := repository.NewDocumentRepository(tx).NextNumber(ctx, projectID)
	if err != nil {
		t.Fatalf("bangkitkan nomor di dalam transaksi: %v", err)
	}
	if number != 1 {
		t.Fatalf("nomor pertama %d, diharapkan 1", number)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	document := fixture.createDocument(owner, projectID, "Sesudah rollback")
	if document.Document.DocumentNumber != "ROLLBK-001" {
		t.Errorf("nomor sesudah rollback %q, diharapkan ROLLBK-001 (nomor tidak terpakai)",
			document.Document.DocumentNumber)
	}
}

// TestCreateDocument_ConcurrentCreatesGetDistinctNumbers menutup ADR-0017:
// dua pembuatan bersamaan mendapat nomor berbeda, bukan 409.
//
// Test ini wajib berjalan di PostgreSQL nyata (`70-TESTING.md` §3.5): jaminannya
// ada di `INSERT ... ON CONFLICT ... RETURNING` pada `document_sequences`, bukan
// di kode aplikasi.
func TestCreateDocument_ConcurrentCreatesGetDistinctNumbers(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "KONKUR")

	const parallel = 2
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		numbers = make([]string, 0, parallel)
		errors  = make([]error, 0, parallel)
	)

	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			document, err := fixture.documents.Create(context.Background(), actorOf(owner), service.CreateDocumentInput{
				ProjectID: projectID,
				Title:     "Dokumen paralel",
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errors = append(errors, err)
				return
			}
			numbers = append(numbers, document.Document.DocumentNumber)
		}(i)
	}
	wg.Wait()

	if len(errors) != 0 {
		t.Fatalf("pembuatan paralel gagal: %v", errors)
	}
	if len(numbers) != parallel {
		t.Fatalf("jumlah dokumen %d, diharapkan %d", len(numbers), parallel)
	}

	seen := map[string]bool{}
	for _, number := range numbers {
		if seen[number] {
			t.Fatalf("nomor %q dipakai dua kali: %v", number, numbers)
		}
		seen[number] = true
	}
	for _, expected := range []string{"KONKUR-001", "KONKUR-002"} {
		if !seen[expected] {
			t.Errorf("nomor %q tidak ada di antara %v", expected, numbers)
		}
	}
}

// TestCreateDocument_RejectsProjectOutsideScope memastikan nomor tidak
// dibangkitkan untuk project di luar cakupan: dokumen di project orang lain
// dibalas seolah project itu tidak ada (`44-SECURITY.md` §3.1.3).
func TestCreateDocument_RejectsProjectOutsideScope(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "TERTUTUP")

	outsider := fixture.createOrgAndUser("manager")

	_, err := fixture.documents.Create(context.Background(), actorOf(outsider), service.CreateDocumentInput{
		ProjectID: projectID,
		Title:     "Percobaan dari luar cakupan",
	})
	requireError(t, err, service.ErrProjectNotFound)

	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM document_sequences WHERE project_id = $1`, projectID).Scan(&count); err != nil {
		t.Fatalf("periksa document_sequences: %v", err)
	}
	if count != 0 {
		t.Errorf("penghitung nomor bertambah %d baris untuk permintaan yang ditolak", count)
	}
}
