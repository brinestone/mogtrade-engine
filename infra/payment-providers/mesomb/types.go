package mesomb

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type DepositParams struct {
	Amount           decimal.Decimal `json:"amount"`
	Service          ServiceType     `json:"service"`
	Country          string          `json:"country"`
	Customer         CustomerRef     `json:"customer,omitempty"`
	TransactionId    string          `json:"trxID"`
	Currency         string          `json:"currency"`
	LocalPhoneNumber string          `json:"payer"`
	ConvertValue     bool            `json:"conversion"`
	Note             string          `json:"message,omitempty"`
}

type CustomerRef struct {
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Town      string `json:"town,omitempty"`
	Region    string `json:"region,omitempty"`
	Country   string `json:"country,omitempty"`
	Address   string `json:"address,omitempty"`
}

type collectFundsParams struct {
	TransactionId string
	Date          time.Time
	Nonce         string
	Body          any
	Mode          string
	Timeout       time.Duration
	Language      string
}

type ServiceType string

// Scan implements the [Scanner] interface
func (m *ServiceType) Scan(raw any) error {
	switch s := raw.(type) {
	case []byte:
		*m = ServiceType(s)
	case string:
		*m = ServiceType(s)
	default:
		return fmt.Errorf("unsupported scan type for mesombService: %T", raw)
	}
	return nil
}

func (m ServiceType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", m)), nil
}

const (
	baseUrl   string = "https://mesomb.hachther.com/api/1.1"
	algorithm string = "HMAC-SHA1"

	MesombServiceMtn    ServiceType = "MTN"
	MesombServiceAirtel ServiceType = "AIRTEL"
	MesombServiceOrange ServiceType = "ORGANGE"

	mesombTransactionStatusSuccess mesombTransactionStatus = "SUCCESS"
	mesombTransactionStatusFailed  mesombTransactionStatus = "FAILED"
	mesombTransactionStatusPending mesombTransactionStatus = "PENDING"
)

var (
	ErrMesombRequestFailed error = errors.New("mesomb request failed")
)

type MesombKeys struct {
	Secret string `json:"secretKey"`
	Access string `json:"accessKey"`
}

func (m *mesombTransactionStatus) Scan(src any) error {
	switch s := src.(type) {
	case []byte:
		*m = mesombTransactionStatus(s)
	case string:
		*m = mesombTransactionStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for mesombTransactionStatus: %T", src)
	}
	return nil
}

type mesombTransactionStatus string

type collectPayload struct {
	Nonce           string      `json:"nonce"`
	Mode            string      `json:"mode"`
	Amount          float32     `json:"amount"`
	Payer           string      `json:"payer"`
	Service         ServiceType `json:"service"`
	Channel         string      `json:"channel"`
	Conversion      bool        `json:"conversion"`
	Country         string      `json:"country"`
	Currency        string      `json:"currency"`
	CollectExtraFee bool        `json:"fees"`
	Message         string      `json:"message,omitempty"`
}

type mesombTimestamp time.Time

func (m mesombTimestamp) MarshalJSON() ([]byte, error) {
	stmt := fmt.Sprintf("\"%s\"", time.Time(m).Format("2006-01-02T15:04:05.999Z07:00"))
	return []byte(stmt), nil
}
func (m *mesombTimestamp) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02T15:04:05.999Z07:00", s)
	if err != nil {
		return err
	}
	*m = mesombTimestamp(t)
	return nil
}
func (m mesombTimestamp) String() string {
	return time.Time(m).Format("2006-01-02T15:04:05.999Z07:00")
}

type MesombCustomer struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Town      string `json:"town"`
	Region    string `json:"region"`
	Country   string `json:"country"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
}

type mesombTransaction struct {
	Pk        string          `json:"pk"`
	Status    string          `json:"status"`
	Type      string          `json:"type"`
	Amount    int             `json:"amount"`
	Fees      int             `json:"fees"`
	BParty    string          `json:"b_party"`
	Message   string          `json:"message"`
	Service   string          `json:"service"`
	Reference string          `json:"reference"`
	Ts        mesombTimestamp `json:"ts"`
	Direction int             `json:"direction"`
	Country   string          `json:"country"`
	Currency  string          `json:"currency"`
	Customer  MesombCustomer  `json:"customer"`
	Products  []mesombProduct `json:"products,omitzero"`
	Location  mesombLocation  `json:"location,omitzero"`
	Trxamount int             `json:"trxamount"`
	FinTrxID  string          `json:"fin_trx_id,omitzero"`
	Name      string          `json:"name"`
}

type mesombCollectResponse struct {
	Message     string            `json:"message"`
	Success     bool              `json:"success"`
	Redirect    string            `json:"redirect"`
	Transaction mesombTransaction `json:"transaction"`
	Status      string            `json:"status"`
}
type mesombLocation struct {
	Town    string `json:"town"`
	Region  string `json:"region"`
	Country string `json:"country"`
}

type mesombProduct struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Quantity int    `json:"quantity"`
	Amount   string `json:"amount"`
}

type mesombErrorResponse struct {
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

type signRequestParams struct {
	Algorithm   string            `json:"algorithm"`
	Service     string            `json:"service"`
	Method      string            `json:"method"`
	Url         string            `json:"url"`
	Date        mesombTimestamp   `json:"date"`
	Nonce       string            `json:"nonce"`
	Credentials MesombKeys        `json:"credentials"`
	Headers     map[string]string `json:"headers"`
	Body        any               `json:"body"`
}

type canonicalRequestData struct {
	Timestamp        mesombTimestamp   `json:"timestamp"`
	Query            string            `json:"canonicalQuery"`
	Path             string            `json:"path"`
	HostHeader       string            `json:"hostHeader"`
	Headers          map[string]string `json:"headers"`
	HeaderKeys       []string          `json:"headersKeys"`
	CanonicalHeaders string            `json:"canonicalHeaders"`
	SignedHeaders    string            `json:"signedHeaders"`
	Scope            string            `json:"scope"`
}

type parsedUrl struct {
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Path     string `json:"pathname"`
	Query    string `json:"query"`
}
