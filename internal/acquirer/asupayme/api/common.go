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

type WithdrawRequest struct {
	Merchant   string   `json:"merchant"`
	WithdrawID string   `json:"withdraw_id"`
	CardData   CardData `json:"card_data"`
	Amount     string   `json:"amount"`
	Signature  string   `json:"signature"`
	Payload    any      `json:"payload,omitempty"`
}

type WithdrawResponse struct {
	Status string `json:"status"`
	ID     string `json:"id"`
}

type CallbackPayload struct {
	Status          int    `json:"status"`
	ConfirmedAmount string `json:"confirmed_amount"`
	WithdrawID      string `json:"withdraw_id"`
}

func Sign256(concatenated string) string {
	sum := sha256.Sum256([]byte(concatenated))
	return hex.EncodeToString(sum[:])
}
