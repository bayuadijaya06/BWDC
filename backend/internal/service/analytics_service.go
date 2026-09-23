package service

import (
	"context"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/repository"
)

// AnalyticsService menghitung dashboard (`52-DASHBOARD-ANALYTICS.md`, ADR-0026).
//
// Tidak menambah tabel/kolom; semua hitungan dari transaksi yang sudah ada.
type AnalyticsService struct {
	analytics *repository.AnalyticsRepository
	users     *repository.UserRepository
}

func NewAnalyticsService(analytics *repository.AnalyticsRepository, users *repository.UserRepository) *AnalyticsService {
	return &AnalyticsService{analytics: analytics, users: users}
}

// Dashboard menghitung KPI 6 + chart 8 MVP dengan cakupan `44-SECURITY.md` §3.1.3.
//
// Non-Administrator hanya melihat data project tempat ia menjadi anggota.
func (s *AnalyticsService) Dashboard(ctx context.Context, actor Actor, q dto.AnalyticsQuery) (*dto.DashboardResponse, error) {
	scope, err := systemScope(ctx, s.users, actor)
	if err != nil {
		return nil, err
	}
	return s.analytics.DashboardData(ctx, scope, q)
}
