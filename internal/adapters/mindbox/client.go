package mind_box

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
)

const (
	PathOperationsSync         = "operations/sync"
	StatusSuccess              = "Success"
	ProcessingStatusCalculated = "Calculated"
)

type Client struct {
	opts   *domain.OptionsSt
	client *http.Client
}

func New(opts *domain.OptionsSt) *Client {
	opts.Normalize()
	return &Client{
		opts: opts,
		client: &http.Client{
			Timeout:   opts.Timeout,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		},
	}
}

func (c *Client) GetFinalPriceInfo(ctx context.Context, reqObj *domain.ImportModelReq) ([]*domain.ImportModelRep, error) {
	if c.opts.Uri == "" {
		return nil, fmt.Errorf("c.GetFinalPriceInfo: mbox client without uri")
	}

	data := domain.SubdivisionGetInfoReq{}
	data.Encode(reqObj)

	dataRaw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	resp := &domain.HttpResponse{}

	err = c.sendHttpRequestWithRetry(
		ctx,
		&domain.HttpRequest{
			Uri:    PathOperationsSync,
			Method: http.MethodPost,
			Params: url.Values{
				"operation": []string{"Shop.GetProductInfo"},
			},
			Body: dataRaw,
		}, resp, nil, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("c.sendHttpRequestWithRetry: %w", err)
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("c.sendHttpRequestWithRetry: %d status code", resp.StatusCode)
	}

	var repObj domain.SubdivisionGetInfoRep

	if len(resp.Body) > 0 && resp.Body != nil {
		err = json.Unmarshal(resp.Body, &repObj)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal: %w", err)
		}
	}

	if repObj.Status != StatusSuccess {
		return nil, fmt.Errorf("bad status")
	}
	if repObj.ProductList.ProcessingStatus != ProcessingStatusCalculated {
		return nil, fmt.Errorf("c.GetFinalPriceInfo: Bad processing status in reply")
	}

	result := make([]*domain.ImportModelRep, 0, len(repObj.ProductList.Items))

	for _, item := range repObj.ProductList.Items {
		if item.Product.Ids.Mechtakz == "" {
			continue
		}
		result = append(result, item.Decode())
	}

	return result, nil
}

func (c *Client) GetPromotionsInfo(ctx context.Context) ([]*domain.ImportPromotionsRep, error) {
	resp, err := c.GetExportData(ctx, "EksportDejstvuyushhiePromoakcii")
	if err != nil {
		return nil, fmt.Errorf("c.GetExportData: %w", err)
	}

	result := make([]*domain.ImportPromotionsRep, len(resp.Promotions))

	for i, promo := range resp.Promotions {
		result[i] = &domain.ImportPromotionsRep{
			ExternalID: promo.Ids.ExternalID,
			Name:       promo.Name,
			SchemaID:   promo.CustomFields.ShemaV1C,
			StartDate:  c.getTimeFromString(promo.StartDateTimeUtc),
			EndDate:    c.getTimeFromString(promo.EndDateTimeUtc),
		}
	}

	return result, nil
}

func (c *Client) GetExportData(
	ctx context.Context,
	operation string,
) (*domain.PromotionsGetInfoRepSt, error) {
	const (
		retryInterval = 3 * time.Second
		retryTimeout  = 1 * time.Minute
	)

	export1Obj, err := c.sendExportRequest(ctx, operation, "")
	if err != nil {
		return nil, fmt.Errorf("c.sendExportRequest: %w", err)
	}
	if export1Obj.ExportID == "" {
		slog.Error("empty export_id in reply", "operation", operation)
		return nil, fmt.Errorf("service not available")
	}

	var (
		exportFileUrl string
		export2Obj    *domain.ExportRepSt
		startTime     = time.Now()
	)

	for {
		export2Obj, err = c.sendExportRequest(ctx, operation, export1Obj.ExportID)
		if err != nil {
			return nil, fmt.Errorf("c.sendExportRequest: %w", err)
		}

		if export2Obj.ExportResult.ProcessingStatus != "Ready" {
			if time.Since(startTime) > retryTimeout {
				return nil, fmt.Errorf("timeout")
			}

			time.Sleep(retryInterval)
			continue
		}

		if len(export2Obj.ExportResult.Urls) <= 0 {
			slog.Error(
				"empty export-file url, on 'ready status",
				"operation", operation,
				"export_id", export1Obj.ExportID)
			return nil, fmt.Errorf("service not available")
		}

		exportFileUrl = export2Obj.ExportResult.Urls[0]

		break
	}

	var (
		resp   = &domain.HttpResponse{}
		repObj = &domain.PromotionsGetInfoRepSt{}
	)

	err = New(&domain.OptionsSt{
		Timeout:       20 * time.Second,
		Uri:           exportFileUrl,
		RetryCount:    1,
		RetryInterval: 5 * time.Second,
	}).sendHttpRequestWithRetry(ctx, &domain.HttpRequest{
		Method: http.MethodGet,
	}, resp, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("New.sendHttpRequestWithRetry: %w", err)
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("New.sendHttpRequestWithRetry: %d status code", resp.StatusCode)
	}

	if len(resp.Body) > 0 && resp.Body != nil {
		err = json.Unmarshal(resp.Body, repObj)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal: %w", err)
		}
	}

	return repObj, nil
}

func (c *Client) sendExportRequest(
	ctx context.Context,
	operation, exportId string,
) (*domain.ExportRepSt, error) {
	var (
		err    error
		reqObj = make(map[string]string)
		repObj = &domain.ExportRepSt{}
	)

	if exportId != "" {
		reqObj["exportId"] = exportId
	}

	dataRaw, err := json.Marshal(reqObj)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal: %w", err)
	}

	resp := &domain.HttpResponse{}

	err = c.sendHttpRequestWithRetry(ctx, &domain.HttpRequest{
		Uri:    PathOperationsSync,
		Method: http.MethodPost,
		Params: url.Values{
			"operation": []string{operation},
		},
		Body: dataRaw,
	}, resp, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("c.sendHttpRequestWithRetry: %w", err)
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("c.sendHttpRequestWithRetry: %d status code", resp.StatusCode)
	}

	if len(resp.Body) > 0 && resp.Body != nil {
		err = json.Unmarshal(resp.Body, &repObj)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal: %w", err)
		}
	}

	return repObj, err
}

// sendHttpRequest

func (c *Client) sendHttpRequestWithRetry(
	ctx context.Context,
	request *domain.HttpRequest,
	response *domain.HttpResponse,
	retryCount *int,
	retryInterval *time.Duration,
) error {
	var (
		err       error
		rCount    = c.opts.RetryCount
		rInterval = c.opts.Timeout
	)

	if retryCount != nil {
		rCount = *retryCount
	}
	if retryInterval != nil {
		rInterval = *retryInterval
	}

	for i := rCount; i >= 0; i-- {
		response.Reset()
		err = c.sendHttpRequest(ctx, &domain.HttpRequest{
			Uri:    request.Uri,
			Method: request.Method,
			Params: request.Params,
			Body:   request.Body,
		}, response)
		if err == nil {
			if response.StatusCode > 0 {
				break
			}
		}
		if i > 0 {
			fmt.Println(i, rInterval, rCount)
			time.Sleep(rInterval)
		}
	}
	if err != nil {
		return fmt.Errorf("c.sendHttpRequest: %w", err)
	}

	return nil
}

func (c *Client) sendHttpRequest(
	ctx context.Context,
	request *domain.HttpRequest,
	response *domain.HttpResponse,
) error {
	var err error

	var destUri string
	if request.Uri != "" {
		destUri, err = url.JoinPath(c.opts.Uri, request.Uri)
		if err != nil {
			return fmt.Errorf("url.JoinPath: %w", err)
		}
	} else {
		destUri = c.opts.Uri
	}

	var req *http.Request
	if c.opts.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
		defer cancel()
		req, err = http.NewRequestWithContext(
			ctx,
			request.Method,
			destUri,
			bytes.NewReader(request.Body),
		)
	} else {
		req, err = http.NewRequest(
			request.Method,
			destUri,
			bytes.NewReader(request.Body),
		)
	}
	if err != nil {
		return fmt.Errorf("http.NewRequest: %w", err)
	}

	headers := c.opts.Headers
	if headers == nil {
		headers = http.Header{}
	}
	if request.Headers != nil {
		for k, v := range request.Headers {
			headers[k] = v
		}
	}
	if request.Body != nil &&
		len(headers.Values("Content-Type")) == 0 {
		headers["Content-Type"] = []string{"application/json"}
	}
	if len(headers.Values("Accept")) == 0 {
		headers["Accept"] = []string{"application/json"}
	}
	req.Header = headers

	params := c.opts.Params
	if params == nil {
		params = url.Values{}
	}
	if request.Params != nil {
		for k, v := range request.Params {
			params[k] = v
		}
	}
	if len(params) > 0 {
		req.URL.RawQuery = params.Encode()
	}

	rand.Seed(time.Now().UnixNano())

	// Simulate request with 80% success, 20% server error
	if rand.Intn(100) < 20 {
		// Fake successful response
		response.StatusCode = http.StatusOK
		response.Body = []byte(`{"status":"Success","productList":{"processingStatus": "Calculated"}}`)
		return nil
	}

	return fmt.Errorf("server down")
	// resp, err := c.client.Do(req)
	// if err != nil {
	// 	return fmt.Errorf("client.Do: %w", err)
	// }
	// defer resp.Body.Close()

	// response.StatusCode = resp.StatusCode
	// response.Body, err = io.ReadAll(resp.Body)
	// if err != nil {
	// 	return fmt.Errorf("io.ReadAll: %w", err)
	// }

	// return nil
}

func (c *Client) getTimeFromString(str *string) *time.Time {
	if str == nil || *str == "" {
		return nil
	}

	if !strings.HasSuffix(*str, "Z") {
		*str += "Z"
	}

	result, err := time.Parse(time.RFC3339, *str)
	if err != nil {
		slog.Error("fail to parse time", "err", err, "src", *str)
		return nil
	}

	return &result
}
