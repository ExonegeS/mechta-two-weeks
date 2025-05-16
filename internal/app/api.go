package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/config"
	"github.com/ExonegeS/mechta-two-weeks/internal/adapters/grpc"
	"github.com/ExonegeS/mechta-two-weeks/internal/adapters/http/handlers"
	"github.com/ExonegeS/mechta-two-weeks/internal/adapters/http/middleware"
	mind_box "github.com/ExonegeS/mechta-two-weeks/internal/adapters/mindbox"
	"github.com/ExonegeS/mechta-two-weeks/internal/core/service"
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

	cfg := &mind_box.ConfigSt{
		Timeout:            s.cfg.ExternalService.Timeout,
		Uri:                s.cfg.ExternalService.URI,
		RetryCount:         5,
		RetryInterval:      5 * time.Second,
		InsecureSkipVerify: false,

		MaxRetries:    5,
		ResetDuration: 15 * time.Second,

		SECRET_KEY: s.cfg.ExternalService.SecretKey,
	}

	entityProvider, err := mind_box.New(cfg)
	if err != nil {
		return err
	}
	workerService := service.NewSyncService(s.cfg.WorkerConfig, s.logger, time.Now, entityProvider)
	SessionHandler := handlers.NewWorkerHandler(s.logger, workerService)
	SessionHandler.RegisterEndpoints(mux)

	go grpc.StartGRPCServer(s.cfg.Server.GRPCPort, workerService, s.logger)

	MWChain := middleware.NewMiddlewareChain(middleware.RecoveryMW, middleware.NewTimeoutContextMW(120))

	serverAddress := fmt.Sprintf("%s:%s", s.cfg.Server.Address, s.cfg.Server.Port)
	s.logger.Info("starting server", slog.String("host", serverAddress))
	httpServer := http.Server{
		Addr:    serverAddress,
		Handler: MWChain(mux),
	}
	return httpServer.ListenAndServe()
}
