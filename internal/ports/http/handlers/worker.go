package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
	"github.com/ExonegeS/mechta-two-weeks/internal/utils"
)

type WorkerService interface {
	GetData(ctx context.Context, subdivisionId string, calculationTime time.Time, products []*domain.BasePrice) ([]*domain.ImportModelRep, error)
}

type WorkerHandler struct {
	logger        *slog.Logger
	workerService WorkerService
}

func NewWorkerHandler(logger *slog.Logger, workerService WorkerService) *WorkerHandler {
	return &WorkerHandler{
		logger:        logger,
		workerService: workerService,
	}
}

func (h *WorkerHandler) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.RootFunc)
	mux.HandleFunc("GET /hello", h.HelloFunc)
	mux.HandleFunc("GET /err", h.ErrorFunc)
	mux.HandleFunc("GET /data/{id}", h.GetData)
	// id подразделения, массив структур base price
	// record timeSince
	//
}

func (h *WorkerHandler) RootFunc(w http.ResponseWriter, r *http.Request) {
	const op = "WorkerHandler.RootFunc"

	err := fmt.Errorf("endpoint not found: %v", r.URL.Path)
	if err != nil {
		h.logger.Error("HandlerError", slog.String("operation", op), slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusNotFound, fmt.Errorf("endpoint not found"))
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{})
}

func (h *WorkerHandler) HelloFunc(w http.ResponseWriter, r *http.Request) {
	const op = "WorkerHandler.HelloFunc"
	var err error

	if err != nil {
		h.logger.Error("HandlerError", slog.String("operation", op), slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusNotImplemented, fmt.Errorf("not implemented"))
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"message": "Hello world!",
	})
}

func (h *WorkerHandler) ErrorFunc(w http.ResponseWriter, r *http.Request) {
	const op = "WorkerHandler.ErrorFunc"

	err := fmt.Errorf("not implemented")
	if err != nil {
		h.logger.Error("HandlerError", slog.String("operation", op), slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusNotImplemented, fmt.Errorf("not implemented"))
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{})
}

func (h *WorkerHandler) GetData(w http.ResponseWriter, r *http.Request) {
	const op = "WorkerHandler.GetData"

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.logger.Error("HandlerError", slog.String("operation", op), slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid data ID format"))
		return
	}

	data, err := h.workerService.GetData(r.Context(),
		"5",
		time.Now(),
		[]*domain.BasePrice{
			&domain.BasePrice{
				"93",
				13.7,
			},
			&domain.BasePrice{
				"93",
				13.7,
			},
			&domain.BasePrice{
				"93",
				13.7,
			},
			&domain.BasePrice{
				"93",
				13.7,
			},
			&domain.BasePrice{
				"93",
				13.7,
			},
			&domain.BasePrice{
				"93",
				13.7,
			},
			&domain.BasePrice{
				"93",
				13.7,
			},
		})
	if err != nil {
		h.logger.Error("HandlerError", slog.String("operation", op), slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusInternalServerError, fmt.Errorf("cannot access data with id %d", id))
		return
	}
	utils.WriteJSON(w, http.StatusOK, struct {
		ID    int64 `json:id`
		Value any   `json:value`
	}{
		ID:    int64(id),
		Value: data,
	})
}
