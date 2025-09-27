package api

const (
	Payment = "BUY"
	Payout  = "SELL"

	Released = "RELEASED"
	Declined = "DECLINED"
	Refunded = "REFUNDED"
	Canceled = "CANCELED"
)

type Request struct {
	FiatSymbol   string `json:"fiat_symbol"`
	FiatAmount   string `json:"fiat_amount"`
	CustomerName string `json:"customer_name"`
	CustomerAddr string `json:"customer_address,omitempty"`
	Direction    string `json:"direction"` // BUY | SELL
	GateID       string `json:"gate_id,omitempty"`
	ExternalID   string `json:"external_id,omitempty"`
	WebhookURL   string `json:"webhook_url"`
}

type Response struct {
	ID            string         `json:"_id"`
	Status        string         `json:"status"`
	ExternalID    string         `json:"external_id"`
	PaymentMethod *PaymentMethod `json:"payment_method"`
}

func (r *Response) IDPtr() *string {
	if r == nil || r.ID == "" {
		return nil
	}
	id := r.ID
	return &id
}

type PaymentMethod struct {
	ID      string `json:"_id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Person  string `json:"person"`
	Gate    struct {
		ID   string `json:"_id"`
		Name string `json:"name"`
	} `json:"gate"`
	IsTemporary bool `json:"is_temporary"`
}

type Callback struct {
	ID          string `json:"_id"`
	Status      string `json:"status"`
	ExternalID  string `json:"external_id"`
	Description string `json:"description"`
	Signature   string `json:"signature"`
}
