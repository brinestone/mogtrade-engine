package httppayloads

import (
	"time"

	"github.com/brinestone/mogtrade/infra/db"
	"github.com/shopspring/decimal"
)

type GetWalletSnapshotRequest struct {
	Type db.WalletType `uri:"type" binding:"required"`
}
type WalletSnapshot struct {
	Balance           decimal.Decimal `json:"balance"`
	Id                string          `json:"walletId"`
	TotalTransactions uint            `json:"totalTransactions"`
	LastActivityAt    *time.Time      `json:"lastActivityAt"`
}

func WalletSnapshotPayload(d db.WalletSnapshot) WalletSnapshot {
	var t *time.Time
	if d.LastActivityAt.Valid {
		t = &d.LastActivityAt.Time
	}
	return WalletSnapshot{
		Balance:           d.CurrentBalance,
		Id:                d.WalletID,
		TotalTransactions: uint(d.TotalTransactions),
		LastActivityAt:    t,
	}
}
