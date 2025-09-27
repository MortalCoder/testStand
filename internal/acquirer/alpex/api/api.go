package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testStand/internal/acquirer/helper"
)

type Client struct {
	baseAddress string
	email       string
	password    string
	apikey      string
	client      *http.Client
}

const (
	login = "/auth/login"
	offer = "/offer/external"
)

func NewClient(ctx context.Context, baseAddress, email, password string, timeout *int) *Client {
	c := http.DefaultClient
	return &Client{
		baseAddress: baseAddress,
		email:       email,
		password:    password,
		client:      c,
	}
}

// ensureToken — логин
func (c *Client) ensureToken(ctx context.Context) error {
	if c.apikey != "" {
		return nil
	}
	body := map[string]string{"email": c.email, "password": c.password}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, helper.JoinUrl(c.baseAddress, login), bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("auth failed: %s", res.Status)
	}

	var outResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&outResponse); err != nil {
		return err
	}
	if outResponse.AccessToken == "" {
		return fmt.Errorf("empty access_token")
	}
	c.apikey = outResponse.AccessToken
	return nil
}

// CreateOffer - BUY|SELL
func (c *Client) CreateOffer(ctx context.Context, reqBody *Request) (*Response, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, helper.JoinUrl(c.baseAddress, offer), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apikey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alpex %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var outResponse Response
	if err := json.NewDecoder(resp.Body).Decode(&outResponse); err != nil {
		return nil, err
	}
	return &outResponse, nil
}
