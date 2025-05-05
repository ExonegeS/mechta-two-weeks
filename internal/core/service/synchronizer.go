package service

import (
	"context"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/config"
	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
)

type EntityDataProvider interface {
	GetUserData(ttl time.Duration) (domain.Data, error)
}

type SyncService struct {
	cfg           config.WorkerConfig
	timeSource    func() time.Time
	entityDataAPI EntityDataProvider
}

func NewSyncService(cfg config.WorkerConfig, timeSource func() time.Time, entityDataAPI EntityDataProvider) *SyncService {
	return &SyncService{
		cfg,
		timeSource,
		entityDataAPI,
	}
}

func (s *SyncService) GetData(ctx context.Context, id int64) (*domain.Data, error) {
	data, err := s.entityDataAPI.GetUserData(time.Hour)
	if err != nil {
		return nil, err
	}
	return &data, err
}
