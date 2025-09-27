package asupayme

import (
	"context"
	"strconv"
	"testStand/internal/acquirer"
	"testStand/internal/acquirer/asupayme/api"
	"testStand/internal/acquirer/helper"
	"testStand/internal/models"
	"testStand/internal/repos"
)

type GatewayParams struct {
	Transport struct {
		BaseAddress string `json:"base_address"`
		TimeoutSec  *int   `json:"timeout"`
	} `json:"transport"`
}

type ChannelParams struct {
	APIKey    string `json:"api_key"`
	MerchID   string `json:"merch_id"`
	SecretKey string `json:"secret_key"`
}

type Acquirer struct {
	api           *api.Client
	db            *repos.Repo
	channelParams ChannelParams
	callbackUrl   string
}

func NewAcquirer(_ context.Context, db *repos.Repo, channelParams *ChannelParams, gatewayParams *GatewayParams) *Acquirer {

	return &Acquirer{
		api:           api.NewClient(gatewayParams.Transport.BaseAddress, channelParams.MerchID, channelParams.APIKey, channelParams.SecretKey, gatewayParams.Transport.TimeoutSec),
		db:            db,
		channelParams: *channelParams,
	}
}

func (a *Acquirer) Payout(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	req := &api.Request{
		Merchant:   a.channelParams.MerchID,
		WithdrawID: strconv.FormatInt(txn.TxnId, 10),
		CardData: api.CardData{
			OwnerName:  txn.Customer.FullName,
			CardNumber: txn.PaymentData.Object.Credentials,
		},
		Amount:    strconv.FormatInt(txn.TxnAmountSrc, 10),
		Signature: api.Sign256(a.channelParams.MerchID + txn.PaymentData.Object.Credentials + strconv.FormatInt(txn.TxnAmountSrc, 10) + a.channelParams.SecretKey),
	}

	resp, err := a.api.MakePayout(ctx, req)

	if err != nil {
		return nil, err
	}

	tr := &acquirer.TransactionStatus{}

	if resp.Status != "" && resp.Status != "success" {
		tr.Status = acquirer.REJECTED
		return tr, nil
	}

	tr.Status = acquirer.APPROVED
	tr.GtwTxnId = &resp.ID

	return tr, nil
}

func (a *Acquirer) HandleCallback(_ context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return helper.UnsupportedMethodError()
}

// заглушка
func (a *Acquirer) Payment(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return helper.UnsupportedMethodError()
}

func (a *Acquirer) FinalizePending(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return helper.UnsupportedMethodError()
}
