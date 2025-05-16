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

	pb "github.com/ExonegeS/mechta-two-weeks/pkg/grpc"
)

type MindboxServer struct {
	pb.UnimplementedMindboxServiceServer
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
	pb.RegisterMindboxServiceServer(grpcServer, invServer)
	reflection.Register(grpcServer)

	logger.Info("gRPC server listening", "port", grpcPort)
	return grpcServer.Serve(lis)
}

func (s *MindboxServer) GetFinalPriceInfo(ctx context.Context, req *pb.GetFinalPriceInfoRequest) (*pb.GetFinalPriceInfoResponse, error) {
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

	return &pb.GetFinalPriceInfoResponse{
		Id:              req.GetId(),
		TotalProcessed:  int32(len(processed)),
		TotalFailed:     int32(len(failed)),
		ProcessDuration: time.Since(start).String(),
		Processed:       convertToProtoImportModels(processed),
		Failed:          convertToProtoItems(failed),
	}, nil
}

func convertToProtoImportModels(prices []*domain.ImportModelRep) []*pb.ImportModel {
	result := make([]*pb.ImportModel, len(prices))
	for i, price := range prices {
		promotions := make([]*pb.Promo, len(price.Promotions))
		for j, promo := range price.Promotions {
			parsedPromo := &pb.Promo{
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
		promotionPlaceholders := make([]*pb.PromoPlaceholder, len(price.PromoPlaceholders))
		for j, promoPlaceholder := range price.PromoPlaceholders {
			parsedPromoPlaceholder := &pb.PromoPlaceholder{
				PhId:       promoPlaceholder.PhId,
				PromoId:    int32(promoPlaceholder.PromoId),
				Type:       promoPlaceholder.Type,
				Message:    promoPlaceholder.Message,
				ProductIds: promoPlaceholder.ProductIds,
			}

			if promoPlaceholder.Promo != nil {
				parsedPromo := &pb.Promo{
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
		result[i] = &pb.ImportModel{
			FinalPrice:       convertToProtoItem(price.FinalPrice),
			Promotions:       promotions,
			PromoPlaceholder: promotionPlaceholders,
		}
	}
	return result
}

func convertToProtoItems(prices []*(domain.BasePrice)) []*pb.Item {
	result := make([]*pb.Item, len(prices))
	for _, price := range prices {
		result = append(result, &pb.Item{
			ProductId: price.ProductId,
			Price:     price.Price,
		})
	}
	return result
}

func convertToProtoItem(price *domain.FinalPrice) *pb.Item {
	if price == nil {
		return nil
	}
	return &pb.Item{
		ProductId: price.ProductId,
		Price:     price.Price,
	}
}

func (s *MindboxServer) GetPromotionsInfo(ctx context.Context, req *pb.Empty) (*pb.GetPromoInfoResponse, error) {
	s.logger.Info("GetPromotionsInfo called")

	start := time.Now()
	data, err := s.service.GetPromotionsInfo(ctx)
	if err != nil {
		s.logger.Error("Error processing request", "error", err)
		return nil, fmt.Errorf("failed to get final price info: %w", err)
	}

	return &pb.GetPromoInfoResponse{
		TotalPromotions: int32(len(data)),
		ProcessDuration: time.Since(start).String(),
		Promotions:      convertToProtoPromotions(data),
	}, nil
}

func convertToProtoPromotions(promotions []*domain.ImportPromotionsRep) []*pb.Promo {
	result := make([]*pb.Promo, len(promotions))
	for i, promo := range promotions {
		result[i] = &pb.Promo{
			ExternalId: promo.ExternalID,
			Name:       promo.Name,
			SchemaId:   promo.SchemaID,
		}
		if promo.StartDate != nil {
			result[i].StartDate = timestamppb.New(*promo.StartDate)
		}
		if promo.EndDate != nil {
			result[i].EndDate = timestamppb.New(*promo.EndDate)
		}
	}
	return result
}
