package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
	"github.com/ExonegeS/mechta-two-weeks/internal/utils"
)

type WorkerService interface {
	GetData(ctx context.Context, subdivisionId string, calculationTime time.Time, products []*domain.BasePrice) ([]*domain.ImportModelRep, []*domain.BasePrice, error)
}

type WorkerHandler struct {
	logger        *slog.Logger
	workerService WorkerService
	semaphore     chan struct{}
}

func NewWorkerHandler(logger *slog.Logger, workerService WorkerService) *WorkerHandler {
	return &WorkerHandler{
		logger:        logger,
		workerService: workerService,
		semaphore:     make(chan struct{}, 1),
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

	h.logger.Debug("Semaphore acquired")
	h.semaphore <- struct{}{}
	defer func() {
		<-h.semaphore
		h.logger.Debug("Semaphore released")
	}()

	start := time.Now()

	id := r.PathValue("id")

	type request struct {
		Items []*struct {
			ProductId string  `json:"product_id"`
			Price     float64 `json:"price"`
		} `json:"item_list"`
	}
	var req request
	err := utils.ParseJSON(r, &req)
	if err != nil {
		h.logger.Error("HandlerError", slog.String("operation", op), slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid payload"))
		return
	}
	items := make([]*domain.BasePrice, len(req.Items))
	for i, item := range req.Items {
		items[i] = &domain.BasePrice{
			ProductId: item.ProductId,
			Price:     item.Price,
		}
	}

	data, failed, err := h.workerService.GetData(
		r.Context(),
		id,
		time.Now(),
		items,
	)
	if err != nil {
		h.logger.Error("HandlerError",
			slog.String("operation", op),
			slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusInternalServerError,
			fmt.Errorf("cannot access data with id %s", id))
		return
	}
	fmt.Printf("max: %v items/hour\nsucceeded: %v items/hour\n", float64(len(items))/time.Since(start).Hours(), float64(len(data))/time.Since(start).Hours())
	utils.WriteJSON(w, http.StatusOK, struct {
		ID              string `json:"id"`
		TotalProcessed  int    `json:"total_processed"`
		TotalFailed     int    `json:"total_failed"`
		ProcessDuration string `json:"process_duration"`

		Processed any                 `json:"processed"`
		Failed    []*domain.BasePrice `json:"failed"`
	}{
		ID:              id,
		TotalProcessed:  len(data),
		TotalFailed:     len(failed),
		Processed:       data,
		Failed:          failed,
		ProcessDuration: time.Since(start).String(),
	})
}
