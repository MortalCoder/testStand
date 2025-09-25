package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testStand/internal/acquirer/helper"
)

type Client struct {
	baseAddress string
	client      *http.Client

	apiKey    string
	merchant  string
	secretKey string
}

const (
	payout = "/withdraw"
)

func NewClient(baseAddress string, merchant string, apiKey string, secretKey string, timeout *int) *Client {
	client := http.DefaultClient
	return &Client{
		baseAddress: baseAddress,
		client:      client,
		merchant:    merchant,
		apiKey:      apiKey,
		secretKey:   secretKey,
	}
}

func (c *Client) MakePayout(ctx context.Context, req *Request) (*Response, error) {
	resp := &Response{}
	err := c.makeRequest(ctx, req, resp, c.apiKey, payout)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// makeRequest
func (c *Client) makeRequest(ctx context.Context, payload any, outResponse any, apiKey string, endpoint string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, helper.JoinUrl(c.baseAddress, endpoint), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-key", apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(outResponse)
}
