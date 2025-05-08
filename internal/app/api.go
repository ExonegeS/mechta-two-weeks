package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/config"
	mind_box "github.com/ExonegeS/mechta-two-weeks/internal/adapters/mindbox"
	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
	"github.com/ExonegeS/mechta-two-weeks/internal/core/service"
	"github.com/ExonegeS/mechta-two-weeks/internal/ports/http/handlers"
	"github.com/ExonegeS/mechta-two-weeks/internal/ports/http/middleware"
)

type APIServer struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewAPIServer(config *config.Config, logger *slog.Logger) *APIServer {
	return &APIServer{
		config,
		logger,
	}
}

func (s *APIServer) Run() error {
	mux := http.NewServeMux()

	opts := &domain.OptionsSt{
		Timeout:            s.cfg.ExternalService.Timeout,
		Uri:                s.cfg.ExternalService.MakeAddressString(),
		RetryCount:         5,
		InsecureSkipVerify: false,
	}

	entityProvider, err := mind_box.New(opts, mind_box.NewCircuitBreaker(5, 15*time.Second))
	if err != nil {
		return err
	}
	workerService := service.NewSyncService(s.cfg.WorkerConfig, s.logger, time.Now, entityProvider)
	SessionHandler := handlers.NewWorkerHandler(s.logger, workerService)
	SessionHandler.RegisterEndpoints(mux)

	MWChain := middleware.NewMiddlewareChain(middleware.RecoveryMW, middleware.NewTimeoutContextMW(120))

	serverAddress := fmt.Sprintf("%s:%s", s.cfg.Server.Address, s.cfg.Server.Port)
	s.logger.Info("starting server", slog.String("host", serverAddress))
	httpServer := http.Server{
		Addr:    serverAddress,
		Handler: MWChain(mux),
	}
	return httpServer.ListenAndServe()
}
