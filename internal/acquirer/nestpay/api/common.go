package api

import "encoding/xml"

const (
	Released = "RELEASED"
	Declined = "DECLINED"
	Refunded = "REFUNDED"
	Canceled = "CANCELED"
)

type Request struct {
	XMLName  xml.Name `xml:"CC5Request"`
	Name     string   `xml:"Name"`
	Password string   `xml:"Password"`
	ClientId string   `xml:"ClientId"`
	Type     string   `xml:"Type"`
	Total    string   `xml:"Total"`
	Currency string   `xml:"Currency"`
	OrderId  string   `xml:"OrderId,omitempty"`

	Number                  string `xml:"Number,omitempty"`                  // md
	PayerSecurityLevel      string `xml:"PayerSecurityLevel,omitempty"`      // eci
	PayerAuthenticationCode string `xml:"PayerAuthenticationCode,omitempty"` // cavv
	PayerTxnId              string `xml:"PayerTxnId,omitempty"`              // xid
}

type Response struct {
	XMLName        xml.Name `xml:"CC5Response"`
	Response       string   `xml:"Response"`
	ProcReturnCode string   `xml:"ProcReturnCode"`
	ErrMsg         string   `xml:"ErrMsg"`
	AuthCode       string   `xml:"AuthCode"`
	HostRefNum     string   `xml:"HostRefNum"`
	TransId        string   `xml:"TransId"`
}
