package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/config"
	syncserver "github.com/ExonegeS/mechta-two-weeks/internal/adapters/syncServer"
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

	entityProvider := syncserver.NewEntityProvider(s.cfg.ExternalService.MakeAddressString(), s.cfg.ExternalService.Timeout)
	workerService := service.NewSyncService(s.cfg.WorkerConfig, time.Now, entityProvider)
	SessionHandler := handlers.NewWorkerHandler(s.logger, workerService)
	SessionHandler.RegisterEndpoints(mux)

	MWChain := middleware.NewMiddlewareChain(middleware.RecoveryMW, middleware.NewTimeoutContextMW(15))

	serverAddress := fmt.Sprintf("%s:%s", s.cfg.Server.Address, s.cfg.Server.Port)
	s.logger.Info("starting server", slog.String("host", serverAddress))
	httpServer := http.Server{
		Addr:    serverAddress,
		Handler: MWChain(mux),
	}
	return httpServer.ListenAndServe()
}
