package orders

import (
	"log/slog"

	"github.com/brinestone/mogtrade/infra/db"
)

type RiskEngine struct {
	Logger  *slog.Logger
	Queries *db.Queries
}

func NewRiskEngine(l *slog.Logger, q *db.Queries) *RiskEngine {
	return &RiskEngine{Logger: l, Queries: q}
}
