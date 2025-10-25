package api

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testStand/internal/acquirer/helper"
)

type Client struct {
	baseAddress string
	clientID    string
	apiName     string
	apiPassword string
	storeKey    string
	currency    string
	gatewayURL  string
	okURL       string
	failURL     string

	client *http.Client
}

func NewClient(ctx context.Context, base string, clientID, apiName, apiPassword, storeKey, currency, gatewayURL, okURL, failURL string) *Client {
	return &Client{
		baseAddress: base,
		clientID:    clientID,
		apiName:     apiName,
		apiPassword: apiPassword,
		storeKey:    storeKey,
		currency:    currency,
		gatewayURL:  gatewayURL,
		okURL:       okURL,
		failURL:     failURL,
		client:      http.DefaultClient,
	}
}

func (c *Client) MakePayment(ctx context.Context, form map[string]string) (url.Values, []byte, error) {
	data := url.Values{}
	for k, v := range form {
		data.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.gatewayURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, body, fmt.Errorf("est3Dgate non-urlencoded response: %w", err)
	}

	return vals, body, nil
}

func (c *Client) FinalizePayment(values url.Values) (string, error) {
	reqXML := Request{
		Name:     c.apiName,
		Password: c.apiPassword,
		ClientId: c.clientID,
		Type:     "Auth",
		Total:    values.Get("amount"),
		Currency: values.Get("currency"),
		OrderId:  values.Get("oid"),

		Number:                  values.Get("md"),
		PayerAuthenticationCode: values.Get("cavv"),
		PayerSecurityLevel:      values.Get("eci"),
		PayerTxnId:              values.Get("xid"),
	}

	xmlBody, err := xml.Marshal(reqXML)
	if err != nil {
		return "", err
	}
	payload := append([]byte(xml.Header), xmlBody...)

	apiURL := helper.JoinUrl(c.baseAddress, "/fim/api")
	req, _ := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var r Response
	if err := xml.Unmarshal(raw, &r); err != nil || (r.Response == "" && r.ProcReturnCode == "") {
		return string(raw), nil
	}

	return string(raw), nil
}

func BuildVer3Hash(form map[string]string, storeKey string) (string, string, []string) {
	esc := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `|`, `\|`)
		return s
	}

	keys := make([]string, 0, len(form))
	for k := range form {
		if strings.EqualFold(k, "hash") || strings.EqualFold(k, "HASH") {
			continue
		}
		if strings.EqualFold(k, "encoding") ||
			strings.EqualFold(k, "lang") {
			continue
		}
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		ki, kj := strings.ToLower(keys[i]), strings.ToLower(keys[j])
		if ki == kj {
			return keys[i] < keys[j]
		}
		return ki < kj
	})

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte("|"[0])
		}
		b.WriteString(esc(form[k]))
	}
	b.WriteByte("|"[0])
	b.WriteString(esc(storeKey))

	raw := b.String()
	sum := sha512.Sum512([]byte(raw))
	hash := base64.StdEncoding.EncodeToString(sum[:])
	return hash, raw, keys
}
