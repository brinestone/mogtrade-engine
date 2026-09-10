package helpers

import (
	"os"
)

func GetSystemVirtualWalletId() string {
	return os.Getenv("SYSTEM_VIRTUAL_WALLET_ID")
}

func GetSystemRealWalletId() string {
	return os.Getenv("SYSTEM_REAL_WALLET_ID")
}
