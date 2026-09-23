package model_test

import (
	"testing"

	"bwdcs/backend/internal/model"
)

// TestCommentEntityTypesMatchDatabaseVocabulary menjaga agar kosakata tertutup
// jenis entitas komentar tetap sama dengan `CHECK` pada kolom
// `comments.entity_type` (`41-DATABASE.md` §2.5, migrasi `007`).
//
// Test ini sengaja menyebut nilainya secara literal: kalau salah satu sisi
// berubah tanpa sisi lain, satu-satunya cara perubahan itu ketahuan adalah test
// yang menuliskan daftarnya sendiri, bukan yang membangunnya dari fungsi yang
// sedang diuji.
func TestCommentEntityTypesMatchDatabaseVocabulary(t *testing.T) {
	want := []string{"project", "document", "task", "workflow"}
	got := model.CommentEntityTypes()

	if len(got) != len(want) {
		t.Fatalf("jumlah jenis entitas = %d, diharapkan %d: %v", len(got), len(want), got)
	}
	for i, value := range want {
		if got[i] != value {
			t.Errorf("jenis entitas ke-%d = %q, diharapkan %q", i, got[i], value)
		}
	}
}

func TestNormalizeCommentEntityType(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"document", "document"},
		{"Document", "document"},
		{"  DOCUMENT  ", "document"},
		{"workflow", "workflow"},
		{"", ""},
		{"   ", ""},
		// Nama panjang **tidak** dinormalkan menjadi `workflow`: menerima dua
		// nama untuk satu kolom berarti setiap pemakaian berikutnya harus
		// menebak mana yang kanonik.
		{"workflow_instance", "workflow_instance"},
	}

	for _, tc := range cases {
		if got := model.NormalizeCommentEntityType(tc.input); got != tc.want {
			t.Errorf("normalisasi %q = %q, diharapkan %q", tc.input, got, tc.want)
		}
	}
}

func TestIsCommentEntityType(t *testing.T) {
	for _, value := range model.CommentEntityTypes() {
		if !model.IsCommentEntityType(value) {
			t.Errorf("%q ditolak sebagai jenis entitas yang sah", value)
		}
	}

	for _, value := range []string{"", "projects", "workflow_instance", "user", "PROJECT"} {
		if model.IsCommentEntityType(value) {
			t.Errorf("%q diterima sebagai jenis entitas, padahal tidak sah (huruf besar harus dinormalkan lebih dulu)", value)
		}
	}

	if model.CommentContentMaxLength != 2000 {
		t.Errorf("batas isi komentar = %d, diharapkan 2000 (`50-FSD.md` §7)", model.CommentContentMaxLength)
	}
}
