package api

import (
	"crypto/sha256"
	"encoding/hex"
)

type CardData struct {
	OwnerName    string `json:"owner_name"`
	CardNumber   string `json:"card_number"`
	ExpiredMonth string `json:"expired_month"`
	ExpiredYear  string `json:"expired_year"`
}

type Request struct {
	Merchant   string   `json:"merchant"`
	WithdrawID string   `json:"withdraw_id"`
	CardData   CardData `json:"card_data"`
	Amount     string   `json:"amount"`
	Signature  string   `json:"signature"`
	Payload    any      `json:"payload,omitempty"`
}

type Response struct {
	Status   string           `json:"status"`
	ID       string           `json:"id"`
	Detail   string           `json:"detail,omitempty"`
	Code     string           `json:"code,omitempty"`
	Messages []map[string]any `json:"messages,omitempty"`
	HTTPCode int              `json:"-"`
	RawBody  string           `json:"-"`
}

type Callback struct {
	Status          int    `json:"status"`
	ConfirmedAmount string `json:"confirmed_amount"`
	WithdrawID      string `json:"withdraw_id"`
}

type StatusRequest struct {
	Id      string `json:"id"`
	MerchId string `json:"merch_id"`
	UserRef string `json:"user_ref,omitempty"`
}

func Sign256(concatenated string) string {
	sum := sha256.Sum256([]byte(concatenated))
	return hex.EncodeToString(sum[:])
}
