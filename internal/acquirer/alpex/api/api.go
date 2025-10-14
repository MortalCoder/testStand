package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	secret      string
}

const (
	login  = "/auth/login"
	offer  = "/offer/external"
	secret = "/user/generate-signature-key"
)

func NewClient(ctx context.Context, baseAddress, email, password string) *Client {
	c := http.DefaultClient
	return &Client{
		baseAddress: baseAddress,
		email:       email,
		password:    password,
		client:      c,
	}
}

func (c *Client) GenerateSignature(ctx context.Context) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, helper.JoinUrl(c.baseAddress, secret), http.NoBody)
	req.Header.Set("Authorization", "Bearer "+c.apikey)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	key := strings.TrimSpace(out["signature_key"])
	if key == "" {
		return fmt.Errorf("alpex: empty signature_key")
	}
	c.secret = key
	return nil
}

func (c *Client) GetSignature() string {
	return c.secret
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

	var out map[string]string
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return err
	}
	key := out["access_token"]
	if key == "" {
		return fmt.Errorf("empty access_token in response")
	}
	c.apikey = key
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

	var outResponse Response
	if err := json.NewDecoder(resp.Body).Decode(&outResponse); err != nil {
		return nil, err
	}
	return &outResponse, nil
}
