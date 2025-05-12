package grpc

import (
	context "context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
	"github.com/ExonegeS/mechta-two-weeks/internal/core/service"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type MindboxServer struct {
	UnimplementedMindboxServiceServer
	service *service.SyncService
	logger  *slog.Logger
}

func NewMindboxServer(logger *slog.Logger, service *service.SyncService) *MindboxServer {
	return &MindboxServer{
		service: service,
		logger:  logger,
	}
}

func StartGRPCServer(grpcPort string, syncService *service.SyncService, logger *slog.Logger) error {
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	invServer := NewMindboxServer(logger, syncService)
	RegisterMindboxServiceServer(grpcServer, invServer)
	reflection.Register(grpcServer)

	logger.Info("gRPC server listening", "port", grpcPort)
	return grpcServer.Serve(lis)
}

func (s *MindboxServer) GetFinalPriceInfo(ctx context.Context, req *GetFinalPriceInfoRequest) (*GetFinalPriceInfoResponse, error) {
	s.logger.Info("GetFinalPriceInfo called", "request", req.GetId(), "products", len(req.GetItems()))

	products := req.GetItems()
	if len(products) == 0 {
		s.logger.Error("No products provided in request")
		return nil, fmt.Errorf("no products provided")
	}

	parsedProducts := make([]*domain.BasePrice, len(products))
	for i, product := range products {
		parsedProducts[i] = &domain.BasePrice{
			ProductId: product.ProductId,
			Price:     product.GetPrice(),
		}
	}
	start := time.Now()
	processed, failed, err := s.service.GetData(ctx, req.GetId(), time.Now(), parsedProducts)
	if err != nil {
		s.logger.Error("Error processing request", "error", err)
		return nil, fmt.Errorf("failed to get final price info: %w", err)
	}

	return &GetFinalPriceInfoResponse{
		Id:              req.GetId(),
		TotalProcessed:  int32(len(processed)),
		TotalFailed:     int32(len(failed)),
		ProcessDuration: time.Since(start).String(),
		Processed:       convertToProtoImportModels(processed),
		Failed:          convertToProtoItems(failed),
	}, nil
}

func convertToProtoImportModels(prices []*domain.ImportModelRep) []*ImportModel {
	result := make([]*ImportModel, len(prices))
	for i, price := range prices {
		promotions := make([]*Promo, len(price.Promotions))
		for j, promo := range price.Promotions {
			parsedPromo := &Promo{
				Id:         int32(promo.Id),
				ExternalId: promo.ExternalId,
				Type:       promo.Type,
				Name:       promo.Name,
				SchemaId:   promo.SchemaId,
			}
			if promo.StartDate != nil {
				parsedPromo.StartDate = timestamppb.New(*promo.StartDate)
			}
			if promo.EndDate != nil {
				parsedPromo.EndDate = timestamppb.New(*promo.EndDate)
			}
			promotions[j] = parsedPromo
		}
		promotionPlaceholders := make([]*PromoPlaceholder, len(price.PromoPlaceholders))
		for j, promoPlaceholder := range price.PromoPlaceholders {
			parsedPromoPlaceholder := &PromoPlaceholder{
				PhId:       promoPlaceholder.PhId,
				PromoId:    int32(promoPlaceholder.PromoId),
				Type:       promoPlaceholder.Type,
				Message:    promoPlaceholder.Message,
				ProductIds: promoPlaceholder.ProductIds,
			}

			if promoPlaceholder.Promo != nil {
				parsedPromo := &Promo{
					Id:         int32(promoPlaceholder.Promo.Id),
					ExternalId: promoPlaceholder.Promo.ExternalId,
					Type:       promoPlaceholder.Promo.Type,
					Name:       promoPlaceholder.Promo.Name,
					SchemaId:   promoPlaceholder.Promo.SchemaId,
				}
				if promoPlaceholder.Promo.StartDate != nil {
					parsedPromo.StartDate = timestamppb.New(*promoPlaceholder.Promo.StartDate)
				}
				if promoPlaceholder.Promo.EndDate != nil {
					parsedPromo.EndDate = timestamppb.New(*promoPlaceholder.Promo.EndDate)
				}
				parsedPromoPlaceholder.Promo = parsedPromo
			}
			promotionPlaceholders[j] = parsedPromoPlaceholder
		}
		result[i] = &ImportModel{
			FinalPrice:       convertToProtoItem(price.FinalPrice),
			Promotions:       promotions,
			PromoPlaceholder: promotionPlaceholders,
		}
	}
	return result
}

func convertToProtoItems(prices []*(domain.BasePrice)) []*Item {
	result := make([]*Item, len(prices))
	for _, price := range prices {
		result = append(result, &Item{
			ProductId: price.ProductId,
			Price:     price.Price,
		})
	}
	return result
}

func convertToProtoItem(price *domain.FinalPrice) *Item {
	if price == nil {
		return nil
	}
	return &Item{
		ProductId: price.ProductId,
		Price:     price.Price,
	}
}
