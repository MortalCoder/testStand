package alpex

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/labstack/gommon/log"
	"strconv"
	"strings"
	"testStand/internal/acquirer"
	"testStand/internal/acquirer/alpex/api"
	"testStand/internal/acquirer/helper"
	"testStand/internal/models"
	"testStand/internal/repos"
)

type GatewayParams struct {
	Transport Transport `json:"transport"`
}

type Transport struct {
	BaseAddress string `json:"base_address"`
	Timeout     *int   `json:"timeout"`
}

type ChannelParams struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	GateId       string `json:"gate_id"`
	SignatureKey string `json:"signature_key"`
}

type Acquirer struct {
	api           *api.Client
	dbClient      *repos.Repo
	channelParams *ChannelParams
	callbackUrl   string
}

// NewAcquirer
func NewAcquirer(ctx context.Context, db *repos.Repo, channelParams *ChannelParams, gatewayParams *GatewayParams, callbackUrl string) *Acquirer {
	a := &Acquirer{
		channelParams: channelParams,
		api:           api.NewClient(ctx, gatewayParams.Transport.BaseAddress, channelParams.Email, channelParams.Password),
		dbClient:      db,
		callbackUrl:   callbackUrl,
	}

	if err := a.api.GenerateSignature(ctx); err != nil {
		log.Errorf("alpex: cannot obtain signature key at init: %v", err)
	}

	return a
}

// Payment
func (a *Acquirer) Payment(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	request := &api.Request{
		FiatSymbol:   strings.ToUpper(txn.TxnCurrencySrc),
		FiatAmount:   strconv.FormatInt(txn.TxnAmountSrc, 10),
		CustomerName: txn.Customer.FullName,
		Direction:    api.Payment,
		ExternalID:   strconv.FormatInt(txn.TxnId, 10),
		GateID:       a.channelParams.GateId,
		WebhookURL:   a.callbackUrl,
	}

	response, err := a.api.CreateOffer(ctx, request)
	if err != nil {
		return nil, err
	}

	tr := &acquirer.TransactionStatus{
		GtwTxnId: response.IDPtr(),
		Status:   acquirer.PENDING,
	}

	if response.Error != "" {
		tr.Info = map[string]string{"ps_error_message": response.Error}
	}

	if response.ApproveCode != "" {
		if tr.Info == nil {
			tr.Info = map[string]string{}
		}
		tr.Info["ps_approve_code"] = response.ApproveCode
	}

	if response.PaymentMethod != nil {
		tr.Outputs = map[string]string{
			"credentials": response.PaymentMethod.Address,
			"bank":        response.PaymentMethod.Gate.Name,
			"description": response.PaymentMethod.Name,
		}
	}
	return tr, nil
}

// Payout
func (a *Acquirer) Payout(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	request := &api.Request{
		FiatSymbol:   strings.ToUpper(txn.TxnCurrencySrc),
		FiatAmount:   strconv.FormatInt(txn.TxnAmountSrc, 10),
		CustomerName: txn.Customer.FullName,
		CustomerAddr: txn.Customer.Address,
		Direction:    api.Payout,
		ExternalID:   strconv.FormatInt(txn.TxnId, 10),
		GateID:       a.channelParams.GateId,
		WebhookURL:   a.callbackUrl,
	}

	response, err := a.api.CreateOffer(ctx, request)
	if err != nil {
		return nil, err
	}

	tr := &acquirer.TransactionStatus{
		GtwTxnId: response.IDPtr(),
		Status:   acquirer.PENDING,
	}

	if response.Error != "" {
		tr.Info = map[string]string{"ps_error_message": response.Error}
	}
	if response.ApproveCode != "" {
		if tr.Info == nil {
			tr.Info = map[string]string{}
		}
		tr.Info["ps_approve_code"] = response.ApproveCode
	}

	if response.PaymentMethod != nil {
		tr.Outputs = map[string]string{
			"credentials": response.PaymentMethod.Address,
			"bank":        response.PaymentMethod.Gate.Name,
			"description": response.PaymentMethod.Name,
		}
	}
	return tr, nil
}

// HandleCallback
func (a *Acquirer) HandleCallback(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	logger := log.New("dev")

	callbackBody, ok := txn.TxnInfo["callback"]
	if !ok {
		return nil, errors.New("callback body is missing")
	}

	callback := api.Callback{}
	if err := json.Unmarshal([]byte(callbackBody), &callback); err != nil {
		logger.Error("Error unmarshalling callback: ", callbackBody)
		return nil, err
	}

	key := strings.TrimSpace(a.api.GetSignature())
	if key == "" {
		return nil, errors.New("alpex: signature key is missing")
	}

	expected := helper.GenerateHMAC(sha256.New, []byte("id="+callback.ID+"\nstatus="+callback.Status), key)

	got, err := hex.DecodeString(callback.Signature)
	if err != nil || !hmac.Equal(got, expected) {
		return nil, errors.New("invalid signature")
	}

	tr := &acquirer.TransactionStatus{}
	if callback.Description != "" {
		tr.Info = map[string]string{"ps_error_code": callback.Description}
	}
	return handleStatus(tr, callback.Status)
}

// FinalizePending
func (a *Acquirer) FinalizePending(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return &acquirer.TransactionStatus{Status: acquirer.PENDING}, nil
}

func handleStatus(tr *acquirer.TransactionStatus, status string) (*acquirer.TransactionStatus, error) {
	switch strings.ToUpper(status) {
	case api.Released:
		tr.Status = acquirer.APPROVED
		return tr, nil
	case api.Declined, api.Refunded, api.Canceled:
		tr.Status = acquirer.REJECTED
		return tr, nil
	default:
		tr.Status = acquirer.PENDING
		return tr, nil
	}
}
