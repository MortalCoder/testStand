package asupayme

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"testStand/internal/acquirer"
	"testStand/internal/acquirer/asupayme/api"
	"testStand/internal/models"
	"testStand/internal/repos"
)

type GatewayParams struct {
	Transport struct {
		BaseAddress string `json:"base_address"`
		TimeoutSec  *int   `json:"timeout,omitempty"`
	} `json:"transport"`
}

type ChannelCredentials struct {
	APIKey    string `json:"api_key"`
	MerchID   string `json:"merch_id"`
	SecretKey string `json:"secret_key"`
}

type Acquirer struct {
	api   *api.Client
	db    *repos.Repo
	creds ChannelCredentials
}

func NewAcquirer(_ context.Context, db *repos.Repo, chCreds *ChannelCredentials, gw *GatewayParams) *Acquirer {
	t := 20 * time.Second
	if gw.Transport.TimeoutSec != nil && *gw.Transport.TimeoutSec > 0 {
		t = time.Duration(*gw.Transport.TimeoutSec) * time.Second
	}
	return &Acquirer{
		api:   api.NewClient(gw.Transport.BaseAddress, t),
		db:    db,
		creds: *chCreds,
	}
}

func (a *Acquirer) Payout(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	card := api.CardData{
		OwnerName:    "",
		CardNumber:   txn.PaymentData.Object.Credentials,
		ExpiredMonth: txn.PaymentData.Object.ExpMonth,
		ExpiredYear:  txn.PaymentData.Object.ExpYear,
	}
	if card.CardNumber == "" || card.ExpiredMonth == "" || card.ExpiredYear == "" {
		return &acquirer.TransactionStatus{
			Status: acquirer.REJECTED,
			Info:   map[string]string{"ps_error_code": "BAD_CARD_DATA"},
		}, nil
	}

	amount := strconv.FormatInt(txn.TxnAmountSrc, 10)
	withdrawID := fmt.Sprintf("txn-%d", txn.TxnId)

	req := &api.WithdrawRequest{
		Merchant:   a.creds.MerchID,
		WithdrawID: withdrawID,
		CardData:   card,
		Amount:     amount,
		Signature:  api.Sign256(a.creds.MerchID + card.CardNumber + amount + a.creds.SecretKey),
	}

	resp, err := a.api.MakePayout(ctx, a.creds.APIKey, req)
	if err != nil {
		return nil, err
	}

	tr := &acquirer.TransactionStatus{Status: acquirer.PENDING}

	if resp != nil && resp.ID != "" {
		tr.GtwTxnId = &resp.ID
	} else {
		wid := fmt.Sprintf("txn-%d", txn.TxnId)
		tr.GtwTxnId = &wid
	}

	return tr, nil
}

func (a *Acquirer) HandleCallback(_ context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	raw, ok := txn.TxnInfo["callback"]
	if !ok {
		return nil, errors.New("asupayme: callback body missing")
	}
	var cb api.CallbackPayload
	if err := json.Unmarshal([]byte(raw), &cb); err != nil {
		return nil, err
	}

	tr := &acquirer.TransactionStatus{
		Info: map[string]string{
			"confirmed_amount": cb.ConfirmedAmount,
			"withdraw_id":      cb.WithdrawID,
		},
	}

	switch cb.Status {
	case 9:
		tr.Status = acquirer.APPROVED
		if cb.ConfirmedAmount != "" {
			tr.Info["ps_amount"] = cb.ConfirmedAmount
		}
	case -1:
		tr.Status = acquirer.REJECTED
	default:
		tr.Status = acquirer.PENDING
	}
	return tr, nil
}

// заглушка
func (a *Acquirer) Payment(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return &acquirer.TransactionStatus{
		Status: acquirer.REJECTED,
		Info: map[string]string{
			"ps_error_code": "NOT_IMPLEMENTED",
			"ps_message":    "AsuPayme: payment is not supported",
		},
	}, nil
}

// заглушка
func (a *Acquirer) FinalizePending(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return &acquirer.TransactionStatus{
		Status: acquirer.PENDING,
		Info:   map[string]string{"ps_message": "AsuPayme: awaiting callback; finalize not supported"},
	}, nil
}
