package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomAlphanumeric(n int) string {
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

func GenerateWalletID() string {
	return "WLT-" + randomAlphanumeric(8)
}

func GenerateTransactionRef() string {
	return fmt.Sprintf("TXN-%s-%s", time.Now().Format("20060102"), randomAlphanumeric(8))
}

func GenerateLedgerEntryID() string {
	return "LED-" + randomAlphanumeric(10)
}

func GenerateAuditLogID() string {
	return "AUD-" + randomAlphanumeric(10)
}
