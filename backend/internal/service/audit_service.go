// Package service memuat business logic dan penulisan audit log.
//
// Aturan yang mengikat (ADR-0011): setiap aksi kritis dijalankan dalam satu
// transaksi — perubahan data DAN pemanggilan audit log, lalu commit. Gagal
// menulis audit membatalkan transaksi, bukan diabaikan. Handler tidak pernah
// menyimpan atau memanggil service ini.
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Nama aksi audit yang dipakai modul auth. Sumber nama: FR-AUDIT-01
// (`20-SRS.md`) — "login" wajib tercatat. Penulisan huruf besar mengikuti
// contoh yang sudah dipakai dokumen (`DOCUMENT_CREATED`, `DOCUMENT_VERSION_CREATED`).
const (
	ActionLogin  = "LOGIN"
	ActionLogout = "LOGOUT"
)

// AuditService menulis entri `audit_logs` (`41-DATABASE.md` §2.5).
//
// Tabelnya append-only (FR-AUDIT-03, `44-SECURITY.md` §6): service hanya pernah
// menjalankan INSERT. Tidak ada endpoint maupun code path yang meng-UPDATE,
// men-DELETE, atau men-TRUNCATE tabel itu — trigger di migrasi `007` akan
// menolaknya dengan SQLSTATE `23001`.
type AuditService struct {
	db pgx.Tx
}

// NewAuditService membuat layanan audit yang menulis di dalam transaksi `tx`.
//
// Signature ini adalah keputusan ADR-0011 butir 2: `Log` menerima `pgx.Tx`,
// bukan pool, supaya entri audit ikut ter-rollback bersama datanya.
func NewAuditService(tx pgx.Tx) *AuditService {
	return &AuditService{db: tx}
}

// Log menulis satu entri audit.
func (s *AuditService) Log(
	ctx context.Context,
	actorID uuid.UUID,
	action, entity, entityID, description string,
	metadata map[string]any,
) error {
	var raw []byte
	if len(metadata) > 0 {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("encode metadata audit: %w", err)
		}
		raw = encoded
	}

	if _, err := s.db.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, action, entity, entity_id, description, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		actorID, action, entity, entityID, description, raw,
	); err != nil {
		return fmt.Errorf("tulis entri audit %s: %w", action, err)
	}
	return nil
}
