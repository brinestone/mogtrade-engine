package helpers

import (
	"os"

	"github.com/brinestone/mogtrade/infra/db"
)

func GetSystemWalletByType(t db.WalletType) string {
	if t == db.WalletTypeReal {
		return GetSystemRealWalletId()
	} else {
		return GetSystemVirtualWalletId()
	}
}
func GetSystemVirtualWalletId() string {
	return os.Getenv("SYSTEM_VIRTUAL_WALLET_ID")
}

func GetSystemRealWalletId() string {
	return os.Getenv("SYSTEM_REAL_WALLET_ID")
}
