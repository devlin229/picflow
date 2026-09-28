package service

import (
	"context"
	"fmt"
	"time"

	"picflow/backend/internal/errno"
	"picflow/backend/internal/svc"
	"picflow/backend/internal/types"
)

type HealthService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthService(ctx context.Context, svcCtx *svc.ServiceContext) *HealthService {
	return &HealthService{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *HealthService) Live() (*types.LiveResponse, error) {
	return &types.LiveResponse{
		Status: "ok",
	}, nil
}

func (s *HealthService) Ready() (*types.ReadyResponse, error) {
	if s.svcCtx.DB != nil {
		sqlDB, err := s.svcCtx.DB.DB()
		if err != nil {
			return nil, errno.Unavailable(fmt.Errorf("get database connection pool: %w", err))
		}
		pingCtx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
		defer cancel()
		if err = sqlDB.PingContext(pingCtx); err != nil {
			return nil, errno.Unavailable(fmt.Errorf("ping database: %w", err))
		}
	}
	return &types.ReadyResponse{Status: "ready"}, nil
}
