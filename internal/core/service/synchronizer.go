package service

import (
	"context"
	"sync"
	"time"

	"log/slog"

	"github.com/ExonegeS/mechta-two-weeks/config"
	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
)

type EntityDataProvider interface {
	GetFinalPriceInfo(ctx context.Context, reqObj *domain.ImportModelReq) ([]*domain.ImportModelRep, error)
	GetPromotionsInfo(ctx context.Context) ([]*domain.ImportPromotionsRep, error)
	GetExportData(ctx context.Context, operation string) (*domain.PromotionsGetInfoRepSt, error)
}

type Result struct {
	Req  *domain.ImportModelReq
	Data []*domain.ImportModelRep
	Err  error
}

type SyncService struct {
	cfg           config.WorkerConfig
	logger        *slog.Logger
	timeSource    func() time.Time
	entityDataAPI EntityDataProvider
}

func NewSyncService(
	cfg config.WorkerConfig,
	logger *slog.Logger,
	timeSource func() time.Time,
	api EntityDataProvider,
) *SyncService {
	return &SyncService{cfg, logger, timeSource, api}
}

func (s *SyncService) GetData(
	ctx context.Context,
	subdivisionId string,
	calculationTime time.Time,
	products []*domain.BasePrice,
) (processed []*domain.ImportModelRep, failed []*domain.BasePrice, err error) {

	batchSize := int(s.cfg.BatchSize)
	if batchSize < 1 {
		batchSize = len(products)
	}

	totalBatches := (len(products) + batchSize - 1) / batchSize
	jobs := make(chan *domain.ImportModelReq, totalBatches)
	results := make(chan Result, totalBatches)
	var wg sync.WaitGroup

	numWorkers := int(s.cfg.MaxWorkers)
	if numWorkers < 1 {
		numWorkers = 1
	}
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range jobs {
				data, err := s.entityDataAPI.GetFinalPriceInfo(ctx, req)
				if err != nil {
					results <- Result{Req: req, Err: err}
					continue
				}
				results <- Result{Data: data, Err: nil}
			}
		}()
	}

	go func() {
		for i := 0; i < len(products); i += batchSize {
			high := i + batchSize
			if high > len(products) {
				high = len(products)
			}
			chunk := products[i:high]
			req := &domain.ImportModelReq{
				SubdivisionId:   subdivisionId,
				CalculationTime: calculationTime,
				Products:        chunk,
			}
			select {
			case jobs <- req:
			case <-ctx.Done():
				break
			}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if res.Err != nil {
			s.logger.Error("GetData worker error", slog.String("err", res.Err.Error()))
			failed = append(failed, res.Req.Products...)
			continue
		}
		s.logger.Info("GetData worker success", slog.Int("processed size:", len(res.Data)))
		processed = append(processed, res.Data...)
	}

	return processed, failed, nil
}

func (s *SyncService) GetPromotionsInfo(
	ctx context.Context,
) ([]*domain.ImportPromotionsRep, error) {
	return s.entityDataAPI.GetPromotionsInfo(ctx)
}
