package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
)

type RequestBuilder struct {
	ctx      context.Context
	baseURL  *url.URL
	method   string
	endpoint string
	query    url.Values
	headers  http.Header
	body     io.Reader
}

func (c *APIClient) NewRequest(method, endpoint string) *RequestBuilder {
	return &RequestBuilder{
		ctx:      context.Background(),
		baseURL:  c.baseURL,
		method:   method,
		endpoint: endpoint,
		query:    make(url.Values),
		headers:  make(http.Header),
	}
}

func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers.Add(key, value)
	return b
}

func (b *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	b.ctx = ctx
	return b
}

func (b *RequestBuilder) WithQueryParam(key, value string) *RequestBuilder {
	b.query.Add(key, value)
	return b
}

func (b *RequestBuilder) WithJSONBody(body any) *RequestBuilder {
	data, _ := json.Marshal(body)
	b.body = bytes.NewReader(data)
	b.headers.Set("Content-Type", "application/json")
	return b
}

func (b *RequestBuilder) Build() (*http.Request, error) {
	u := *b.baseURL
	u.Path = path.Join(u.Path, b.endpoint)
	u.RawQuery = b.query.Encode()

	req, err := http.NewRequestWithContext(b.ctx, b.method, u.String(), b.body)
	if err != nil {
		return nil, err
	}

	req.Header = b.headers
	return req, nil
}
