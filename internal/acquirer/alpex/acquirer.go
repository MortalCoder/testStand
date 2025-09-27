package alpex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/gommon/log"
	"strconv"
	"strings"
	"testStand/internal/acquirer"
	"testStand/internal/acquirer/alpex/api"
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
	Email    string `json:"email"`
	Password string `json:"password"`
	GateId   string `json:"gate_id"`
}

type Acquirer struct {
	api           *api.Client
	dbClient      *repos.Repo
	channelParams *ChannelParams
	callbackUrl   string
}

// NewAcquirer
func NewAcquirer(ctx context.Context, db *repos.Repo, channelParams *ChannelParams, gatewayParams *GatewayParams, callbackUrl string) *Acquirer {
	return &Acquirer{
		channelParams: channelParams,
		api:           api.NewClient(ctx, gatewayParams.Transport.BaseAddress, channelParams.Email, channelParams.Password, gatewayParams.Transport.Timeout),
		dbClient:      db,
		callbackUrl:   callbackUrl,
	}
}

// Payment — direction=BUY
func (a *Acquirer) Payment(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	request := &api.Request{
		FiatSymbol:   strings.ToUpper(txn.TxnCurrencySrc),
		FiatAmount:   strconv.FormatInt(txn.TxnAmountSrc, 10),
		CustomerName: txn.Customer.FullName,
		Direction:    api.Payment,
		ExternalID:   fmt.Sprintf("%d", txn.TxnId),
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
	if response.PaymentMethod != nil {
		tr.Outputs = map[string]string{
			"credentials": response.PaymentMethod.Address,
			"bank":        response.PaymentMethod.Gate.Name,
			"description": response.PaymentMethod.Name,
		}
	}
	return tr, nil
}

// Payout — direction=SELL
func (a *Acquirer) Payout(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	request := &api.Request{
		FiatSymbol:   strings.ToUpper(txn.TxnCurrencySrc),
		FiatAmount:   strconv.FormatInt(txn.TxnAmountSrc, 10),
		CustomerName: txn.Customer.FullName,
		CustomerAddr: txn.Customer.Address,
		Direction:    api.Payout,
		ExternalID:   fmt.Sprintf("%d", txn.TxnId),
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
