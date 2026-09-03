package orders

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

var (
	acceptedRiskCheck RiskCheckResult = RiskCheckResult{Approved: true}
)

type RiskCheckResult struct {
	Approved bool
	Reason   string
}

type OrderContext struct {
	Symbol          string
	Side            string
	OrderType       string
	Quantity        decimal.Decimal
	LimitPrice      decimal.NullDecimal
	StopPrice       decimal.NullDecimal
	AccountBalance  decimal.Decimal
	CurrentAsk      decimal.Decimal
	CurrentPosition decimal.Decimal
}

type RiskCheck interface {
	Name() string
	Evaluate(ctx context.Context, o OrderContext) RiskCheckResult
}

type RiskEngine struct {
	Logger *slog.Logger
}

func (re *RiskEngine) ValidateOrder(ctx context.Context, o OrderContext, checks ...RiskCheck) RiskCheckResult {
	re.Logger.Info("validating order", "sym", o.Symbol, "side", o.Side, "type", o.OrderType, "qty", o.Quantity)
	for _, check := range checks {
		re.Logger.Debug("evaluating check", "check", check.Name())
		result := check.Evaluate(ctx, o)
		if !result.Approved {
			re.Logger.Warn("order not approved", "sym", o.Symbol, "side", o.Side, "type", o.OrderType, "qty", o.Quantity)
			return result
		}
	}
	re.Logger.Info("order validated", "sym", o.Symbol, "side", o.Side, "type", o.OrderType, "qty", o.Quantity)
	return acceptedRiskCheck
}

func NewRiskEngine(l *slog.Logger) *RiskEngine {
	return &RiskEngine{Logger: l.With("service", "risk-engine")}
}

type velocityCheck struct {
	maxQuantity decimal.Decimal
	maxNotional decimal.Decimal
}

func (v *velocityCheck) Name() string {
	return "velocity"
}

func (v *velocityCheck) Evaluate(ctx context.Context, o OrderContext) RiskCheckResult {
	if o.Quantity.GreaterThan(v.maxQuantity) {
		return RiskCheckResult{Approved: false, Reason: "order quantity exceeds maximum allowed limit"}
	}

	var estimatedNotional decimal.Decimal
	var ot db.OrderType
	err := ot.Scan(o.OrderType)
	if err != nil {
		return RiskCheckResult{Approved: false, Reason: fmt.Sprintf("invalid order type: %s", o.OrderType)}
	}
	switch ot {
	case db.OrderTypeLimit:
		estimatedNotional = o.Quantity.Mul(o.LimitPrice.Decimal)
	case db.OrderTypeMarket:
		estimatedNotional = o.Quantity.Mul(o.CurrentAsk)
	}

	if estimatedNotional.GreaterThan(v.maxNotional) {
		return RiskCheckResult{Approved: false, Reason: "order notional exceeds maximum threshold"}
	}
	return acceptedRiskCheck
}

func WithVelocityChecking(maxQ decimal.Decimal, maxNotional decimal.Decimal) RiskCheck {
	return &velocityCheck{
		maxQuantity: maxQ,
		maxNotional: maxNotional,
	}
}

type marginCheck struct{}

func (m *marginCheck) Name() string {
	return "margin"
}

func (m *marginCheck) Evaluate(ctx context.Context, o OrderContext) RiskCheckResult {
	var side db.OrderSide
	if err := side.Scan(o.Side); err != nil {
		return RiskCheckResult{Approved: false, Reason: fmt.Sprintf("invalid order side: %s", o.Side)}
	}
	var ot db.OrderType
	if err := ot.Scan(o.OrderType); err != nil {
		return RiskCheckResult{Approved: false, Reason: fmt.Sprintf("invalid order type: %s", o.OrderType)}
	}
	if side != db.OrderSideBuy {
		return acceptedRiskCheck
	}

	var requiredCapital decimal.Decimal
	if ot == db.OrderTypeLimit && o.LimitPrice.Valid {
		requiredCapital = o.Quantity.Mul(o.LimitPrice.Decimal)
	} else if ot == db.OrderTypeMarket {
		requiredCapital = o.Quantity.Mul(o.CurrentAsk)
	}

	if requiredCapital.GreaterThan(o.AccountBalance) {
		return RiskCheckResult{false, "insufficient funds"}
	}
	return acceptedRiskCheck
}

func WithMarginChecking() RiskCheck {
	return &marginCheck{}
}

type positionCheck struct {
	maxPositions       map[string]decimal.Decimal
	newPositionDefault decimal.Decimal
}

func (p *positionCheck) Name() string {
	return "position"
}

func (p *positionCheck) Evaluate(ctx context.Context, o OrderContext) RiskCheckResult {
	maxAllowed, exists := p.maxPositions[o.Symbol]
	if !exists {
		maxAllowed = p.newPositionDefault
	}
	orderQty := o.Quantity
	var side db.OrderSide
	if err := side.Scan(o.Side); err != nil {
		return RiskCheckResult{Approved: false, Reason: fmt.Sprintf("invalid order side: %s", o.Side)}
	}

	if side == db.OrderSideSell {
		orderQty = orderQty.Neg()
	}
	projectedPosition := o.CurrentPosition.Add(orderQty)

	if projectedPosition.Abs().GreaterThan(maxAllowed) {
		return RiskCheckResult{false, fmt.Sprintf("order would exceed maximum allowed position limit for %s", o.Symbol)}
	}
	return acceptedRiskCheck
}

// WithPositionChecking creates a position checker with a map of the max positions of asset symbols
// [maxPositions] is a map containing the max positions of symbols
// [newDefault] is a value to use if there's no default for the symbol being checked.
func WithPositionChecking(maxPositions map[string]decimal.Decimal, newDefault decimal.Decimal) RiskCheck {
	return &positionCheck{maxPositions: maxPositions, newPositionDefault: newDefault}
}
