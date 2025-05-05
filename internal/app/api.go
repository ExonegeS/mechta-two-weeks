package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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

	entityProvider := mind_box.New(&domain.OptionsSt{
		Timeout:       s.cfg.ExternalService.Timeout,
		Uri:           s.cfg.ExternalService.MakeAddressString(),
		Params:        url.Values{},
		Headers:       http.Header{},
		RetryCount:    3,
		RetryInterval: 100 * time.Millisecond,
	})
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
