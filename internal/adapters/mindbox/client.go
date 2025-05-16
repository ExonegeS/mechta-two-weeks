package mindbox

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
	"github.com/ExonegeS/mechta-two-weeks/internal/utils"
	"github.com/ExonegeS/mechta-two-weeks/pkg/httpclient"
)

const (
	PathOperationsSync         = "operations/sync"
	StatusSuccess              = "Success"
	ProcessingStatusCalculated = "Calculated"
)

type ConfigSt struct {
	Timeout time.Duration
	Uri     string

	RetryCount         int
	RetryInterval      time.Duration
	InsecureSkipVerify bool

	MaxRetries    int
	ResetDuration time.Duration
	SECRET_KEY    string
}
type Client struct {
	apiClient *httpclient.APIClient
	config    *ConfigSt
}

func New(cfg *ConfigSt) (*Client, error) {
	opts := &httpclient.OptionsSt{
		Timeout: cfg.Timeout,

		RetryCount:    cfg.RetryCount,
		RetryInterval: cfg.RetryInterval,

		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}
	opts.Normalize()
	cb := httpclient.NewCircuitBreaker(
		cfg.MaxRetries,
		cfg.ResetDuration,
	)
	apiClient, err := httpclient.NewAPIClient(cfg.Uri, opts, cb)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return &Client{
		apiClient: apiClient,
		config:    cfg,
	}, nil
}

func (c *Client) GetFinalPriceInfo(ctx context.Context, reqObj *domain.ImportModelReq) ([]*domain.ImportModelRep, error) {
	data := domain.SubdivisionGetInfoReq{}
	data.Encode(reqObj)
	req, err := c.apiClient.NewRequest(http.MethodPost, PathOperationsSync).
		WithQueryParam("operation", "Shop.GetProductInfo").
		WithQueryParam("endpointId", "MECHTA").
		WithJSONBody(data).
		WithContext(ctx).
		WithHeader("Authorization", fmt.Sprintf("Mindbox secretKey=\"%s\"", c.config.SECRET_KEY)).
		Build()
	if err != nil {
		return nil, err
	}

	var repObj domain.SubdivisionGetInfoRep
	if err := c.apiClient.Execute(ctx, req, &repObj); err != nil {
		return nil, fmt.Errorf("failed to get price info: %w", err)
	}

	if repObj.Status != StatusSuccess {
		return nil, errors.New("bad status")
	}

	if repObj.ProductList.ProcessingStatus != ProcessingStatusCalculated {
		return nil, errors.New("invalid processing status")
	}

	return processProductItems(repObj.ProductList.Items), nil
}

func processProductItems(items []*domain.SubdivisionGetInfoRepItem) []*domain.ImportModelRep {
	result := make([]*domain.ImportModelRep, 0, len(items))
	for _, item := range items {
		if item.Product.Ids.Mechtakz != "" {
			result = append(result, item.Decode())
		}
	}
	return result
}

func (c *Client) GetPromotionsInfo(ctx context.Context) ([]*domain.ImportPromotionsRep, error) {
	resp, err := c.GetExportData(ctx, "EksportDejstvuyushhiePromoakcii")
	if err != nil {
		return nil, fmt.Errorf("failed to get export data: %w", err)
	}

	return convertPromotions(resp.Promotions), nil
}

func convertPromotions(promotions []domain.PromotionSt) []*domain.ImportPromotionsRep {
	result := make([]*domain.ImportPromotionsRep, len(promotions))
	for i, promo := range promotions {
		result[i] = &domain.ImportPromotionsRep{
			ExternalID: promo.Ids.ExternalID,
			Name:       promo.Name,
			SchemaID:   promo.CustomFields.ShemaV1C,
			StartDate:  utils.ParseTime(promo.StartDateTimeUtc),
			EndDate:    utils.ParseTime(promo.EndDateTimeUtc),
		}
	}
	return result
}

func (c *Client) GetExportData(ctx context.Context, operation string) (*domain.PromotionsGetInfoRepSt, error) {
	exportID, err := c.startExport(ctx, operation)
	if err != nil {
		return nil, err
	}

	fileURL, err := c.waitForExport(ctx, operation, exportID)
	if err != nil {
		return nil, err
	}

	return c.fetchExportData(ctx, fileURL)
}

func (c *Client) startExport(ctx context.Context, operation string) (string, error) {
	req, err := c.apiClient.NewRequest(http.MethodPost, PathOperationsSync).
		WithQueryParam("operation", operation).
		WithContext(ctx).
		WithJSONBody(nil).
		Build()
	if err != nil {
		return "", err
	}

	var response domain.ExportRepSt
	if err := c.apiClient.Execute(ctx, req, &response); err != nil {
		return "", err
	}

	if response.ExportID == "" {
		return "", errors.New("empty export ID")
	}
	return response.ExportID, nil
}

func (c *Client) waitForExport(ctx context.Context, operation, exportID string) (string, error) {
	const timeout = 1 * time.Minute
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			fileURL, done, err := c.checkExportStatus(ctx, operation, exportID)
			if done {
				return fileURL, err
			}
			time.Sleep(3 * time.Second)
		}
	}
	return "", errors.New("export timeout")
}

func (c *Client) checkExportStatus(ctx context.Context, operation, exportID string) (string, bool, error) {
	req, err := c.apiClient.NewRequest(http.MethodPost, PathOperationsSync).
		WithQueryParam("operation", operation).
		WithJSONBody(map[string]string{"exportId": exportID}).Build()
	if err != nil {
		return "", false, err
	}

	var response domain.ExportRepSt
	if err := c.apiClient.Execute(ctx, req, &response); err != nil {
		return "", false, err
	}

	if response.ExportResult.ProcessingStatus == "Ready" {
		if len(response.ExportResult.Urls) == 0 {
			return "", true, errors.New("no export URLs")
		}
		return response.ExportResult.Urls[0], true, nil
	}
	return "", false, nil
}

func (c *Client) fetchExportData(ctx context.Context, fileURL string) (*domain.PromotionsGetInfoRepSt, error) {
	req, err := c.apiClient.NewRequest(http.MethodGet, fileURL).
		WithContext(ctx).
		Build()
	if err != nil {
		return nil, err
	}
	var result domain.PromotionsGetInfoRepSt
	if err := c.apiClient.Execute(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
