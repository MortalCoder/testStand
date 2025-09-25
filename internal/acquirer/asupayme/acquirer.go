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
	card := api.CardData{
		OwnerName:  txn.Customer.FullName,
		CardNumber: txn.PaymentData.Object.Credentials,
	}

	req := &api.Request{
		Merchant:   a.channelParams.MerchID,
		WithdrawID: strconv.FormatInt(txn.TxnId, 10),
		CardData:   card,
		Amount:     strconv.FormatInt(txn.TxnAmountSrc, 10),
		Signature:  api.Sign256(a.channelParams.MerchID + card.CardNumber + strconv.FormatInt(txn.TxnAmountSrc, 10) + a.channelParams.SecretKey),
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

	/*
		if resp.HTTPCode < 200 || resp.HTTPCode >= 300 {
			tr.Status = acquirer.REJECTED
			tr.Info = map[string]string{
				"ps_error_code": api.FirstNonEmpty(resp.Code, strconv.Itoa(resp.HTTPCode)),
				"ps_message":    resp.Detail,
			}
			return tr, nil
		}

		tr.GtwTxnId = &resp.ID
	*/

	tr.Status = acquirer.PENDING
	tr.GtwTxnId = &resp.ID

	return tr, nil
}

func (a *Acquirer) HandleCallback(_ context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	/*
		logger := log.New("dev")

		callbackBody, ok := txn.TxnInfo["callback"]
		if !ok {
			return nil, errors.New("asupayme: callback body missing")
		}

		callback := api.Callback{}
		if err := json.Unmarshal([]byte(callbackBody), &callback); err != nil {
			logger.Error("Invalid callback - ", callbackBody)
			return nil, err
		}

		tr := &acquirer.TransactionStatus{
			Info: map[string]string{
				"confirmed_amount": callback.ConfirmedAmount,
				"withdraw_id":      callback.WithdrawID,
			},
		}

		return handleStatus(tr, callback.Status)
	*/
	return helper.UnsupportedMethodError()
}

// заглушка
func (a *Acquirer) Payment(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return helper.UnsupportedMethodError()
}

func (a *Acquirer) FinalizePending(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	/*
		return &acquirer.TransactionStatus{
			Status: acquirer.PENDING,
			Info:   map[string]string{"ps_message": "AsuPayme: finalize via callback; no status endpoint"},
		}, nil
	*/
	return helper.UnsupportedMethodError()
}

func handleStatus(tr *acquirer.TransactionStatus, status int) (*acquirer.TransactionStatus, error) {
	switch status {
	case 9:
		tr.Status = acquirer.APPROVED
		return tr, nil
	case -1:
		tr.Status = acquirer.REJECTED
		return tr, nil
	default:
		tr.Status = acquirer.PENDING
		return tr, nil
	}
}
