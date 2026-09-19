package service

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// TestLimitedReaderStopsAtLimit membuktikan penjaga ukuran bekerja saat berkas
// mengalir: header multipart yang mengaku kecil tidak dapat menyelundupkan
// isi yang lebih besar (`44-SECURITY.md` §4.2, batas 100 MB `50-FSD.md` §4.2).
//
// Test ini internal karena `limitedReader` sengaja tidak diekspor: yang boleh
// memakainya hanya service ini, dan buktinya cukup di sini.
func TestLimitedReaderStopsAtLimit(t *testing.T) {
	content := []byte("0123456789")

	cases := []struct {
		name      string
		remaining int64
		wantErr   bool
		wantRead  int64
	}{
		{name: "isi lebih kecil dari kuota", remaining: 20, wantRead: int64(len(content))},
		{name: "isi sama dengan kuota", remaining: int64(len(content)), wantRead: int64(len(content))},
		{name: "isi melampaui kuota", remaining: 4, wantErr: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			reader := &limitedReader{reader: bytes.NewReader(content), remaining: testCase.remaining}

			written, err := io.Copy(io.Discard, reader)
			if testCase.wantErr {
				if !errors.Is(err, ErrDocumentFileTooLarge) {
					t.Fatalf("error %v, diharapkan ErrDocumentFileTooLarge", err)
				}
				if written != testCase.remaining {
					t.Errorf("byte terbaca %d, diharapkan %d (berhenti tepat di batas)",
						written, testCase.remaining)
				}
				if reader.Size() != testCase.remaining {
					t.Errorf("Size() %d, diharapkan %d", reader.Size(), testCase.remaining)
				}
				return
			}

			if err != nil {
				t.Fatalf("baca isi: %v", err)
			}
			if written != testCase.wantRead || reader.Size() != testCase.wantRead {
				t.Errorf("byte terbaca %d / Size() %d, diharapkan %d", written, reader.Size(), testCase.wantRead)
			}
		})
	}
}
