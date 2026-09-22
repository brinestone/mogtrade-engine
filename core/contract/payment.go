package contract

type ProcessPaymentParams struct {
	PaymentMethod string
	Currency      string
	Value         float64
	ClientId      string
	Id            string
}
type PaymentProcessor interface {
	ProcessPayment(ProcessPaymentParams) error
}
