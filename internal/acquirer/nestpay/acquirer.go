package nestpay

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"testStand/internal/acquirer"
	"testStand/internal/acquirer/helper"
	"testStand/internal/acquirer/nestpay/api"
	"testStand/internal/models"
	"testStand/internal/repos"
	"time"
)

type GatewayParams struct {
	Transport struct {
		BaseAddress string `json:"base_address"`
	} `json:"transport"`
}

type ChannelParams struct {
	ClientID    string `json:"client_id"`
	ApiName     string `json:"api_name"`
	ApiPassword string `json:"api_pass"`
	StoreKey    string `json:"store_key"`
	Currency    string `json:"currency"`
	GatewayURL  string `json:"three_d_gateway"`
	OkURL       string `json:"ok_url"`
	FailURL     string `json:"fail_url"`
	CallbackURL string `json:"callback_url"`
}

type Acquirer struct {
	api           *api.Client
	dbClient      *repos.Repo
	channelParams *ChannelParams
}

func NewAcquirer(ctx context.Context, db *repos.Repo, channelParams *ChannelParams, gtw *GatewayParams) *Acquirer {
	return &Acquirer{
		channelParams: channelParams,
		api: api.NewClient(
			ctx,
			gtw.Transport.BaseAddress,
			channelParams.ClientID,
			channelParams.ApiName,
			channelParams.ApiPassword,
			channelParams.StoreKey,
			channelParams.Currency,
			channelParams.GatewayURL,
			channelParams.OkURL,
			channelParams.FailURL,
		),
		dbClient: db,
	}
}

func (a *Acquirer) Payout(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return helper.UnsupportedMethodError()
}

func (a *Acquirer) Payment(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	expMonth := txn.PaymentData.Object.ExpMonth
	expYear := txn.PaymentData.Object.ExpYear
	if len(expMonth) == 1 {
		expMonth = "0" + expMonth
	}
	if len(expYear) == 4 {
		expYear = expYear[2:]
	}
	rnd := strconv.FormatInt(time.Now().UnixNano(), 10)

	form := map[string]string{
		"clientid":      a.channelParams.ClientID,
		"oid":           strconv.FormatInt(txn.TxnId, 10),
		"amount":        fmt.Sprintf("%.2f", float64(txn.TxnAmountSrc)),
		"TranType":      "Auth",
		"rnd":           rnd,
		"storetype":     "3d_pay",
		"hashAlgorithm": "ver3",
		"encoding":      "utf-8",
		"currency":      a.channelParams.Currency,
		"okUrl":         a.channelParams.OkURL,
		"failUrl":       a.channelParams.FailURL,

		"pan":                             txn.PaymentData.Object.Credentials,
		"Ecom_Payment_Card_ExpDate_Month": expMonth,
		"Ecom_Payment_Card_ExpDate_Year":  expYear,
		"cv2":                             txn.PaymentData.Object.Cvv,
	}

	hash, _, _ := api.BuildVer3Hash(form, a.channelParams.StoreKey)
	form["hash"] = hash

	vals, raw, err := a.api.MakePayment(ctx, form)
	if err != nil {
		return &acquirer.TransactionStatus{
			Status: acquirer.REJECTED,
			Info:   map[string]string{"reason": "est3dgate_error", "raw": string(raw)},
		}, nil
	}

	mdStatus := strings.TrimSpace(vals.Get("mdStatus"))
	switch mdStatus {
	case "1", "2", "3", "4":
		// ok
	default:
		return &acquirer.TransactionStatus{
			Status: acquirer.REJECTED,
			Info: map[string]string{
				"mdStatus": mdStatus,
				"ErrMsg":   vals.Get("ErrMsg"),
				"raw":      string(raw),
			},
		}, nil
	}

	vals.Set("oid", form["oid"])
	vals.Set("amount", form["amount"])
	vals.Set("currency", form["currency"])

	rawXML, err := a.api.FinalizePayment(vals)
	if err != nil {
		return &acquirer.TransactionStatus{
			Status: acquirer.REJECTED,
			Info:   map[string]string{"reason": "finalize_error", "finalize_raw": rawXML},
		}, nil
	}
	var resp api.Response
	_ = xml.Unmarshal([]byte(rawXML), &resp)

	tr := &acquirer.TransactionStatus{
		Info: map[string]string{
			"mdStatus":       vals.Get("mdStatus"),
			"eci":            vals.Get("eci"),
			"xid":            vals.Get("xid"),
			"AuthCode":       resp.AuthCode,
			"TransId":        resp.TransId,
			"HostRefNum":     resp.HostRefNum,
			"ProcReturnCode": resp.ProcReturnCode,
			"Response":       resp.Response,
			"ErrMsg":         resp.ErrMsg,
			"finalize_raw":   rawXML,
		},
	}

	statusStr := mapFinalizeToApiStatus(&resp)
	return handleStatus(tr, statusStr)
}

func (a *Acquirer) HandleCallback(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return helper.UnsupportedMethodError()
}

func (a *Acquirer) FinalizePending(ctx context.Context, txn *models.Transaction) (*acquirer.TransactionStatus, error) {
	return helper.UnsupportedMethodError()
}

func mapFinalizeToApiStatus(r *api.Response) string {
	if strings.EqualFold(r.Response, "Approved") || strings.EqualFold(r.Response, "Approve") ||
		r.ProcReturnCode == "00" || r.ProcReturnCode == "0" || r.ProcReturnCode == "05" {
		return api.Released
	}
	return api.Declined
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
