package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseAddress string
	http        *http.Client
}

func NewClient(baseAddress string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Client{
		baseAddress: baseAddress,
		http:        &http.Client{Timeout: timeout},
	}
}

func (c *Client) MakePayout(ctx context.Context, apiKey string, req *WithdrawRequest) (*WithdrawResponse, error) {
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseAddress+"/withdraw", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Api-key", apiKey)
	httpReq.Header.Set("Merchant", req.Merchant)
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		b, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("asupayme: http %d, body=%s", httpResp.StatusCode, string(b))
	}

	var out WithdrawResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
