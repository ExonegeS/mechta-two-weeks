package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/config"
	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
)

type EntityDataProvider interface {
	GetUserData(ttl time.Duration) (domain.Data, error)
}

type SyncService struct {
	cfg           config.WorkerConfig
	logger        *slog.Logger
	timeSource    func() time.Time
	entityDataAPI EntityDataProvider
}

func NewSyncService(cfg config.WorkerConfig, logger *slog.Logger, timeSource func() time.Time, entityDataAPI EntityDataProvider) *SyncService {
	return &SyncService{
		cfg,
		logger,
		timeSource,
		entityDataAPI,
	}
}

func (s *SyncService) GetData(ctx context.Context, id int64) (*domain.Data, error) {
	// Create workers
	s.logger.Info("Starting batch processing with synchronized result collection...")
	dispatcher(int(s.cfg.MaxJobs), int(s.cfg.MaxWorkers))

	data, err := s.entityDataAPI.GetUserData(time.Hour)
	if err != nil {
		return nil, err
	}
	return &data, err
}

type Job struct {
	ID    int
	Value int
}

type Result struct {
	JobID  int
	Square int
}

func worker(id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		results <- Result{JobID: job.ID, Square: job.Value * job.Value}
	}
}

func collectResults(results <-chan Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for result := range results {
		fmt.Printf("Job ID: %d, Input: %d, Squared Value: %d\n", result.JobID, result.JobID, result.Square)
	}
}

func dispatcher(jobCount, workerCount int) {
	jobs := make(chan Job, jobCount)
	results := make(chan Result, jobCount)

	var wg sync.WaitGroup

	// Start workers
	wg.Add(workerCount)
	for w := 1; w <= workerCount; w++ {
		go worker(w, jobs, results, &wg)
	}

	// Start collecting results
	var resultsWg sync.WaitGroup
	resultsWg.Add(1)
	go collectResults(results, &resultsWg)

	// Distribute jobs and wait for completion
	for j := 1; j <= jobCount; j++ {
		jobs <- Job{ID: j, Value: j}
	}
	close(jobs)
	wg.Wait()
	close(results)

	// Ensure all results are collected
	resultsWg.Wait()
}
