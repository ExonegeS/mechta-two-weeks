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
	GetPromotionsInfo(ctx context.Context) ([]*domain.ImportPromotionsRep, error)
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
	mux.HandleFunc("GET /data/{id}", h.GetData)
	mux.HandleFunc("GET /promotions", h.GetPromotionsInfo)
}

func (h *WorkerHandler) RootFunc(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Endpoint string `json:"endpoint"`
		Body     string `json:"body"`
	}
	type response struct {
		API []request `json:"api"`
	}
	utils.WriteJSON(w, http.StatusOK, response{
		API: []request{
			{
				Endpoint: "/data/{id}",
				Body:     "{item_list: [{product_id: string, price: numeric}]}",
			},
			{
				Endpoint: "/promotions",
			},
		},
	})
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
		} `json:"items"`
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

func (h *WorkerHandler) GetPromotionsInfo(w http.ResponseWriter, r *http.Request) {
	const op = "WorkerHandler.GetPromotionsInfo"

	h.logger.Debug("Semaphore acquired")
	h.semaphore <- struct{}{}
	defer func() {
		<-h.semaphore
		h.logger.Debug("Semaphore released")
	}()

	start := time.Now()
	data, err := h.workerService.GetPromotionsInfo(
		r.Context(),
	)
	if err != nil {
		h.logger.Error("HandlerError",
			slog.String("operation", op),
			slog.String("error", err.Error()))
		utils.WriteError(w, http.StatusInternalServerError,
			fmt.Errorf("cannot access PromotionsInfo data"))
		return
	}

	type Promotion struct {
		ExternalID string `json:"external_id"`
		Name       string `json:"name"`
		SchemaID   string `json:"schema_id"`
		StartDate  string `json:"start_date"`
		EndDate    string `json:"end_date"`
	}

	type response struct {
		TotalPromotions int          `json:"total_promotions"`
		ProcessDuration string       `json:"process_duration"`
		Promotions      []*Promotion `json:"promotions"`
	}

	resp := make([]*Promotion, len(data))

	for i, promo := range data {
		resp[i] = &Promotion{
			ExternalID: promo.ExternalID,
			Name:       promo.Name,
			SchemaID:   promo.SchemaID,
			StartDate:  promo.StartDate.Format(time.RFC3339),
			EndDate:    promo.EndDate.Format(time.RFC3339),
		}
	}

	utils.WriteJSON(w, http.StatusOK, response{
		ProcessDuration: time.Since(start).String(),
		TotalPromotions: len(data),
		Promotions:      resp,
	})
}
