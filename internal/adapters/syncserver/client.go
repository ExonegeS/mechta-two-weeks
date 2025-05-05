package syncserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
)

type EntityProvider struct {
	baseURL string
	client  *http.Client
}

func NewEntityProvider(baseURL string, timeout time.Duration) *EntityProvider {
	return &EntityProvider{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		}}
}

// Some external server interface implementation

func (p *EntityProvider) GetUserData(ttl time.Duration) (domain.Data, error) {
	data, err := p.fetchData(context.Background(), 5)
	if err != nil {
		return domain.Data{}, err
	}
	return *data, nil
}

func (p *EntityProvider) fetchData(ctx context.Context, id int) (*domain.Data, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/data/%d", p.baseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Id   int64  `json:"id"`
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &domain.Data{
		ID:    data.Id,
		Value: data.Data,
	}, nil
}
